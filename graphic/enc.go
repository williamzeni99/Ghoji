package graphic

import (
	"fmt"
	"ghoji/compressor"
	"ghoji/encryptor"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func DoEncryption(path string, numCpu int, chunks int, maxfiles int, compress bool) {

	info, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	passwd, err := readPassword()
	if err != nil {
		fmt.Println(err)
		return
	}

	startTime := time.Now()

	if compress {
		if info.IsDir() {

			fmt.Printf("Starting compression of dir: %s \n", path)

			newname := filepath.Base(path) + ".zst"
			newpath := filepath.Join(filepath.Dir(path), newname)

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

			err = compressor.CompressDirectory(path, newpath, compressor.DefaultCompresissionLevel, progress)
			if err != nil {
				fmt.Println(err)
				return
			}

			wg.Wait()
			fmt.Printf("\nCompressed in dir: %s \n", path)

			path = newpath
			info, err = os.Stat(path)
			if err != nil {
				fmt.Println(err)
				return
			}

		}
	}

	if info.IsDir() {

		fmt.Printf("Encrypting dir: %s \nwith %d CPUs, %d files per time, %d chunks each file per time\n", path, numCpu, maxfiles, chunks)

		// getting files
		fmt.Println("Crawling files...")
		files, err := crawlFiles(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("\rCrawled %d files\n\n", len(files))

		// start parallelism
		var wg sync.WaitGroup

		progress_channel := make(chan int)

		wg.Add(1)
		go func() {
			sum := 0
			for p := range progress_channel {
				sum += p
				fmt.Print("\r")
				fmt.Printf("Progress: %d / %d ", sum, len(files))
			}
			wg.Done()
		}()

		//do encryption
		err = encryptor.EncryptMultipleFiles(passwd, files, numCpu, chunks, progress_channel, maxfiles)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		wg.Wait()

	} else {

		fmt.Printf("Encrypting file: %s \nwith %d CPUs and %d goroutines\n", path, numCpu, chunks)

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

		_, err = encryptor.EncryptFile(passwd, path, numCpu, chunks, progress)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		wg.Wait()
	}

	elapsedTime := time.Since(startTime)
	fmt.Println("\n\nElapsed time:", elapsedTime)

}
