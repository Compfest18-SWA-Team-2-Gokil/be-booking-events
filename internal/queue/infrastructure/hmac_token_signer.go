package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/domain"
)

// HMACTokenSigner mengimplementasikan TokenSigner dengan HMAC-SHA256.
// Format identik dengan HMACQRSigner: base64url(payload_json) + "." + hex(hmac)
type HMACTokenSigner struct {
	key []byte
}

func NewHMACTokenSigner(secret string) *HMACTokenSigner {
	return &HMACTokenSigner{key: []byte(secret)}
}

var _ application.TokenSigner = (*HMACTokenSigner)(nil)

func (s *HMACTokenSigner) Sign(token domain.QueueToken) (string, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("marshal token: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	sig := s.computeHMAC(encoded)
	return encoded + "." + sig, nil
}

func (s *HMACTokenSigner) Verify(tokenStr string) (*domain.QueueToken, error) {
	parts := strings.SplitN(tokenStr, ".", 2)
	if len(parts) != 2 {
		return nil, domain.ErrInvalidToken
	}

	encoded, sig := parts[0], parts[1]
	expected := s.computeHMAC(encoded)

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, domain.ErrInvalidToken
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	var t domain.QueueToken
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, domain.ErrInvalidToken
	}

	return &t, nil
}

func (s *HMACTokenSigner) computeHMAC(data string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
