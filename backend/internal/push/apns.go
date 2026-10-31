package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// APNsSender delivers iOS notifications. Empty credentials stub the send.
type APNsSender struct {
	keyID    string
	teamID   string
	bundleID string
	keyPath  string
	sandbox  bool
	client   *http.Client

	mu     sync.Mutex
	key    *ecdsa.PrivateKey
	bearer string
	exp    time.Time
}

func NewAPNs(cfg config.Config) *APNsSender {
	return &APNsSender{
		keyID:    cfg.APNSKeyID,
		teamID:   cfg.APNSTeamID,
		bundleID: cfg.APNSBundleID,
		keyPath:  cfg.APNSKeyPath,
		sandbox:  cfg.APNSUseSandbox,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *APNsSender) Platform() string { return PlatformIOS }

func (s *APNsSender) Configured() bool {
	return s.keyID != "" && s.teamID != "" && s.bundleID != "" && s.keyPath != ""
}

func (s *APNsSender) Send(ctx context.Context, token string, msg Message) error {
	if !s.Configured() {
		log.Printf("push stub: provider=apns token_len=%d title=%q", len(token), msg.Title)
		return nil
	}
	return s.send(ctx, token, msg)
}

func (s *APNsSender) send(ctx context.Context, token string, msg Message) error {
	bearer, err := s.authorization()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{"title": msg.Title, "body": msg.Body},
			"sound": "default",
		},
	})
	if err != nil {
		return err
	}
	host := "https://api.push.apple.com"
	if s.sandbox {
		host = "https://api.sandbox.push.apple.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/3/device/"+url.PathEscape(token), bytesReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+bearer)
	req.Header.Set("apns-topic", s.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("apns status %d: %s", res.StatusCode, body)
	}
	return nil
}

func (s *APNsSender) authorization() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bearer != "" && time.Now().Before(s.exp) {
		return s.bearer, nil
	}
	if s.key == nil {
		key, err := loadECPrivateKey(s.keyPath)
		if err != nil {
			return "", err
		}
		s.key = key
	}
	claims := jwt.MapClaims{"iss": s.teamID, "iat": time.Now().Unix()}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = s.keyID
	signed, err := tok.SignedString(s.key)
	if err != nil {
		return "", err
	}
	s.bearer = signed
	s.exp = time.Now().Add(50 * time.Minute)
	return signed, nil
}

func loadECPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("apns key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("apns key is not ECDSA")
	}
	return key, nil
}
