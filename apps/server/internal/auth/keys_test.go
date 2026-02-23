package auth

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupTestKeyPair(t *testing.T) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	keyPair = &KeyPair{
		PrivateKey: priv,
		PublicKey:  priv.Public().(ed25519.PublicKey),
		KID:        "test-kid",
	}
	t.Cleanup(func() { keyPair = nil })
}

func TestGenerateAPIKeyJWT(t *testing.T) {
	setupTestKeyPair(t)

	token, err := GenerateAPIKeyJWT("user-123", "key-456", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have mpt_ prefix
	if !strings.HasPrefix(token, "mpt_") {
		t.Errorf("expected mpt_ prefix, got %q", token[:10])
	}

	// Parse and verify claims
	jwtStr := token[4:] // strip mpt_ prefix
	parsed, err := jwt.Parse(jwtStr, func(t *jwt.Token) (interface{}, error) {
		return keyPair.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse failed: %v", err)
	}

	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "user-123" {
		t.Errorf("sub = %v, want %q", claims["sub"], "user-123")
	}
	if claims["kid"] != "key-456" {
		t.Errorf("kid = %v, want %q", claims["kid"], "key-456")
	}
	if _, ok := claims["iat"]; !ok {
		t.Error("expected iat claim")
	}
}

func TestGenerateAPIKeyJWTWithExpiry(t *testing.T) {
	setupTestKeyPair(t)

	expiry := time.Now().Add(24 * time.Hour)
	token, err := GenerateAPIKeyJWT("user-123", "key-456", &expiry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jwtStr := token[4:]
	parsed, err := jwt.Parse(jwtStr, func(t *jwt.Token) (interface{}, error) {
		return keyPair.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse failed: %v", err)
	}

	claims := parsed.Claims.(jwt.MapClaims)
	if _, ok := claims["exp"]; !ok {
		t.Error("expected exp claim when expiresAt is provided")
	}
}

func TestGenerateAPIKeyJWTNoKeyPair(t *testing.T) {
	// Ensure keyPair is nil
	old := keyPair
	keyPair = nil
	defer func() { keyPair = old }()

	_, err := GenerateAPIKeyJWT("user-123", "key-456", nil)
	if err == nil {
		t.Error("expected error when keyPair is nil")
	}
}
