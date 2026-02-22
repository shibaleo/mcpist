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

	"gorm.io/gorm"
)

var encryptionKey []byte

// InitEncryptionKey loads CREDENTIAL_ENCRYPTION_KEY from env.
// Must be called at startup. If not set, encryption is disabled (passthrough).
func InitEncryptionKey() {
	raw := os.Getenv("CREDENTIAL_ENCRYPTION_KEY")
	if raw == "" {
		return
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		panic(fmt.Sprintf("CREDENTIAL_ENCRYPTION_KEY must be 32 bytes base64-encoded (got %d bytes)", len(key)))
	}
	encryptionKey = key
}

// EncryptionEnabled reports whether credential encryption is configured.
func EncryptionEnabled() bool {
	return encryptionKey != nil
}

const encryptionVersion = "v1"

// encrypt encrypts plaintext with AES-256-GCM.
// Returns "v1:" + base64-encoded nonce+ciphertext.
func encrypt(plaintext []byte) (string, error) {
	if encryptionKey == nil {
		return "", errors.New("encryption key not configured")
	}
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
// Supports "v1:base64data" format. Falls back to raw base64 for unversioned data.
func decrypt(ciphertext string) ([]byte, error) {
	if encryptionKey == nil {
		return nil, errors.New("encryption key not configured")
	}

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

// MigrateEncryption encrypts all plaintext credentials in the database.
// Skips rows that already have encrypted data. Returns counts.
func MigrateEncryption(gormDB *gorm.DB, dryRun bool) (credsMigrated, oauthMigrated int, err error) {
	if !EncryptionEnabled() {
		return 0, 0, errors.New("CREDENTIAL_ENCRYPTION_KEY not set")
	}

	// 1. user_credentials: encrypt rows where credentials != "" and encrypted_credentials is null
	var creds []UserCredential
	if err := gormDB.Where("credentials != '' AND (encrypted_credentials IS NULL OR encrypted_credentials = '')").Find(&creds).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to query user_credentials: %w", err)
	}
	for _, c := range creds {
		enc, err := encrypt([]byte(c.Credentials))
		if err != nil {
			return credsMigrated, oauthMigrated, fmt.Errorf("failed to encrypt credential %s: %w", c.ID, err)
		}
		if !dryRun {
			if err := gormDB.Model(&c).Updates(map[string]interface{}{
				"encrypted_credentials": enc,
				"credentials":           "",
			}).Error; err != nil {
				return credsMigrated, oauthMigrated, fmt.Errorf("failed to update credential %s: %w", c.ID, err)
			}
		}
		credsMigrated++
	}

	// 2. oauth_apps: encrypt rows where client_secret != "" and encrypted_client_secret is null
	var apps []OAuthApp
	if err := gormDB.Where("client_secret != '' AND (encrypted_client_secret IS NULL OR encrypted_client_secret = '')").Find(&apps).Error; err != nil {
		return credsMigrated, 0, fmt.Errorf("failed to query oauth_apps: %w", err)
	}
	for _, a := range apps {
		enc, err := encrypt([]byte(a.ClientSecret))
		if err != nil {
			return credsMigrated, oauthMigrated, fmt.Errorf("failed to encrypt oauth_app %s: %w", a.Provider, err)
		}
		if !dryRun {
			if err := gormDB.Model(&a).Updates(map[string]interface{}{
				"encrypted_client_secret": enc,
				"client_secret":           "",
			}).Error; err != nil {
				return credsMigrated, oauthMigrated, fmt.Errorf("failed to update oauth_app %s: %w", a.Provider, err)
			}
		}
		oauthMigrated++
	}

	return credsMigrated, oauthMigrated, nil
}
