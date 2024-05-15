package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

/*
*
This function encrypt a buffer with a 32 byte key. The resulting encrypted buffer
will be composed as follow: nonce + enc_buffer + gcmTag
So, the resulting buffer length will be 28 bytes longer.
*/
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

/*
*
 */
func EncryptFile(password string, filePath string, numCpu int, goroutines int, progress chan<- float64) error {
	//check parameters
	if numCpu > MaxCPUs || numCpu < 0 {
		return fmt.Errorf("\nthe number of cpus must be between 1 and  %d", runtime.NumCPU())
	}

	if goroutines <= 0 {
		return fmt.Errorf("\nthe number of maxgoroutines must be greater than 0")
	}

	//setting max cpu usage
	runtime.GOMAXPROCS(numCpu)

	//generating the key (nice way to be sure to have 32 bytes password? I'm not sure)
	key := sha256.Sum256([]byte(password))

	//file opening
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("\ncannot open %s\n%s", filePath, err)
	}

	filename := filepath.Base(filePath)
	encFileName := filename + encExt
	encfilePath := filepath.Join(filepath.Dir(filePath), encFileName)

	encFile, err := os.Create(encfilePath)
	if err != nil {
		return fmt.Errorf("\ncannot create %s\n%s", encFileName, err)
	}
	defer encFile.Close()

	//setting up the chunks
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	numChunks := int(int(fileInfo.Size()) / chunkSize)
	lastChunksize := int(fileInfo.Size()) % chunkSize

	//setting the parallelism
	var wg sync.WaitGroup
	wg.Add(numChunks) //one for the progress bar
	if lastChunksize > 0 {
		wg.Add(1)
	}

	maxGoroutinesChannel := make(chan struct{}, goroutines)

	// progress bar
	counter := make(chan int)

	wg.Add(1)
	go func() {
		totalPackets := numChunks
		if lastChunksize > 0 {
			totalPackets += 1
		}
		sum := 0
		progress <- 0
		for plus := range counter {
			sum += plus
			if sum == totalPackets {
				close(counter)
			}
			progress <- float64(sum) / float64(totalPackets)
		}
		close(progress)
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
			if err != nil && err != io.EOF {
				x := fmt.Errorf("\nchunk failed reading --> read offset: %d", readOffset)
				panic(x)
			}

			data, err := encryptBuffer(key, buffer)
			if err != nil {
				x := fmt.Errorf("\nchunk failed enc --> read offset: %d", readOffset)
				panic(x)
			}

			_, err = encFile.WriteAt(data, int64(writeOffset))
			if err != nil {
				x := fmt.Errorf("\nchunk failed writing --> read offset: %d \t write offset %d", readOffset, writeOffset)
				panic(x)
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
			if err != nil && err != io.EOF {
				x := fmt.Errorf("\nchunk failed last reading --> read offset: %d", readOffset)
				panic(x)
			}

			data, err := encryptBuffer(key, buffer)
			if err != nil {
				x := fmt.Errorf("\nchunk failed last enc --> read offset: %d", readOffset)
				panic(x)
			}

			_, err = encFile.WriteAt(data, int64(writeOffset))
			if err != nil {
				x := fmt.Errorf("\nchunk failed last writing --> read offset: %d \t write offset %d", readOffset, writeOffset)
				panic(x)
			}

			counter <- 1
			<-maxGoroutinesChannel
			wg.Done()

		}(currentReadOffset, currentWriteOffset)
	}

	wg.Wait()

	//removing file after encryption
	err = file.Close()
	if err != nil {
		return fmt.Errorf("\ncannot close %s\n%s", filePath, err)
	}
	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("\ncannot delete %s\n%s", filePath, err)
	}

	return nil
}
