package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

var encryptionKey []byte

// InitEncryptionKey loads CREDENTIAL_ENCRYPTION_KEY from env.
// Must be called at startup. Panics if the key is not set or invalid.
func InitEncryptionKey() {
	raw := os.Getenv("CREDENTIAL_ENCRYPTION_KEY")
	if raw == "" {
		panic("CREDENTIAL_ENCRYPTION_KEY is required")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		panic(fmt.Sprintf("CREDENTIAL_ENCRYPTION_KEY must be 32 bytes base64-encoded (got %d bytes)", len(key)))
	}
	encryptionKey = key
}

const encryptionVersion = "v1"

// encrypt encrypts plaintext with AES-256-GCM.
// Returns "v1:" + base64-encoded nonce+ciphertext.
func encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil) // nonce || ciphertext || tag
	return encryptionVersion + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// decrypt decrypts a versioned ciphertext string.
// Supports "v1:base64data" format.
func decrypt(ciphertext string) ([]byte, error) {
	data := ciphertext
	if strings.HasPrefix(ciphertext, "v1:") {
		data = ciphertext[3:]
	}

	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	return gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
}
