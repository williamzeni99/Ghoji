package compressor

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

const DefaultCompresissionLevel = 4

// CompressDirectory compresses the directory at inputDir and writes the compressed output to outputFilePath
func CompressDirectory(inputDir, outputFilePath string, compressionLevel int) error {

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

	// Walk through the input directory and add files to the tar archive
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

	return nil
}

// DecompressDirectory decompresses the zstd compressed file at inputFilePath and extracts it to outputDir
func DecompressDirectory(inputFilePath, outputDir string) error {
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
	tarReader := tar.NewReader(zstdReader)

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
	}

	err = os.Remove(inputFilePath)
	if err != nil {
		return err
	}

	return nil
}
