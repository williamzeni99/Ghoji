package graphic

import (
	"fmt"
	"ghoji/encryptor"
	"sync"
	"time"
)

func DoEncryption(path string, numCpu int, chunks int, maxfiles int, compress bool) {

	encryptor.DefaultGoRoutines = chunks
	encryptor.DefaultMaxFiles = maxfiles
	encryptor.MaxCPUs = numCpu
	passwd, err := readPassword()
	if err != nil {
		fmt.Printf("unable to read the password\nerr: %s", err)
		return
	}

	startTime := time.Now()

	file := encryptor.GhojiFile{
		FilePath:     path,
		New_filePath: "",
		Password:     passwd,
		Progress:     make(chan float32),
		Faults:       nil,
	}

	fmt.Printf("Encrypting file: %s \nwith %d CPUs and %d goroutines\n", path, numCpu, chunks)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for p := range file.Progress {
			fmt.Print("\r")
			fmt.Printf("Progress: %d %%", int(p*100))
		}
		wg.Done()
	}()

	file.Encrypt()
	wg.Wait()

	if file.Faults != nil {
		fmt.Println("\n\n" + file.Faults.Error())
		err = file.Rollback()
		if err != nil {
			fmt.Println("IMPOSSIBLE TO ROLLBACK")
		}
		return
	}

	elapsedTime := time.Since(startTime)
	fmt.Println("\n\nElapsed time:", elapsedTime)

	// errors := ghojierrors.GetErrorHandler()

	// info, err := os.Stat(path)
	// if err != nil {
	// 	errors <- &ghojierrors.InfoFileError{Path: path, Error: err}
	// 	close(errors)
	// }

	// passwd, err := readPassword()
	// if err != nil {
	// 	errors <- &ghojierrors.ReadPasswordError{Error: err}
	// 	close(errors)
	// }

	// startTime := time.Now()

	// if compress {
	// 	if info.IsDir() {

	// 		fmt.Printf("Starting compression of dir: %s \n", path)

	// 		newname := filepath.Base(path) + ".zst"
	// 		newpath := filepath.Join(filepath.Dir(path), newname)

	// 		progress := make(chan float64)

	// 		var wg sync.WaitGroup

	// 		wg.Add(1)
	// 		go func() {
	// 			for p := range progress {
	// 				fmt.Print("\r")
	// 				fmt.Printf("Progress: %d %%", int(p*100))
	// 			}
	// 			wg.Done()
	// 		}()

	// 		err = compressor.CompressDirectory(path, newpath, compressor.DefaultCompresissionLevel, progress)
	// 		if err != nil {
	// 			errors <- &ghojierrors.CompressionError{Path: newpath, Error: err}
	// 			close(errors)
	// 		}

	// 		wg.Wait()
	// 		fmt.Printf("\nCompressed in dir: %s \n", path)

	// 		path = newpath
	// 		info, err = os.Stat(path)
	// 		if err != nil {
	// 			errors <- &ghojierrors.InfoFileError{Path: path, Error: err}
	// 			close(errors)
	// 		}

	// 	}
	// }

	// if info.IsDir() {

	// 	fmt.Printf("Encrypting dir: %s \nwith %d CPUs, %d files per time, %d chunks each file per time\n", path, numCpu, maxfiles, chunks)

	// 	// getting files
	// 	fmt.Println("Crawling files...")
	// 	files, err := crawlFiles(path)
	// 	if err != nil {
	// 		errors <- &ghojierrors.CrawlingFilesError{Path: path, Error: err}
	// 		close(errors)
	// 	}
	// 	fmt.Printf("\rCrawled %d files\n\n", len(files))

	// 	// start parallelism
	// 	var wg sync.WaitGroup

	// 	progress_channel := make(chan int)

	// 	wg.Add(1)
	// 	go func() {
	// 		sum := 0
	// 		for p := range progress_channel {
	// 			sum += p
	// 			fmt.Print("\r")
	// 			fmt.Printf("Progress: %d / %d ", sum, len(files))
	// 		}
	// 		wg.Done()
	// 	}()

	// 	//do encryption
	// 	encryptor.EncryptMultipleFiles(passwd, files, numCpu, chunks, progress_channel, maxfiles, errors)
	// 	wg.Wait()

	// } else {

	// 	fmt.Printf("Encrypting file: %s \nwith %d CPUs and %d goroutines\n", path, numCpu, chunks)

	// 	progress := make(chan float64)
	// 	var wg sync.WaitGroup

	// 	wg.Add(1)
	// 	go func() {
	// 		for p := range progress {
	// 			fmt.Print("\r")
	// 			fmt.Printf("Progress: %d %%", int(p*100))
	// 		}
	// 		wg.Done()
	// 	}()

	// 	_ = encryptor.EncryptFile(passwd, path, numCpu, chunks, progress, errors)
	// 	wg.Wait()
	// }
	// close(errors)
	// elapsedTime := time.Since(startTime)
	// fmt.Println("\n\nElapsed time:", elapsedTime)

}
