package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"ghoji/ghojierrors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

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

// This method encrypts a file with the AES256 with GCM. It split the file in different chunks
// and encrypt all of them in parallel.
// You can set the number of physical cores to use with 'numCpu', default Max.
// You can also set the max number of 'goroutines' going in parallel (one chunk one goroutine), default 1000.
// With 'progress' you can get the advancement updates as a fraction (number between 0 and 1) of the encrypted chunks over all the chunks.
// IMPORTANT: To encrypt the file you need more than one time the file size free in the hard drive memory. Remember that for each chunk of
// 1MB you gain 28 bytes. In addition, the encrypted chunks are stored in a new file and the previous one is then deleted.
func EncryptFile(password string, filePath string, numCpu int, goroutines int, progress chan<- float64, errors chan<- ghojierrors.Handable) string {
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
	file, err := os.Open(filePath)
	if err != nil {
		errors <- &ghojierrors.OpenFileError{Path: filePath, Error: err}
		return ""
	}

	filename := filepath.Base(filePath)
	encFileName := filename + encExt
	encfilePath := filepath.Join(filepath.Dir(filePath), encFileName)

	encFile, err := os.Create(encfilePath)
	if err != nil {
		errors <- &ghojierrors.CreateFileError{Path: encfilePath, Error: err}
		return ""
	}
	defer encFile.Close()

	//setting up the chunks
	fileInfo, err := file.Stat()
	if err != nil {
		errors <- &ghojierrors.InfoFileError{Path: filePath, Error: err}
		return ""
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

	//menaging one error on just one chunk
	chunksFailed := make(chan bool)
	first := true
	wg.Add(1)
	go func() {
		for boolean := range chunksFailed {
			if first && boolean {
				first = false
				errors <- &ghojierrors.FileEncryptionFailed{Path: filePath}
			}
		}
		wg.Done()
	}()

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
		close(chunksFailed)
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
				chunksFailed <- true
			} else {
				data, err := encryptBuffer(key, buffer)
				if err != nil {
					chunksFailed <- true
				} else {
					_, err = encFile.WriteAt(data, int64(writeOffset))
					if err != nil {
						chunksFailed <- true
					}
				}
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
				chunksFailed <- true
			} else {
				data, err := encryptBuffer(key, buffer)
				if err != nil {
					chunksFailed <- true
				} else {
					_, err = encFile.WriteAt(data, int64(writeOffset))
					if err != nil {
						chunksFailed <- true
					}
				}
			}

			counter <- 1
			<-maxGoroutinesChannel
			wg.Done()

		}(currentReadOffset, currentWriteOffset)
	}

	wg.Wait()

	//if there was errors
	if !first {
		return ""
	}

	//removing file after encryption
	err = file.Close()
	if err != nil {
		errors <- &ghojierrors.CloseFileError{Path: filePath, Error: err}
		return ""
	}
	err = os.Remove(filePath)
	if err != nil {
		errors <- &ghojierrors.RemoveFileError{Path: filePath, Error: err}
		return ""
	}

	return encfilePath
}

// This method encrypts a list of file with the method EncryptFile.
// You can set the number of files to encrypt in parallel.
// For numCpu, goroutines read EncryptFile
// 'progress' is a channel that recieves a 1 for each file encrypted
// IMPORTANT: high values fro maxfiles can cause a crash. Use it at your own risk
func EncryptMultipleFiles(password string, filePaths []string, numCpu int, goroutines int, progress chan<- int, maxfiles int, errors chan<- ghojierrors.Handable) {
	var wg sync.WaitGroup

	if maxfiles <= 0 {
		maxfiles = DefaultMaxFiles
	}

	maxfiles_channel := make(chan struct{}, maxfiles)
	fileProgress := make([]chan float64, len(filePaths))

	progress <- 0

	for i := range filePaths {
		fileProgress[i] = make(chan float64)
	}

	// Funzione per monitorare il progresso di ciascun file
	monitorProgress := func(index int) {
		for range fileProgress[index] {
		}
		wg.Done()
	}

	// Avvio dei goroutines per monitorare il progresso
	for i := range filePaths {
		wg.Add(1)
		go monitorProgress(i)
	}

	// Avvio dei goroutines per cifrare i file
	for i, filePath := range filePaths {
		wg.Add(1)
		go func(index int, path string) {
			maxfiles_channel <- struct{}{}
			_ = EncryptFile(password, path, numCpu, goroutines, fileProgress[index], errors)
			progress <- 1
			<-maxfiles_channel
			wg.Done()
		}(i, filePath)
	}

	wg.Wait()
	close(progress)
}
