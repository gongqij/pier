package gmsm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	text = "123456"
)

func TestEncryptPassword(t *testing.T) {
	ciphertext, err := Sm4Encrypt(text, Sm4Key)
	if err != nil {
		t.Errorf("Sm4Encrypt meet error:%v", err)
		return
	}
	t.Log("sm4加密后:", ciphertext)
	plaintext, err := Sm4Decrypt(ciphertext, Sm4Key)
	if err != nil {
		t.Errorf("Sm4Decrypt meet error:%v", err)
		return
	}
	t.Log("sm4解密后:", plaintext)
	assert.Equal(t, text, plaintext)
}
