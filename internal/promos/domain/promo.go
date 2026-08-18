package domain

import (
	"errors"
	"strings"
	"time"
)

type DiscountType string

const (
	DiscountTypePercentage  DiscountType = "PERCENTAGE"
	DiscountTypeFixedAmount DiscountType = "FIXED_AMOUNT"
)

type PromoType string

const (
	PromoTypeVoucher PromoType = "VOUCHER" // Global untuk seluruh event
	PromoTypePromo   PromoType = "PROMO"   // Spesifik untuk 1 konser/event tertentu
)

var (
	ErrPromoNotFound        = errors.New("kode promo/voucher tidak ditemukan")
	ErrPromoInactive        = errors.New("kode promo/voucher tidak aktif")
	ErrPromoNotStarted      = errors.New("voucher/promo belum dapat digunakan (periode belum dimulai)")
	ErrPromoExpired         = errors.New("voucher/promo sudah kedaluwarsa")
	ErrPromoUsageLimitExceed = errors.New("kuota penggunaan promo/voucher telah habis")
	ErrPromoMinOrderNotMet  = errors.New("nominal pesanan belum memenuhi syarat minimal")
	ErrPromoEventMismatch   = errors.New("promo ini hanya berlaku khusus untuk konser tertentu")
	ErrInvalidPromoCode     = errors.New("kode promo/voucher tidak valid")
	ErrInvalidDiscountValue = errors.New("nilai diskon tidak valid")
	ErrEventRequiredForPromo = errors.New("event wajib dipilih untuk tipe Promo Event")
)

type Promo struct {
	ID                string       `json:"id"`
	Code              string       `json:"code"`
	Title             string       `json:"title"`
	Description       string       `json:"description"`
	Type              PromoType    `json:"type"` // "VOUCHER" atau "PROMO"
	EventID           *string      `json:"event_id,omitempty"`
	EventName         string       `json:"event_name,omitempty"`
	DiscountType      DiscountType `json:"discount_type"`
	DiscountValue     int64        `json:"discount_value"`
	MinOrderAmount    int64        `json:"min_order_amount"`
	MaxDiscountAmount int64        `json:"max_discount_amount"`
	MaxUsage          int          `json:"max_usage"`
	UsedCount         int          `json:"used_count"`
	IsActive          bool         `json:"is_active"`
	StartDate         *time.Time   `json:"start_date,omitempty"`
	EndDate           *time.Time   `json:"end_date,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

func (p *Promo) Validate() error {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	if p.Code == "" {
		return ErrInvalidPromoCode
	}
	if p.DiscountValue <= 0 {
		return ErrInvalidDiscountValue
	}
	if p.DiscountType == DiscountTypePercentage && (p.DiscountValue > 100) {
		return ErrInvalidDiscountValue
	}
	if p.DiscountType != DiscountTypePercentage && p.DiscountType != DiscountTypeFixedAmount {
		return ErrInvalidDiscountValue
	}
	if p.Type == PromoTypePromo && (p.EventID == nil || strings.TrimSpace(*p.EventID) == "") {
		return ErrEventRequiredForPromo
	}
	if p.Type == PromoTypeVoucher {
		p.EventID = nil // Pastikan voucher murni global
	}
	if p.StartDate != nil && p.EndDate != nil && p.EndDate.Before(*p.StartDate) {
		return errors.New("waktu berakhir tidak boleh lebih awal dari waktu mulai")
	}
	return nil
}

// CalculateDiscountForEvent menghitung potongan harga dengan memeriksa waktu, kuota, dan kecocokan event.
func (p *Promo) CalculateDiscountForEvent(totalAmount int64, eventID string) (int64, error) {
	if !p.IsActive {
		return 0, ErrPromoInactive
	}

	now := time.Now()
	if p.StartDate != nil && now.Before(*p.StartDate) {
		return 0, ErrPromoNotStarted
	}
	if p.EndDate != nil && now.After(*p.EndDate) {
		return 0, ErrPromoExpired
	}

	if p.MaxUsage > 0 && p.UsedCount >= p.MaxUsage {
		return 0, ErrPromoUsageLimitExceed
	}

	if p.Type == PromoTypePromo {
		if p.EventID == nil || *p.EventID == "" || *p.EventID != eventID {
			return 0, ErrPromoEventMismatch
		}
	}

	if totalAmount < p.MinOrderAmount {
		return 0, ErrPromoMinOrderNotMet
	}

	var discount int64
	if p.DiscountType == DiscountTypePercentage {
		discount = (totalAmount * p.DiscountValue) / 100
		if p.MaxDiscountAmount > 0 && discount > p.MaxDiscountAmount {
			discount = p.MaxDiscountAmount
		}
	} else {
		discount = p.DiscountValue
	}

	if discount > totalAmount {
		discount = totalAmount
	}

	return discount, nil
}
