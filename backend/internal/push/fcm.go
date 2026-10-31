package push

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// FCMSender delivers Android notifications over FCM HTTP v1.
// Empty project id or credentials path stubs the send; registration still works.
type FCMSender struct {
	projectID string
	credPath  string
	client    *http.Client

	mu        sync.Mutex
	account   *serviceAccount
	key       *rsa.PrivateKey
	access    string
	accessExp time.Time
}

type serviceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

func NewFCM(cfg config.Config) *FCMSender {
	return &FCMSender{
		projectID: cfg.FCMProjectID,
		credPath:  cfg.FCMCredentialsPath,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *FCMSender) Platform() string { return PlatformAndroid }

func (s *FCMSender) Configured() bool {
	return s.projectID != "" && s.credPath != ""
}

func (s *FCMSender) Send(ctx context.Context, token string, msg Message) error {
	if !s.Configured() {
		log.Printf("push stub: provider=fcm token_len=%d title=%q", len(token), msg.Title)
		return nil
	}
	return s.send(ctx, token, msg)
}

func (s *FCMSender) send(ctx context.Context, token string, msg Message) error {
	access, err := s.accessToken(ctx)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]string{
				"title": msg.Title,
				"body":  msg.Body,
			},
			"android": map[string]any{"priority": "HIGH"},
		},
	})
	if err != nil {
		return err
	}
	endpoint := "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(s.projectID) + "/messages:send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytesReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("fcm status %d: %s", res.StatusCode, body)
	}
	return nil
}

func (s *FCMSender) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.access != "" && time.Now().Before(s.accessExp) {
		return s.access, nil
	}
	if s.account == nil {
		raw, err := os.ReadFile(s.credPath)
		if err != nil {
			return "", err
		}
		var account serviceAccount
		if err := json.Unmarshal(raw, &account); err != nil {
			return "", err
		}
		if account.TokenURI == "" {
			account.TokenURI = "https://oauth2.googleapis.com/token"
		}
		key, err := parseRSAPrivateKey(account.PrivateKey)
		if err != nil {
			return "", err
		}
		s.account = &account
		s.key = key
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   s.account.ClientEmail,
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
		"aud":   s.account.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	assertion, err := tok.SignedString(s.key)
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.account.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("fcm oauth status %d", res.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("fcm oauth response")
	}
	ttl := time.Duration(out.ExpiresIn) * time.Second
	if ttl > time.Minute {
		ttl -= time.Minute
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	s.access = out.AccessToken
	s.accessExp = time.Now().Add(ttl)
	return s.access, nil
}

func parseRSAPrivateKey(pemText string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, fmt.Errorf("fcm private key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, err
		}
		return key, nil
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("fcm private key is not RSA")
	}
	return key, nil
}
