package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// Enabled controls whether authentication is enforced.
	// When false, all requests pass through (development mode).
	Enabled bool

	// APIKeys is a set of valid API keys for simple key-based auth.
	// Keys are passed via X-API-Key header or ?api_key= query parameter.
	APIKeys map[string]bool

	// JWTSecret is the HMAC-SHA256 secret for JWT verification.
	// When set, Bearer tokens are verified against this secret.
	JWTSecret []byte

	// JWTIssuer is the expected token issuer (iss claim).
	JWTIssuer string

	// PublicPaths are paths that don't require authentication.
	PublicPaths map[string]bool
}

// NewAuthConfig creates auth configuration from environment.
// Auth is disabled by default (development). Set AUTH_ENABLED=true to enable.
func NewAuthConfig(enabled bool, apiKeys []string, jwtSecret string, jwtIssuer string) *AuthConfig {
	keySet := make(map[string]bool, len(apiKeys))
	for _, k := range apiKeys {
		if k != "" {
			keySet[k] = true
		}
	}

	publicPaths := map[string]bool{
		"/health":  true,
		"/ready":   true,
		"/metrics": true,
	}

	var secret []byte
	if jwtSecret != "" {
		secret = []byte(jwtSecret)
	}

	return &AuthConfig{
		Enabled:     enabled,
		APIKeys:     keySet,
		JWTSecret:   secret,
		JWTIssuer:   jwtIssuer,
		PublicPaths: publicPaths,
	}
}

// Authenticate is middleware that enforces authentication on protected endpoints.
// It supports two authentication methods:
//  1. API Key: X-API-Key header or ?api_key= query parameter
//  2. JWT Bearer: Authorization: Bearer <token> (HMAC-SHA256)
//
// Public paths (health, ready, metrics) are always accessible.
func (m *Middleware) Authenticate(cfg *AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if disabled
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for public paths
			if cfg.PublicPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Try API key authentication
			if m.tryAPIKey(r, cfg) {
				next.ServeHTTP(w, r)
				return
			}

			// Try JWT Bearer authentication
			if m.tryJWT(r, cfg) {
				next.ServeHTTP(w, r)
				return
			}

			// Authentication failed
			w.Header().Set("WWW-Authenticate", `Bearer realm="cryptomarket", error="invalid_token"`)
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{
				Error: "authentication required: provide X-API-Key header or Bearer token",
			})
		})
	}
}

// tryAPIKey checks for a valid API key in the request.
func (m *Middleware) tryAPIKey(r *http.Request, cfg *AuthConfig) bool {
	if len(cfg.APIKeys) == 0 {
		return false
	}

	// Check X-API-Key header
	key := r.Header.Get("X-API-Key")
	if key == "" {
		// Check query parameter (for SSE connections that can't set headers)
		key = r.URL.Query().Get("api_key")
	}

	if key == "" {
		return false
	}

	return cfg.APIKeys[key]
}

// tryJWT verifies a JWT Bearer token using HMAC-SHA256.
func (m *Middleware) tryJWT(r *http.Request, cfg *AuthConfig) bool {
	if len(cfg.JWTSecret) == 0 {
		return false
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return false
	}

	token := parts[1]
	claims, err := verifyJWT(token, cfg.JWTSecret)
	if err != nil {
		return false
	}

	// Verify issuer if configured
	if cfg.JWTIssuer != "" && claims.Issuer != cfg.JWTIssuer {
		return false
	}

	// Verify expiration
	if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
		return false
	}

	return true
}

// jwtClaims represents the JWT payload claims we verify.
type jwtClaims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// verifyJWT performs HMAC-SHA256 verification of a compact JWT.
// Format: header.payload.signature (base64url encoded)
func verifyJWT(token string, secret []byte) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format: expected 3 parts, got %d", len(parts))
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	expectedSig := computeHMACSHA256([]byte(signingInput), secret)

	actualSig, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Decode payload
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return &claims, nil
}

// computeHMACSHA256 computes the HMAC-SHA256 of data with the given key.
func computeHMACSHA256(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// base64URLDecode decodes a base64url string (no padding).
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode base64url: %w", err)
	}
	return decoded, nil
}
