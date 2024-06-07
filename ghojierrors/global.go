package ghojierrors

import (
	"fmt"
	"os"
	"sync"
)

type Handable interface {
	Handle()
	Message()
}

// Open file error
type OpenFileError struct {
	Path  string
	Error error
}

func (e *OpenFileError) Handle() {
}

func (e *OpenFileError) Message() {
	fmt.Printf("\n\n[!]Error: Open file failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// Create file error
type CreateFileError struct {
	Path  string
	Error error
}

func (e *CreateFileError) Handle() {
	//
}

func (e *CreateFileError) Message() {
	fmt.Printf("\n\n[!]Error: Create file failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// Info file error
type InfoFileError struct {
	Path  string
	Error error
}

func (e *InfoFileError) Handle() {
	//
}

func (e *InfoFileError) Message() {
	fmt.Printf("\n\n[!]Error: Getting file info failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// remove file error
type RemoveFileError struct {
	Path  string
	Error error
}

func (e *RemoveFileError) Handle() {
	//
}

func (e *RemoveFileError) Message() {
	fmt.Printf("\n\n[!]Error: Remove file failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// remove dir error
type RemoveDirError struct {
	Path  string
	Error error
}

func (e *RemoveDirError) Handle() {
	//
}

func (e *RemoveDirError) Message() {
	fmt.Printf("\n\n[!]Error: Remove directory failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// close file error
type CloseFileError struct {
	Path  string
	Error error
}

func (e *CloseFileError) Handle() {
	//
}

func (e *CloseFileError) Message() {
	fmt.Printf("\n\n[!]Error: Close file failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// Compression Error
type CompressionError struct {
	Path  string
	Error error
}

func (e *CompressionError) Handle() {
	os.Remove(e.Path)
}

func (e *CompressionError) Message() {
	fmt.Printf("\n\n[!]Error: Compression failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// Decompression Error
type DecompressionError struct {
	Path  string
	Error error
}

func (e *DecompressionError) Handle() {
	os.Remove(e.Path)
}

func (e *DecompressionError) Message() {
	fmt.Printf("\n\n[!]Error: Decompression failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// Crawling Error
type CrawlingFilesError struct {
	Path  string
	Error error
}

func (e *CrawlingFilesError) Handle() {
	//
}

func (e *CrawlingFilesError) Message() {
	fmt.Printf("\n\n[!]Error: Crawling files failed.\npath: %s\nTrigger: %s\n\n", e.Path, e.Error)
}

// Read Password Error
type ReadPasswordError struct {
	Error error
}

func (e *ReadPasswordError) Handle() {
	//
}

func (e *ReadPasswordError) Message() {
	fmt.Printf("\n\n[!]Error: Reading password failed.\nTrigger: %s\n\n", e.Error)
}

// File encryption error
type FileEncryptionFailed struct {
	Path string
}

func (e *FileEncryptionFailed) Handle() {
	err := os.Remove(e.Path)
	if err != nil {
		fmt.Println(err)
	}
}

func (e *FileEncryptionFailed) Message() {
	fmt.Printf("\n\n[!]Error: Encryption file failed.\npath: %s\nIf you see this error please open an issue on the github repo\n\n", e.Path)
}

// file decryption error
type FileDecryptionFailed struct {
	Path string
}

func (e *FileDecryptionFailed) Handle() {
	err := os.Remove(e.Path)
	if err != nil {
		fmt.Println(err)
	}
}

func (e *FileDecryptionFailed) Message() {
	fmt.Printf("\n\n[!]Error: Decryption file failed.\npath: %s\nProbably the password is wrong.\nIf you think this is an error please open an issue on the github repo\n\n", e.Path)
}

// failed decryption extension
type FileExtDecryptionFailed struct {
	Path string
}

func (e *FileExtDecryptionFailed) Handle() {
	err := os.Remove(e.Path)
	if err != nil {
		fmt.Println(err)
	}
}

func (e *FileExtDecryptionFailed) Message() {
	fmt.Printf("\n\n[!]Error: Decryption file failed.\npath: %s\nWrong file extension.\nIf you think this is an error please open an issue on the github repo\n\n", e.Path)
}

func GetErrorHandler() chan Handable {
	errors := make(chan Handable)

	go func() {
		first := true

		var wg sync.WaitGroup

		for err := range errors {
			wg.Add(1)
			if first {
				err.Message()
				first = false
			}
			go func(ghojierror Handable) {
				ghojierror.Handle()
				wg.Done()
			}(err)
		}

		wg.Wait()

		if !first {
			os.Exit(0)
		}
	}()

	return errors
}
