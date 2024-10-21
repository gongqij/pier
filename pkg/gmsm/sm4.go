package gmsm

import (
	"encoding/base64"

	"github.com/tjfoc/gmsm/sm4"
)

var Sm4Key = "PierSm4SecretKey"

func Sm4Encrypt(plainText string, key string) (string, error) {
	cipherText, err := sm4.Sm4Ecb([]byte(key), []byte(plainText), true)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func Sm4Decrypt(cipherTextBase64Str string, key string) (string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(cipherTextBase64Str)
	if err != nil {
		return "", err
	}
	plainText, err := sm4.Sm4Ecb([]byte(key), cipherText, false)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}
