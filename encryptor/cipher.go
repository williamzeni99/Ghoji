package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Ghojier interface {
	Rollback()
	Encrypt()
	Decrypt()
}

type GhojiFile struct {
	FilePath     string
	New_filePath string
	Password     [32]byte
	Progress     chan float32
	Faults       error
}

// This function encrypts a plain byte list with a 32 byte key. The resulting encrypted buffer
// will be composed as follow: nonce + enc_buffer + gcmTag
// So, the resulting buffer length will be 28 bytes longer.
func encryptBuffer(key [32]byte, buffer []byte) ([]byte, error) {
	c, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	data := gcm.Seal(nil, nonce, buffer, nil)

	data = append(nonce, data...)

	return data, nil
}

// This function decrypts a buffer with a 32 byte key. The 'encBuffer'
// must be composed as follow: nonce + cypherText + gcmTag
// So, the resulting buffer length will be 28 bytes less.
func decryptBuffer(key [32]byte, encBuffer []byte) ([]byte, error) {
	c, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := encBuffer[:gcm.NonceSize()]
	buffer := encBuffer[gcm.NonceSize():]

	return gcm.Open(nil, nonce, buffer, nil)
}

func (x *GhojiFile) Encrypt() {
	runtime.GOMAXPROCS(MaxCPUs)
	//file opening
	file, err := os.Open(x.FilePath)
	if err != nil {
		x.Faults = fmt.Errorf("unable to open %s\nerr:%s", x.FilePath, err)
		close(x.Progress)
		return
	}
	defer file.Close()

	filename := filepath.Base(x.FilePath)
	newFileName := filename + encExt
	x.New_filePath = filepath.Join(filepath.Dir(x.FilePath), newFileName)

	newFile, err := os.Create(x.New_filePath)
	if err != nil {
		x.Faults = fmt.Errorf("unable to create %s\nerr:%s", x.New_filePath, err)
		close(x.Progress)
		return
	}
	defer newFile.Close()

	//setting up the chunks
	fileInfo, err := file.Stat()
	if err != nil {
		x.Faults = fmt.Errorf("unable to read %s\nerr:%s", x.FilePath, err)
		close(x.Progress)
		return
	}

	numChunks := int(int(fileInfo.Size()) / chunkSize)
	lastChunksize := int(fileInfo.Size()) % chunkSize

	//setting the parallelism
	var wg sync.WaitGroup
	wg.Add(numChunks) //one for the progress bar
	if lastChunksize > 0 {
		wg.Add(1)
	}

	maxGoroutinesChannel := make(chan struct{}, DefaultGoRoutines)

	// progress bar
	counter := make(chan int)

	wg.Add(1)
	go func() {
		totalPackets := numChunks
		if lastChunksize > 0 {
			totalPackets += 1
		}
		sum := 0
		x.Progress <- 0
		for plus := range counter {
			sum += plus
			if sum == totalPackets {
				close(counter)
			}
			x.Progress <- float32(sum) / float32(totalPackets)
		}
		close(x.Progress)
		wg.Done()
	}()

	//doing the parallelism

	currentReadOffset := 0
	currentWriteOffset := 0
	for i := 0; i < numChunks; i++ {
		go func(readOffset int, writeOffset int) {
			maxGoroutinesChannel <- struct{}{}
			buffer := make([]byte, chunkSize)
			_, err := file.ReadAt(buffer, int64(readOffset))
			if err == nil || err == io.EOF {
				data, err := encryptBuffer(x.Password, buffer)
				if err == nil {
					_, err := newFile.WriteAt(data, int64(writeOffset))
					if err != nil && err != io.EOF {
						x.Faults = fmt.Errorf("something strange happened when writing at %d of file %s\nerr: %s", readOffset, x.New_filePath, err)
					}
				} else {
					x.Faults = fmt.Errorf("encryption of file %s failed\nerr: %s", x.FilePath, err)
				}

			} else {
				x.Faults = fmt.Errorf("something strange happened when reading at %d of file %s\nerr: %s", readOffset, x.FilePath, err)
			}

			counter <- 1
			<-maxGoroutinesChannel
			wg.Done()

		}(currentReadOffset, currentWriteOffset)
		currentReadOffset += chunkSize
		currentWriteOffset += enc_chunkSize
	}

	if lastChunksize > 0 {
		go func(readOffset int, writeOffset int) {
			maxGoroutinesChannel <- struct{}{}
			buffer := make([]byte, lastChunksize)
			_, err := file.ReadAt(buffer, int64(readOffset))
			if err == nil || err == io.EOF {
				data, err := encryptBuffer(x.Password, buffer)
				if err == nil {
					_, err = newFile.WriteAt(data, int64(writeOffset))
					if err != nil && err != io.EOF {
						x.Faults = fmt.Errorf("something strange happened when writing at %d of file %s\nerr: %s", readOffset, x.New_filePath, err)
					}
				} else {
					x.Faults = fmt.Errorf("encryption of file %s failed\nerr: %s", x.FilePath, err)
				}

			} else {
				x.Faults = fmt.Errorf("something strange happened when reading at %d of file %s\nerr: %s", readOffset, x.FilePath, err)
			}

			counter <- 1
			<-maxGoroutinesChannel
			wg.Done()

		}(currentReadOffset, currentWriteOffset)
	}

	wg.Wait()
}

func (x *GhojiFile) Rollback() error {
	if x.New_filePath != "" {
		return os.Remove(x.New_filePath)
	}
	return nil
}

func (x *GhojiFile) Decrypt() {
	runtime.GOMAXPROCS(MaxCPUs)
	//file opening
	file, err := os.Open(x.FilePath)
	if err != nil {
		x.Faults = fmt.Errorf("unable to open %s\nerr:%s", x.FilePath, err)
		close(x.Progress)
		return
	}
	defer file.Close()

	if filepath.Ext(x.FilePath) != encExt {
		x.Faults = fmt.Errorf("this is not a %s file. I cannot perform a decryption", encExt)
		close(x.Progress)
		return
	}

	jiFileName := filepath.Base(x.FilePath)
	filename := jiFileName[:len(jiFileName)-len(encExt)]
	x.New_filePath = filepath.Join(filepath.Dir(x.FilePath), filename)

	newFile, err := os.Create(x.New_filePath)
	if err != nil {
		x.Faults = fmt.Errorf("unable to create %s\nerr:%s", x.New_filePath, err)
		close(x.Progress)
		return
	}
	defer newFile.Close()

	//setting up the chunks
	fileInfo, err := file.Stat()
	if err != nil {
		x.Faults = fmt.Errorf("unable to read %s\nerr:%s", x.FilePath, err)
		close(x.Progress)
		return
	}

	numChunks := int(int(fileInfo.Size()) / enc_chunkSize)
	lastChunksize := int(fileInfo.Size()) % enc_chunkSize

	//setting the parallelism
	var wg sync.WaitGroup
	wg.Add(numChunks) //one for the progress bar
	if lastChunksize > 0 {
		wg.Add(1)
	}

	maxGoroutinesChannel := make(chan struct{}, DefaultGoRoutines)

	// progress bar
	counter := make(chan int)

	wg.Add(1)
	go func() {
		totalPackets := numChunks
		if lastChunksize > 0 {
			totalPackets += 1
		}
		sum := 0
		x.Progress <- 0
		for plus := range counter {
			sum += plus
			if sum == totalPackets {
				close(counter)
			}
			x.Progress <- float32(sum) / float32(totalPackets)
		}
		close(x.Progress)
		wg.Done()
	}()

	//doing the parallelism

	currentReadOffset := 0
	currentWriteOffset := 0
	for i := 0; i < numChunks; i++ {
		go func(readOffset int, writeOffset int) {
			maxGoroutinesChannel <- struct{}{}
			buffer := make([]byte, enc_chunkSize)
			_, err := file.ReadAt(buffer, int64(readOffset))
			if err == nil || err == io.EOF {
				data, err := decryptBuffer(x.Password, buffer)
				if err == nil {
					_, err := newFile.WriteAt(data, int64(writeOffset))
					if err != nil && err != io.EOF {
						x.Faults = fmt.Errorf("something strange happened when writing at %d of file %s\nerr: %s", readOffset, x.New_filePath, err)
					}
				} else {
					x.Faults = fmt.Errorf("decryption of file %s failed\nerr: %s", x.FilePath, err)
				}

			} else {
				x.Faults = fmt.Errorf("something strange happened when reading at %d of file %s\nerr: %s", readOffset, x.FilePath, err)
			}

			counter <- 1
			<-maxGoroutinesChannel
			wg.Done()

		}(currentReadOffset, currentWriteOffset)
		currentReadOffset += enc_chunkSize
		currentWriteOffset += chunkSize
	}

	if lastChunksize > 0 {
		go func(readOffset int, writeOffset int) {
			maxGoroutinesChannel <- struct{}{}
			buffer := make([]byte, lastChunksize)
			_, err := file.ReadAt(buffer, int64(readOffset))
			if err == nil || err == io.EOF {
				data, err := decryptBuffer(x.Password, buffer)
				if err == nil {
					_, err = newFile.WriteAt(data, int64(writeOffset))
					if err != nil && err != io.EOF {
						x.Faults = fmt.Errorf("something strange happened when writing at %d of file %s\nerr: %s", readOffset, x.New_filePath, err)
					}
				} else {
					x.Faults = fmt.Errorf("decryption of file %s failed\nerr: %s", x.FilePath, err)
				}

			} else {
				x.Faults = fmt.Errorf("something strange happened when reading at %d of file %s\nerr: %s", readOffset, x.FilePath, err)
			}

			counter <- 1
			<-maxGoroutinesChannel
			wg.Done()

		}(currentReadOffset, currentWriteOffset)
	}

	wg.Wait()
}
