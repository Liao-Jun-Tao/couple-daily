package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/golang-jwt/jwt/v5"
)

type accessClaims struct {
	Use string `json:"use"`
	jwt.RegisteredClaims
}

// Issuer signs short-lived access tokens.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

// Sign returns an HS256 access token for userID.
func (i *Issuer) Sign(userID string) (string, error) {
	now := time.Now().UTC()
	claims := accessClaims{
		Use: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

// ParseAccess reads a Bearer access token.
func ParseAccess(secret, header string) (string, error) {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", apierr.ErrUnauthorized
	}
	raw := strings.TrimSpace(header[len(prefix):])
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, apierr.ErrUnauthorized
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid || claims.Use != "access" || claims.Subject == "" {
		return "", apierr.ErrUnauthorized
	}
	return claims.Subject, nil
}

// Middleware requires a valid access token and stores the user id on the request.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := ParseAccess(secret, r.Header.Get("Authorization"))
			if err != nil {
				apierr.Write(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(httpapi.WithUserID(r.Context(), userID)))
		})
	}
}

// NewRefresh returns a raw refresh token and its sha256 hex digest.
func NewRefresh() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

// HashRefresh is the stored form of a refresh token.
func HashRefresh(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
