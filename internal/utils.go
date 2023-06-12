package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func Unwrap(err error) {
	if err != nil {
		panic(err)
	}
}

func CheckSum(path string) (*string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	hexString := hex.EncodeToString(sum[:])
	return &hexString, nil
}
