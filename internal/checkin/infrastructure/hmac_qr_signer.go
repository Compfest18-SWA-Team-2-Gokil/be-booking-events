package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/domain"
)

// HMACQRSigner mengimplementasikan QRSigner dengan HMAC-SHA256.
// Format: base64url(payload_json) + "." + hex(hmac)
type HMACQRSigner struct {
	key []byte
}

func NewHMACQRSigner(secret string) *HMACQRSigner {
	return &HMACQRSigner{key: []byte(secret)}
}

var _ application.QRSigner = (*HMACQRSigner)(nil)

func (s *HMACQRSigner) Sign(payload domain.QRPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	sig := s.computeHMAC(encoded)
	return encoded + "." + sig, nil
}

func (s *HMACQRSigner) Verify(qrContent string) (*domain.QRPayload, error) {
	parts := strings.SplitN(qrContent, ".", 2)
	if len(parts) != 2 {
		return nil, domain.ErrInvalidSignature
	}

	encoded, sig := parts[0], parts[1]
	expected := s.computeHMAC(encoded)

	// Constant-time comparison untuk mencegah timing attack.
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, domain.ErrInvalidSignature
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, domain.ErrInvalidSignature
	}

	var p domain.QRPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, domain.ErrInvalidSignature
	}

	return &p, nil
}

func (s *HMACQRSigner) computeHMAC(data string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
