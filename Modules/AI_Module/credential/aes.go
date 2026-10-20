package credential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type AESGCMEncryptor struct {
	aead cipher.AEAD
}

func NewAESGCMEncryptor(key []byte) (*AESGCMEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf(
			"AES-256 requires a 32-byte key, got %d bytes",
			len(key),
		)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &AESGCMEncryptor{
		aead: aead,
	}, nil
}

func (e *AESGCMEncryptor) Encrypt(
	ctx context.Context,
	plaintext []byte,
) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := e.aead.Seal(
		nonce,
		nonce,
		plaintext,
		nil,
	)

	return ciphertext, nil
}

func (e *AESGCMEncryptor) Decrypt(
	ctx context.Context,
	ciphertext []byte,
) ([]byte, error) {
	nonceSize := e.aead.NonceSize()

	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := e.aead.Open(
		nil,
		nonce,
		encrypted,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}

	return plaintext, nil
}
