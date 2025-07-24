package util

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
)

func DecryptAES(key []byte, ct string) string {
	data, _ := hex.DecodeString(ct)

	c, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	size := c.BlockSize()
	data = PKCS7Padding(data, size)
	pt := make([]byte, len(data))
	for bs, be := 0, size; bs < len(data); bs, be = bs+size, be+size {
		c.Decrypt(pt[bs:be], data[bs:be])
	}
	return string(pt)
}

func AesDecrypt(key []byte, ciphertext string) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil 
	}
	if block == nil {
		return []byte{}
	}
	blockSize := block.BlockSize()
	if len(ciphertext)%blockSize != 0 {
		return nil
	}
	plaintext := make([]byte, len(ciphertext))
	for bs, be := 0, blockSize; bs < len(ciphertext); bs, be = bs+blockSize, be+blockSize {
		block.Decrypt(plaintext[bs:be], ciphertext[bs:be])
	}
	plaintext = PKCS7UnPadding(plaintext)
	return plaintext
}
func PKCS7UnPadding(data []byte) []byte {
	padding := data[len(data)-1]
	if int(padding) > len(data) {
		return nil
	}
	return data[:len(data)-int(padding)]
}

func AesEncrypt(data, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	if block == nil {
		return []byte{}
	}
	data = PKCS7Padding(data, block.BlockSize())
	decrypted := make([]byte, len(data))
	size := block.BlockSize()
	for bs, be := 0, size; bs < len(data); bs, be = bs+size, be+size {
		block.Encrypt(decrypted[bs:be], data[bs:be])
	}
	return decrypted
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
