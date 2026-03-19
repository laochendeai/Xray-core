package webpanel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// AuthManager handles JWT authentication.
type AuthManager struct {
	username string
	password string
	secret   []byte

	// Rate limiting for login attempts
	mu       sync.Mutex
	attempts map[string][]time.Time
}

// NewAuthManager creates a new AuthManager.
func NewAuthManager(username, password, jwtSecret string) *AuthManager {
	return &AuthManager{
		username: username,
		password: password,
		secret:   []byte(jwtSecret),
		attempts: make(map[string][]time.Time),
	}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// Login validates credentials and returns a JWT token.
func (a *AuthManager) Login(username, password, clientIP string) (string, error) {
	// Rate limiting: max 5 attempts per minute per IP
	a.mu.Lock()
	now := time.Now()
	attempts := a.attempts[clientIP]
	var recent []time.Time
	for _, t := range attempts {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	a.attempts[clientIP] = recent
	if len(recent) >= 5 {
		a.mu.Unlock()
		return "", fmt.Errorf("too many login attempts, please try again later")
	}
	a.attempts[clientIP] = append(recent, now)
	a.mu.Unlock()

	if username != a.username || password != a.password {
		return "", fmt.Errorf("invalid credentials")
	}

	return a.generateToken(username)
}

func (a *AuthManager) generateToken(username string) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	claims := jwtClaims{
		Sub: username,
		Iat: time.Now().Unix(),
		Exp: time.Now().Add(24 * time.Hour).Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// ValidateToken validates a JWT token and returns the username.
func (a *AuthManager) ValidateToken(tokenStr string) (string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid token signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token claims")
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", fmt.Errorf("invalid token claims")
	}

	// Check expiration
	if time.Now().Unix() > claims.Exp {
		return "", fmt.Errorf("token expired")
	}

	return claims.Sub, nil
}
