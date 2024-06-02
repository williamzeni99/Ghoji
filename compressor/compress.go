package compressor

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

const DefaultCompresissionLevel = 3

// CompressDirectory compresses the directory at inputDir and writes the compressed output to outputFilePath
func CompressDirectory(inputDir, outputFilePath string, compressionLevel int, progress chan<- float64) error {

	// Create the output file
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create a zstd writer with the specified compression level
	zstdWriter, err := zstd.NewWriter(outputFile, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
	if err != nil {
		return fmt.Errorf("failed to create zstd writer: %w", err)
	}
	defer zstdWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(zstdWriter)
	defer tarWriter.Close()

	totalFiles := 0
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalFiles++
		}
		return nil
	})

	// Walk through the input directory and add files to the tar archive
	compressedFiles := 0
	progress <- 0.0
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a tar header for the file
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Use a relative path in the tar archive
		header.Name, err = filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		// Write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If the file is not a directory, write its content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
			compressedFiles++
			progress <- float64(compressedFiles) / float64(totalFiles)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to compress directory: %w", err)
	}

	err = os.RemoveAll(inputDir)
	if err != nil {
		return err
	}

	close(progress)
	return nil
}

// DecompressDirectory decompresses the zstd compressed file at inputFilePath and extracts it to outputDir
func DecompressDirectory(inputFilePath, outputDir string, progress chan<- float64) error {
	// Open the input file
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Create a zstd reader
	zstdReader, err := zstd.NewReader(inputFile)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	// Create a tar reader
	totalFiles := 0
	tarReader := tar.NewReader(zstdReader)
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("not a tar")
		}
		totalFiles++
	}

	//reset file
	inputFile.Seek(0, io.SeekStart)
	zstdReader.Reset(inputFile)
	tarReader = tar.NewReader(zstdReader)

	progress <- 0.0
	decompressedFiles := 0

	// Extract files from the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("not a tar")
		}
		// Determine the output path
		outputPath := filepath.Join(outputDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create file
			outputFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outputFile.Close()

			if _, err := io.Copy(outputFile, tarReader); err != nil {
				return fmt.Errorf("failed to copy file content: %w", err)
			}
		default:
			return fmt.Errorf("unknown tar header type: %v", header.Typeflag)
		}

		decompressedFiles++
		progress <- float64(decompressedFiles) / float64(totalFiles)
	}

	err = os.Remove(inputFilePath)
	if err != nil {
		return err
	}

	close(progress)
	return nil
}
