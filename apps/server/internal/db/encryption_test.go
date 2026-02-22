package db

import (
	"encoding/base64"
	"strings"
	"testing"
)

func setupTestKey(t *testing.T) {
	t.Helper()
	// 32 bytes for AES-256
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	encryptionKey = key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setupTestKey(t)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"empty string", ""},
		{"json credentials", `{"access_token":"abc","refresh_token":"def"}`},
		{"unicode", "こんにちは世界"},
		{"large payload", strings.Repeat("a", 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encrypt([]byte(tt.plaintext))
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			decrypted, err := decrypt(encrypted)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if string(decrypted) != tt.plaintext {
				t.Errorf("roundtrip mismatch: got %q, want %q", string(decrypted), tt.plaintext)
			}
		})
	}
}

func TestEncryptProducesVersionedFormat(t *testing.T) {
	setupTestKey(t)

	encrypted, err := encrypt([]byte("test"))
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if !strings.HasPrefix(encrypted, "v1:") {
		t.Errorf("expected v1: prefix, got %q", encrypted[:10])
	}

	// After v1: should be valid base64
	b64 := encrypted[3:]
	if _, err := base64.StdEncoding.DecodeString(b64); err != nil {
		t.Errorf("base64 decode failed: %v", err)
	}
}

func TestEncryptProducesUniqueOutput(t *testing.T) {
	setupTestKey(t)

	plaintext := []byte("same input")
	a, _ := encrypt(plaintext)
	b, _ := encrypt(plaintext)

	if a == b {
		t.Error("two encryptions of same plaintext should differ (random nonce)")
	}
}

func TestDecryptInvalidInput(t *testing.T) {
	setupTestKey(t)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", "v1:not-valid-base64!!!"},
		{"too short", "v1:" + base64.StdEncoding.EncodeToString([]byte("short"))},
		{"tampered", func() string {
			encrypted, _ := encrypt([]byte("original"))
			// Flip a byte in the ciphertext portion
			b := []byte(encrypted)
			b[len(b)-2] ^= 0xff
			return string(b)
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decrypt(tt.ciphertext)
			if err == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}
