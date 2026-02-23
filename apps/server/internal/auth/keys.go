package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// KeyPair holds the Ed25519 signing key pair for JWT API keys.
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KID        string // Key ID for JWKS
}

var keyPair *KeyPair

// Init loads the Ed25519 private key from the API_KEY_PRIVATE_KEY environment variable.
// The key must be base64-encoded (64-byte Ed25519 seed+public or 32-byte seed).
func Init() error {
	encoded := os.Getenv("API_KEY_PRIVATE_KEY")
	if encoded == "" {
		log.Printf("[auth] API_KEY_PRIVATE_KEY not set, JWT API key features disabled")
		return nil
	}

	seed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("failed to decode API_KEY_PRIVATE_KEY: %w", err)
	}

	var privKey ed25519.PrivateKey
	switch len(seed) {
	case ed25519.SeedSize: // 32 bytes — seed only
		privKey = ed25519.NewKeyFromSeed(seed)
	case ed25519.PrivateKeySize: // 64 bytes — full private key
		privKey = ed25519.PrivateKey(seed)
	default:
		return fmt.Errorf("invalid key size: %d (expected 32 or 64)", len(seed))
	}

	keyPair = &KeyPair{
		PrivateKey: privKey,
		PublicKey:  privKey.Public().(ed25519.PublicKey),
		KID:        "mcpist-api-key-v1",
	}

	log.Printf("[auth] Ed25519 key pair loaded (kid: %s)", keyPair.KID)
	return nil
}

// GetKeyPair returns the loaded key pair, or nil if not initialized.
func GetKeyPair() *KeyPair {
	return keyPair
}

// GenerateAPIKeyJWT creates a signed JWT for an API key.
func GenerateAPIKeyJWT(userID, keyID string, expiresAt *time.Time) (string, error) {
	if keyPair == nil {
		return "", fmt.Errorf("signing key not configured")
	}

	claims := jwt.MapClaims{
		"sub":  userID,
		"role": "user",
		"kid":  keyID,
		"iat":  time.Now().Unix(),
	}
	if expiresAt != nil {
		claims["exp"] = expiresAt.Unix()
	}

	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, claims)
	token.Header["kid"] = keyPair.KID

	signed, err := token.SignedString(keyPair.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return "mpt_" + signed, nil
}
