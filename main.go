package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"io"
)

const (
	maxFilename_length = 256 //necessary to get the name from the encoded data
)

type Zeni struct {
	FileName string
	Data     []byte
}

func encode(x Zeni) []byte {
	filename := make([]byte, maxFilename_length)
	copy(filename, []byte(x.FileName))

	var buf []byte

	buf = append(buf, filename...)
	buf = append(buf, x.Data...)

	return buf
}

func decode(file []byte) (Zeni, error) {

	if len(file) < maxFilename_length {
		return Zeni{}, fmt.Errorf("decode failed")
	}

	bytesname := file[:maxFilename_length]
	trimmed := bytes.Trim(bytesname, "\x00")
	filename := string(trimmed)
	data := file[maxFilename_length:]

	return Zeni{
		FileName: filename,
		Data:     data,
	}, nil
}

func main() {

	// file, err := os.ReadFile("test.txt")
	// if err != nil {
	// 	panic(err.Error())
	// }
	// test := Zeni{FileName: "test2.txt", Data: file}

	// data, err := encrypt("lol", test)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// err = os.WriteFile(test.FileName, data, 0777)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// file2, err := os.ReadFile(test.FileName)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// data2, err := decrypt("lol", file2)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// err = os.WriteFile(data2.FileName, data2.Data, 0777)
	// if err != nil {
	// 	panic(err.Error())
	// }
}

func encryptFile() {

}

func encrypt(password string, file Zeni) ([]byte, error) {
	key := get32bitkey(password)
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	data := gcm.Seal(nil, nonce, encode(file), nil)

	out := append(nonce, data...)

	return out, nil
}

func get32bitkey(password string) []byte {
	key := sha512.Sum512([]byte(password))
	return key[:32]
}

func decrypt(password string, file []byte) (Zeni, error) {
	key := get32bitkey(password)
	c, err := aes.NewCipher(key)
	if err != nil {
		return Zeni{}, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return Zeni{}, err
	}

	nonce := file[:gcm.NonceSize()]
	data := file[gcm.NonceSize():]

	plaindata, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return Zeni{}, err
	}

	return decode(plaindata)

}
