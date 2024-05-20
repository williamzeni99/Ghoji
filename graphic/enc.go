package graphic

import (
	"fmt"
	"ghoji/encryptor"
	"sync"
	"time"
)

func DoEncryption(path string, numCpu int, goroutines int) {

	var passwd string

	fmt.Print("Insert password: ")
	_, err := fmt.Scanf("%s", &passwd)
	if err != nil {
		panic(err)
	}

	progress := make(chan float64)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for p := range progress {
			fmt.Print("\r")
			fmt.Printf("Progress: %d %%", int(p*100))
		}
		wg.Done()
	}()

	fmt.Printf("Encrypting file: %s \nwith %d CPUs and %d goroutines\n", path, numCpu, goroutines)
	startTime := time.Now()

	err = encryptor.EncryptFile(passwd, path, numCpu, goroutines, progress)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	wg.Wait()

	elapsedTime := time.Since(startTime)
	fmt.Println("\n\nElapsed time:", elapsedTime)

}
