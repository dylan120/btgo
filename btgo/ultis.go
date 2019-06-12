package btgo

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"reflect"
)

const bufferSize = 65536

func MD5sum(filename string) (string, error) {
	if info, err := os.Stat(filename); err != nil {
		return "", err
	} else if info.IsDir() {
		return "", nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	for buf, reader := make([]byte, bufferSize), bufio.NewReader(file); ; {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		hash.Write(buf[:n])
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))
	return checksum, nil
}

func SendByte(conn io.ReadWriter, sendChan chan []byte, doneChan chan error) {
	var err error
	for b := range sendChan {
		n, err := conn.Write(b)
		if err != nil {
			fmt.Println("SendByte err",err)
			break
		}
		fmt.Println("SendByte ",n)
	}
	doneChan <- err
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GenerateID(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	return []byte(encoded[:n])
}

func SliceExists(slice interface{}, item interface{}) bool {
	s := reflect.ValueOf(slice)
	isExist := false
	if s.Kind() != reflect.Slice {
		log.Println("SliceExists() given a non-slice type")
	}
	for i := 0; i < s.Len(); i++ {
		if s.Index(i).Interface() == item {
			isExist = true
		}
	}
	return isExist
}
