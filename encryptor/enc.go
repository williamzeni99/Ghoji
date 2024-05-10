package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/sethvargo/go-password/password"
)

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

func EncryptFile(filePath string, saveDir string, numCpu int, numChunks int) (string, error) {

	//check parameters
	if numCpu > runtime.NumCPU() || numCpu < 0 {
		return "", fmt.Errorf("the number of cpus must be between 1 and  %d", runtime.NumCPU())
	}

	if numChunks <= 0 {
		return "", fmt.Errorf("chunk size must be between 0 and 2^64")
	}

	//setting max cpu usage
	runtime.GOMAXPROCS(numCpu)

	//generating the key
	password_length := mrand.Intn(16) + 17
	nums := mrand.Intn(password_length / 2)
	symbols := mrand.Intn(password_length / 2)
	pass, err := password.Generate(password_length, nums, symbols, false, true)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(pass))

	//file opening
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	filename := filepath.Base(filePath)

	if len(filename) > maxFilename_length {
		return "", fmt.Errorf("filename length must be less than %d characters", maxFilename_length)
	}

	shaName := sha256.Sum256([]byte(filename))
	encFileName := hex.EncodeToString(shaName[:])
	encfilePath := filepath.Join(saveDir, encFileName)

	encFile, err := os.Create(encfilePath)
	if err != nil {
		return "", err
	}
	defer encFile.Close()

	//save numChunks at the beginning of the file
	numChunks_buff := make([]byte, numChunksBufferMax)
	binary.LittleEndian.PutUint64(numChunks_buff, uint64(numChunks))
	_, err = encFile.WriteAt(numChunks_buff, 0)
	if err != nil {
		panic(err)
	}

	//saving the filename encrypted after numChunks
	filenameBuffer := make([]byte, maxFilename_length)
	copy(filenameBuffer, []byte(filename))

	enc_filenameBuffer, err := encryptBuffer(key, filenameBuffer)
	if err != nil {
		panic(err)
	}

	_, err = encFile.WriteAt(enc_filenameBuffer, int64(numChunksBufferMax))
	if err != nil {
		panic(err)
	}

	//setting up the chunks
	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	chunkSize := int(int(fileInfo.Size()) / numChunks) //I want chunksize to be integer to not have problems with buffer
	lastChunksize := (int(fileInfo.Size()) % numChunks) + chunkSize
	enc_chunkSize := chunkSize + nonceSize + gcmTagSize
	initialWriteOffset := len(numChunks_buff) + len(enc_filenameBuffer)
	// fmt.Println("File size: ", fileInfo.Size())
	// fmt.Printf("Calculated chunks: chunkSize %d, lastChunksize %d, enc_chunkSize %d\n", chunkSize, lastChunksize, enc_chunkSize)

	//making the parallelism
	var wg sync.WaitGroup
	wg.Add(numChunks)

	for i := 0; i < numChunks-1; i++ {
		go func(id int, readOffset int, writeOffset int) {
			defer wg.Done()

			//fmt.Printf("chunk %d: read at %d --> writing at %d\n", id, readOffset, writeOffset)

			buffer := make([]byte, chunkSize)
			_, err := file.ReadAt(buffer, int64(readOffset))
			if err != nil && err != io.EOF {
				x := fmt.Errorf("chunk failed reading --> %d", id)
				panic(x)
			}

			data, err := encryptBuffer(key, buffer)
			if err != nil {
				x := fmt.Errorf("chunk failed enc --> %d", id)
				panic(x)
			}

			_, err = encFile.WriteAt(data, int64(writeOffset))
			if err != nil {
				x := fmt.Errorf("chunk failed writing --> %d", id)
				panic(x)
			}

		}(i+1, i*chunkSize, initialWriteOffset+i*enc_chunkSize)
	}

	go func(id int, readOffset int, writeOffset int) {
		defer wg.Done()
		//fmt.Printf("chunk %d: read at %d --> writing at %d\n", id, readOffset, writeOffset)
		buffer := make([]byte, lastChunksize)
		_, err := file.ReadAt(buffer, int64(readOffset))
		if err != nil && err != io.EOF {
			x := fmt.Errorf("chunk failed reading --> %d", id)
			panic(x)
		}

		data, err := encryptBuffer(key, buffer)
		if err != nil {
			x := fmt.Errorf("chunk failed enc --> %d", id)
			panic(x)
		}

		_, err = encFile.WriteAt(data, int64(writeOffset))
		if err != nil {
			x := fmt.Errorf("chunk failed writing --> %d", id)
			panic(x)
		}

	}(numChunks, (numChunks-1)*chunkSize, initialWriteOffset+(numChunks-1)*enc_chunkSize)

	wg.Wait()

	return pass, nil

}
