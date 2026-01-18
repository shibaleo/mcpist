package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
)

// sha256Hash computes SHA-256 hash
func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// rsaVerifyPKCS1v15 verifies RSA PKCS#1 v1.5 signature
func rsaVerifyPKCS1v15(pubKey *rsa.PublicKey, hashed, signature []byte) error {
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed, signature)
}
