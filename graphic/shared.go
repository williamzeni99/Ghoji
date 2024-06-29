package graphic

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/term"
)

func crawlFiles(path string) ([]string, error) {

	var files []string

	// WalkDir the directory tree
	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// If it's a file, add it to the slice
		if !d.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, absPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("directory empty. No files found")
	}

	return files, nil
}

func readPassword() ([32]byte, error) {
	fmt.Print("Enter password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return [32]byte{}, err
	}
	fmt.Printf("\n\n")
	password := sha256.Sum256(bytePassword)

	return password, nil

}
