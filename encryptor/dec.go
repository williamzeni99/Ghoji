package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// This function decrypt a buffer with a 32 byte key. The 'encBuffer'
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

// This method decrypt a file with the AES256 with GCM. It split the file in different chunks
// and decrypt all of them in parallel. If you change the chunkSize between encryption and decryption
// it will not work.
// You can set the number of physical cores to use with 'numCpu', default Max.
// You can also set the max number of 'goroutines' going in parallel (one chunk one goroutine), default 1000.
// With 'progress' you can get the advancement updates as a fraction (number between 0 and 1) of the decrypted chunks over all the chunks.
// IMPORTANT: To decrypt the file you need at least one time the file size free in the hard drive memory. Remember that for each chunk of
// 1MB you shrink the file of 28 bytes. In addition, the decrypted chunks are stored in a new file and the previous one is then deleted.
func DecryptFile(password string, encfilePath string, numCpu int, goroutines int, progress chan<- float64) error {
	//check parameters
	if numCpu > MaxCPUs || numCpu < 0 {
		numCpu = MaxCPUs
	}

	if goroutines <= 0 {
		goroutines = DefaultGoRoutines
	}

	//setting max cpu usage
	runtime.GOMAXPROCS(numCpu)

	//generating the key (nice way to be sure to have 32 bytes password? I'm not sure)
	key := sha256.Sum256([]byte(password))

	//file opening
	encFile, err := os.Open(encfilePath)
	if err != nil {
		return fmt.Errorf("\ncannot open %s\n%s", encfilePath, err)
	}
	defer encFile.Close()

	if filepath.Ext(encfilePath) != encExt {
		return fmt.Errorf("\nthis file is not a %s file. Decrypt failed", encExt)
	}

	encFileName := filepath.Base(encfilePath)
	filename := encFileName[:len(encFileName)-len(encExt)]
	filePath := filepath.Join(filepath.Dir(encfilePath), filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("\ncannot create %s\n%s", filePath, err)
	}
	defer file.Close()

	//setting up the chunks
	encfileInfo, err := encFile.Stat()
	if err != nil {
		return err
	}

	numChunks := int(int(encfileInfo.Size()) / enc_chunkSize)
	lastChunksize := int(encfileInfo.Size()) % enc_chunkSize

	//setting the parallelism
	var wg sync.WaitGroup
	wg.Add(numChunks) //one for the progress bar
	if lastChunksize > 0 {
		wg.Add(1)
	}

	maxGoroutinesChannel := make(chan struct{}, goroutines)

	//progress bar
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
			buffer := make([]byte, enc_chunkSize)
			_, err := encFile.ReadAt(buffer, int64(readOffset))
			if err != nil && err != io.EOF {
				x := fmt.Errorf("\nchunk failed reading --> read offset: %d", readOffset)
				panic(x)
			}

			data, err := decryptBuffer(key, buffer)
			if err != nil {
				x := fmt.Errorf("\nchunk failed dec --> read offset: %d", readOffset)
				panic(x)
			}

			_, err = file.WriteAt(data, int64(writeOffset))
			if err != nil {
				x := fmt.Errorf("\nchunk failed writing --> read offset: %d \t write offset %d", readOffset, writeOffset)
				panic(x)
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
			_, err := encFile.ReadAt(buffer, int64(readOffset))
			if err != nil && err != io.EOF {
				x := fmt.Errorf("\nchunk failed last reading --> read offset: %d", readOffset)
				panic(x)
			}

			data, err := decryptBuffer(key, buffer)
			if err != nil {
				x := fmt.Errorf("\nchunk failed last dec --> read offset: %d", readOffset)
				panic(x)
			}

			_, err = file.WriteAt(data, int64(writeOffset))
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

	//removing file after decryption
	err = encFile.Close()
	if err != nil {
		return fmt.Errorf("\ncannot close %s\n%s", encfilePath, err)
	}
	err = os.Remove(encfilePath)
	if err != nil {
		return fmt.Errorf("\ncannot delete %s\n%s", encfilePath, err)
	}
	return nil

}
