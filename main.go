package main

import (
	"fmt"
	"go-warshield/encryptor"
	"os"
	"sync"
	"time"
)

func main() {

	var password string
	var filePath string
	var option string

	if len(os.Args) < 2 {
		//todo
		panic("nope")
	} else {
		option = os.Args[1]
		filePath = os.Args[2]
	}

	fmt.Print("Insert password: ")
	_, err := fmt.Scanf("%s", &password)
	if err != nil {
		panic(err)
	}

	progress := make(chan float64)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for p := range progress {

			fmt.Print("\r") // Torna a capo sulla stessa riga
			fmt.Printf("Progress: %d %%", int(p*100))
		}
		wg.Done()
	}()

	startTime := time.Now()

	if option == "enc" {
		fmt.Printf("Start encryption\n")
		err := encryptor.EncryptFile(password, filePath, encryptor.MaxCPUs, encryptor.MaxGoRoutines, progress)
		if err != nil {
			panic(err)
		}

	} else {
		fmt.Println("Start decryption")
		err = encryptor.DecryptFile(password, filePath, encryptor.MaxCPUs, encryptor.MaxGoRoutines, progress)
		if err != nil {
			panic(err)
		}
	}

	wg.Wait()

	elapsedTime := time.Since(startTime)
	fmt.Println("\n\nElapsed time:", elapsedTime)

}
