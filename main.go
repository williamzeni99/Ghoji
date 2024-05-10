package main

import (
	"fmt"
	"go-warshield/encryptor"
	"os"
	"strconv"
	"time"
)

func main() {
	numCpu, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	numChunks, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(err)
	}

	fmt.Printf("Start encryption with %d CPUs and %d numChunks\n", numCpu, numChunks)
	startTime := time.Now()

	pass, err := encryptor.EncryptFile("/home/willianzeni/Desktop/backup.zip", "/home/willianzeni/Desktop", numCpu, numChunks)
	if err != nil {
		panic(err)
	}
	fmt.Println(pass)

	elapsedTime := time.Since(startTime)
	fmt.Println("Elapsed time encrypt:", elapsedTime)

	// fmt.Println("Start decryption")

	// startTime = time.Now()

	// name := "backup.zip"
	// shaName := sha256.Sum256([]byte(name))
	// encFileName := hex.EncodeToString(shaName[:])

	// path := filepath.Join("/home/willianzeni/Desktop", encFileName)

	// err = encryptor.DecryptFile(pass, path, "/home/willianzeni/Downloads", numCpu)
	// if err != nil {
	// 	panic(err)
	// }
	// elapsedTime = time.Since(startTime)
	// fmt.Println("Elapsed time decrypt:", elapsedTime)
}
