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
// Also accepts X-User-ID header from trusted gateway (Cloudflare Worker)
func (m *Middleware) ValidateRequest(r *http.Request) (string, error) {
	// Check for X-User-ID from trusted gateway (Worker経由のリクエスト)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		// Verify gateway secret in production
		gatewaySecret := os.Getenv("GATEWAY_SECRET")
		if gatewaySecret != "" {
			// Production: require valid gateway secret
			requestSecret := r.Header.Get("X-Gateway-Secret")
			if requestSecret != gatewaySecret {
				log.Printf("Auth: gateway secret mismatch")
				return "", fmt.Errorf("invalid gateway secret")
			}
		}
		log.Printf("Auth: validated via gateway (user: %s)", userID)
		return userID, nil
	}

	// 直接アクセスの場合は Authorization ヘッダーで認証
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Delegate all token validation to Supabase RPC
	userID, err := m.validateToken(token)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	log.Printf("Auth: validated (user: %s)", userID)
	return userID, nil
}

// validateToken validates any token (API Key or JWT)
// Token type is determined by format:
// - API Key: mpt_<32 hex chars> -> validated via Supabase RPC (Vault access required)
// - JWT: header.payload.signature -> validated locally using JWKS
func (m *Middleware) validateToken(token string) (string, error) {
	// Validate API Key format: mpt_<32 hex chars> (total 36 chars)
	if strings.HasPrefix(token, "mpt_") {
		if len(token) != 36 {
			return "", fmt.Errorf("invalid API key format: wrong length")
		}
		// Validate hex chars after prefix
		hexPart := token[4:] // after "mpt_"
		for _, c := range hexPart {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return "", fmt.Errorf("invalid API key format: invalid characters")
			}
		}
		return m.validateAPIKey(token)
	}

	// Validate JWT format: base64url.base64url.base64url
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		// Basic validation: each part should be non-empty and valid base64url
		for i, part := range parts {
			if len(part) == 0 {
				return "", fmt.Errorf("invalid JWT format: empty part %d", i)
			}
			// Check for valid base64url characters
			for _, c := range part {
				if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
					(c >= '0' && c <= '9') || c == '-' || c == '_') {
					return "", fmt.Errorf("invalid JWT format: invalid characters in part %d", i)
				}
			}
		}
		return m.validateJWT(token)
	}

	return "", fmt.Errorf("unknown token format")
}

// validateAPIKey validates an API Key by calling Supabase RPC
func (m *Middleware) validateAPIKey(apiKey string) (string, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if serviceKey == "" {
		return "", fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY not configured")
	}

	// Call Supabase RPC to validate API key
	reqBody := fmt.Sprintf(`{"p_api_key": "%s", "p_service": "mcpist"}`, apiKey)
	req, err := http.NewRequest("POST", supabaseURL+"/rest/v1/rpc/validate_api_key", strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Authorization", "Bearer "+serviceKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to validate API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API key validation failed: status %d", resp.StatusCode)
	}

	var result []struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result) == 0 || result[0].UserID == "" {
		return "", fmt.Errorf("invalid API key")
	}

	log.Printf("Auth: API key validated")
	return result[0].UserID, nil
}

// validateJWT validates a JWT token locally using JWKS
func (m *Middleware) validateJWT(token string) (string, error) {
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
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", fmt.Errorf("failed to parse header: %w", err)
	}

	if header.Alg != "RS256" {
		return "", fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// Get public key from JWKS cache
	pubKey, err := m.jwksCache.GetKey(header.Kid)
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify signature
	if err := m.verifySignature(parts[0]+"."+parts[1], parts[2], pubKey); err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}

	// Decode and validate payload
	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims struct {
		Iss string `json:"iss"`
		Sub string `json:"sub"`
		Aud string `json:"aud"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	// Validate expiration
	if claims.Exp < time.Now().Unix() {
		return "", fmt.Errorf("token expired")
	}

	// Validate issuer (if configured)
	if m.issuer != "" && claims.Iss != m.issuer {
		return "", fmt.Errorf("invalid issuer")
	}

	// Validate audience (if configured)
	if m.audience != "" && claims.Aud != m.audience {
		return "", fmt.Errorf("invalid audience")
	}

	// Validate subject
	if claims.Sub == "" {
		return "", fmt.Errorf("missing subject claim")
	}

	log.Printf("Auth: JWT validated")
	return claims.Sub, nil
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
