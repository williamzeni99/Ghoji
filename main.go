package main

import (
	"fmt"
	"go-warshield/encryptor"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {

	var password string
	var filePath string
	var mode string
	var numCpu int
	var goroutines int
	var err error

	if len(os.Args) < 2 {
		//todo graphic part
		panic("nope")
	} else {
		mode = os.Args[1]
		filePath = os.Args[2]

		numCpu, err = strconv.Atoi(os.Args[3])
		if err != nil {
			panic(err)
		}

		goroutines, err = strconv.Atoi(os.Args[4])
		if err != nil {
			panic(err)
		}

		//todo make it better

	}

	fmt.Print("Insert password: ")
	_, err = fmt.Scanf("%s", &password)
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

	if mode == "enc" {
		fmt.Printf("Start encryption\n")
		err := encryptor.EncryptFile(password, filePath, numCpu, goroutines, progress)
		if err != nil {
			panic(err)
		}

	} else {
		fmt.Println("Start decryption")
		err = encryptor.DecryptFile(password, filePath, numCpu, goroutines, progress)
		if err != nil {
			panic(err)
		}
	}

	wg.Wait()

	elapsedTime := time.Since(startTime)
	fmt.Println("\n\nElapsed time:", elapsedTime)

}
