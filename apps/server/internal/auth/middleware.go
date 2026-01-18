package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "userId"
)

// Middleware handles authentication
type Middleware struct {
	jwksCache *JWKSCache
	issuer    string
	audience  string
}

// Config holds middleware configuration
type Config struct {
	JWKSURL  string
	Issuer   string
	Audience string
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(cfg Config) *Middleware {
	return &Middleware{
		jwksCache: NewJWKSCache(cfg.JWKSURL),
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
	}
}

// NewMiddlewareFromEnv creates middleware from environment variables
func NewMiddlewareFromEnv() *Middleware {
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		supabaseURL = "http://localhost:54321"
	}

	return NewMiddleware(Config{
		JWKSURL:  supabaseURL + "/auth/v1/.well-known/jwks.json",
		Issuer:   supabaseURL + "/auth/v1",
		Audience: os.Getenv("JWT_AUDIENCE"), // Optional
	})
}

// Authenticate is HTTP middleware that validates tokens
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := m.ValidateRequest(r)
		if err != nil {
			log.Printf("Auth failed: %v", err)
			m.writeUnauthorizedResponse(w)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateRequest validates the token from the Authorization header
// Supports: MCP Token (long-lived, 64 char hex), Supabase Auth JWT (short-lived)
func (m *Middleware) ValidateRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// 1. MCP Token check (64 char hex, long-lived)
	if len(token) == 64 && isHexString(token) {
		userID, err := m.validateMCPToken(token)
		if err == nil {
			log.Printf("Auth: MCP token (user: %s)", userID)
			return userID, nil
		}
		// If it looks like MCP token but invalid, return error
		return "", fmt.Errorf("invalid MCP token: %w", err)
	}

	// 2. Supabase Auth JWT check (short-lived)
	return m.ValidateJWT(token)
}

// isHexString checks if the string is a valid hex string
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// validateMCPToken validates a long-lived MCP token via Console API
func (m *Middleware) validateMCPToken(token string) (string, error) {
	vaultURL := os.Getenv("VAULT_URL")
	if vaultURL == "" {
		vaultURL = "http://localhost:3000/api"
	}
	internalKey := os.Getenv("INTERNAL_SERVICE_KEY")

	// Call Console API to validate token
	req, err := http.NewRequest("POST", vaultURL+"/mcp-token/validate", strings.NewReader(`{"token":"`+token+`"}`))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if internalKey != "" {
		req.Header.Set("X-Internal-Service-Key", internalKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to validate MCP token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MCP token validation failed: status %d", resp.StatusCode)
	}

	var result struct {
		UserID string `json:"user_id"`
		Valid  bool   `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Valid {
		return "", fmt.Errorf("MCP token is invalid or expired")
	}

	return result.UserID, nil
}

// ValidateJWT validates a JWT token and returns the user ID
func (m *Middleware) ValidateJWT(token string) (string, error) {
	// Parse JWT (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	// Decode header
	headerJSON, err := base64URLDecode(parts[0])
	if err != nil {
		return "", fmt.Errorf("failed to decode header: %w", err)
	}

	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", fmt.Errorf("failed to parse header: %w", err)
	}

	// Verify algorithm
	if header.Alg != "RS256" {
		return "", fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// Get public key from JWKS
	pubKey, err := m.jwksCache.GetKey(header.Kid)
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify signature
	if err := m.verifySignature(parts[0]+"."+parts[1], parts[2], pubKey); err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}

	// Decode payload
	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	// Validate claims
	if err := m.validateClaims(&claims); err != nil {
		return "", err
	}

	log.Printf("Auth: JWT (user: %s)", claims.Sub)
	return claims.Sub, nil
}

// Claims represents JWT claims
type Claims struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Aud   string `json:"aud"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

// validateClaims validates JWT claims
func (m *Middleware) validateClaims(claims *Claims) error {
	now := time.Now().Unix()

	// Check expiration
	if claims.Exp < now {
		return fmt.Errorf("token expired")
	}

	// Check issuer (if configured)
	if m.issuer != "" && claims.Iss != m.issuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", m.issuer, claims.Iss)
	}

	// Check audience (if configured)
	if m.audience != "" && claims.Aud != m.audience {
		return fmt.Errorf("invalid audience: expected %s, got %s", m.audience, claims.Aud)
	}

	// Check subject (user ID)
	if claims.Sub == "" {
		return fmt.Errorf("missing subject claim")
	}

	return nil
}

// verifySignature verifies the JWT signature using RS256
func (m *Middleware) verifySignature(message, signature string, pubKey *rsa.PublicKey) error {
	// Decode signature
	sigBytes, err := base64URLDecode(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the message
	hash := sha256Hash([]byte(message))

	// Verify signature
	return rsaVerifyPKCS1v15(pubKey, hash, sigBytes)
}

// writeUnauthorizedResponse writes a 401 response with WWW-Authenticate header
func (m *Middleware) writeUnauthorizedResponse(w http.ResponseWriter) {
	consoleURL := os.Getenv("CONSOLE_URL")
	if consoleURL == "" {
		consoleURL = "http://localhost:3000"
	}
	resourceMetadataURL := consoleURL + "/.well-known/oauth-protected-resource"

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s"`, resourceMetadataURL))
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}
