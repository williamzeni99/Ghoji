package encryptor

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

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

func DecryptFile(pass string, encfilePath string, saveDir string, numCpu int) error {
	//check parameters
	if numCpu > runtime.NumCPU() || numCpu < 0 {
		return fmt.Errorf("the number of cpus must be between 1 and  %d", runtime.NumCPU())
	}

	//setting max cpu usage
	runtime.GOMAXPROCS(numCpu)

	//generating the key
	key := sha256.Sum256([]byte(pass))

	//open encfile
	encFile, err := os.Open(encfilePath)
	if err != nil {
		return err
	}
	defer encFile.Close()

	//get numChunks from beginning of file
	chuncksize_buff := make([]byte, numChunksBufferMax)
	_, err = encFile.ReadAt(chuncksize_buff, 0)
	if err != nil && err != io.EOF {
		return err
	}
	numChunks := int(binary.LittleEndian.Uint64(chuncksize_buff))

	//get original filename
	nonce := make([]byte, nonceSize)
	_, err = encFile.ReadAt(nonce, int64(numChunksBufferMax))
	if err != nil && err != io.EOF {
		return err
	}

	encfilenameBuffer := make([]byte, maxFilename_length+gcmTagSize)
	_, err = encFile.ReadAt(encfilenameBuffer, int64(numChunksBufferMax+nonceSize))
	if err != nil && err != io.EOF {
		return err
	}

	encfilenameBuffer = append(nonce, encfilenameBuffer...)
	filenameBuffer, err := decryptBuffer(key, encfilenameBuffer)
	if err != nil {
		return err
	}
	filename := string(bytes.Trim(filenameBuffer, "\x00"))

	//create the file
	filePath := filepath.Join(saveDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	//setting up the chunks
	encfileInfo, err := encFile.Stat()
	if err != nil {
		return err
	}

	readInitialOffset := numChunksBufferMax + len(encfilenameBuffer)
	originalFileSize := int(encfileInfo.Size()) - readInitialOffset - numChunks*(nonceSize+gcmTagSize)
	chunkSize := int(originalFileSize / numChunks)
	lastchunkSize := (originalFileSize % numChunks) + chunkSize
	lastenc_chunkSize := lastchunkSize + nonceSize + gcmTagSize
	enc_chunkSize := chunkSize + nonceSize + gcmTagSize

	// fmt.Println("Encrypted File size: ", encfileInfo.Size())
	// fmt.Println("Original File size calculated: ", originalFileSize)
	// fmt.Printf("Calculated chunks: chunkSize %d, lastchunkSize %d, enc_chunkSize %d\n", chunkSize, lastchunkSize, enc_chunkSize)

	//making the parallelism
	var wg sync.WaitGroup
	wg.Add(numChunks)

	for i := 0; i < numChunks-1; i++ {
		go func(id int, readOffset int, writeOffset int) {
			defer wg.Done()

			//fmt.Printf("chunk %d: read at %d --> writing at %d\n", id, readOffset, writeOffset)

			buffer := make([]byte, enc_chunkSize)
			_, err := encFile.ReadAt(buffer, int64(readOffset))
			if err != nil && err != io.EOF {
				x := fmt.Errorf("chunk failed reading --> %d", id)
				panic(x)
			}

			data, err := decryptBuffer(key, buffer)
			if err != nil {
				x := fmt.Errorf("chunk failed dec --> %d", id)
				panic(x)
			}

			_, err = file.WriteAt(data, int64(writeOffset))
			if err != nil {
				x := fmt.Errorf("chunk failed writing --> %d", id)
				panic(x)
			}

		}(i+1, readInitialOffset+i*enc_chunkSize, i*chunkSize)
	}

	go func(id int, readOffset int, writeOffset int) {
		defer wg.Done()

		//fmt.Printf("chunk %d: read at %d --> writing at %d\n", id, readOffset, writeOffset)

		buffer := make([]byte, lastenc_chunkSize)
		_, err := encFile.ReadAt(buffer, int64(readOffset))
		if err != nil && err != io.EOF {
			x := fmt.Errorf("chunk failed reading --> %d", id)
			panic(x)
		}

		data, err := decryptBuffer(key, buffer)
		if err != nil {
			x := fmt.Errorf("chunk failed dec --> %d", id)
			panic(x)
		}

		_, err = file.WriteAt(data, int64(writeOffset))
		if err != nil {
			x := fmt.Errorf("chunk failed wrtining --> %d", id)
			panic(x)
		}

	}(numChunks, readInitialOffset+(numChunks-1)*enc_chunkSize, (numChunks-1)*chunkSize)

	wg.Wait()

	return nil
}
