package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key Type (RSA)
	Kid string `json:"kid"` // Key ID
	Use string `json:"use"` // Key Use (sig)
	Alg string `json:"alg"` // Algorithm (RS256)
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
}

// JWKSCache caches JWKS with automatic refresh
type JWKSCache struct {
	mu          sync.RWMutex
	jwksURL     string
	keys        map[string]*rsa.PublicKey // kid -> public key
	lastFetched time.Time
	cacheTTL    time.Duration
	httpClient  *http.Client
}

// NewJWKSCache creates a new JWKS cache
func NewJWKSCache(jwksURL string) *JWKSCache {
	return &JWKSCache{
		jwksURL:    jwksURL,
		keys:       make(map[string]*rsa.PublicKey),
		cacheTTL:   1 * time.Hour,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetKey retrieves a public key by key ID
func (c *JWKSCache) GetKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if key, ok := c.keys[kid]; ok && time.Since(c.lastFetched) < c.cacheTTL {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	// Refresh cache
	if err := c.refresh(); err != nil {
		return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	key, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", kid)
	}
	return key, nil
}

// refresh fetches JWKS from the server
func (c *JWKSCache) refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check after acquiring write lock
	if time.Since(c.lastFetched) < c.cacheTTL && len(c.keys) > 0 {
		return nil
	}

	resp, err := c.httpClient.Get(c.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS fetch returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}
		pubKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			continue // Skip invalid keys
		}
		newKeys[jwk.Kid] = pubKey
	}

	c.keys = newKeys
	c.lastFetched = time.Now()

	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64URLDecode(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode N: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode E (exponent)
	eBytes, err := base64URLDecode(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E: %w", err)
	}
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

// base64URLDecode decodes a base64url encoded string
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
