package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GatewayClaims represents the claims in a gateway JWT from Worker.
type GatewayClaims struct {
	jwt.RegisteredClaims
	UserID  string `json:"user_id,omitempty"`
	ClerkID string `json:"clerk_id,omitempty"`
	Email   string `json:"email,omitempty"`
}

type jwksKey struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// GatewayVerifier verifies gateway JWTs using JWKS from the Worker.
type GatewayVerifier struct {
	jwksURL   string
	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
	cacheTTL  time.Duration
}

// NewGatewayVerifier creates a new gateway verifier that fetches public keys
// from the Worker's /.well-known/jwks.json endpoint.
func NewGatewayVerifier(jwksURL string) *GatewayVerifier {
	return &GatewayVerifier{
		jwksURL:  jwksURL,
		keys:     make(map[string]ed25519.PublicKey),
		cacheTTL: 5 * time.Minute,
	}
}

// VerifyToken verifies a gateway JWT and returns the claims.
func (v *GatewayVerifier) VerifyToken(tokenString string) (*GatewayClaims, error) {
	// Parse without verification to get kid from header
	unverified, _, err := jwt.NewParser().ParseUnverified(tokenString, &GatewayClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	kid, ok := unverified.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, fmt.Errorf("missing kid in token header")
	}

	// Get the public key (from cache or by fetching JWKS)
	key, err := v.getKey(kid)
	if err != nil {
		return nil, err
	}

	// Verify the token with proper validation
	claims := &GatewayClaims{}
	_, err = jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	}, jwt.WithIssuer("mcpist-gateway"), jwt.WithLeeway(5*time.Second))

	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	return claims, nil
}

// getKey returns the public key for the given kid, fetching JWKS if cache
// is empty or expired. On unknown kid, forces a refetch (key rotation support).
func (v *GatewayVerifier) getKey(kid string) (ed25519.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	expired := time.Since(v.fetchedAt) > v.cacheTTL
	v.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	// Fetch JWKS (either cache expired or kid not found)
	if err := v.fetchJWKS(); err != nil {
		// If we have a cached key, use it even if expired
		if ok {
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("key with kid %q not found in JWKS", kid)
	}

	return key, nil
}

func (v *GatewayVerifier) fetchJWKS() error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(v.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS from %s: %w", v.jwksURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS fetch returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	keys := make(map[string]ed25519.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" || k.X == "" {
			continue
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			log.Printf("[gateway] failed to decode key %s: %v", k.Kid, err)
			continue
		}
		if len(xBytes) != ed25519.PublicKeySize {
			log.Printf("[gateway] invalid key size for %s: %d", k.Kid, len(xBytes))
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(xBytes)
	}

	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	v.mu.Unlock()

	log.Printf("[gateway] JWKS refreshed: %d key(s) loaded from %s", len(keys), v.jwksURL)
	return nil
}
