package domain_test

import (
	"testing"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/inventory/domain"
)

func TestTicketUnit_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from domain.TicketUnitStatus
		to   domain.TicketUnitStatus
		want bool
	}{
		{"available → held", domain.StatusAvailable, domain.StatusHeld, true},
		{"available → confirmed (lompat status)", domain.StatusAvailable, domain.StatusConfirmed, false},
		{"available → available (idempoten)", domain.StatusAvailable, domain.StatusAvailable, false},
		{"held → payment_pending", domain.StatusHeld, domain.StatusPaymentPending, true},
		{"held → available (expiry rollback)", domain.StatusHeld, domain.StatusAvailable, true},
		{"held → confirmed (lompat)", domain.StatusHeld, domain.StatusConfirmed, false},
		{"payment_pending → confirmed", domain.StatusPaymentPending, domain.StatusConfirmed, true},
		{"payment_pending → refunded (lost seat)", domain.StatusPaymentPending, domain.StatusRefunded, true},
		{"confirmed → refunded", domain.StatusConfirmed, domain.StatusRefunded, true},
		{"confirmed → admitted", domain.StatusConfirmed, domain.StatusAdmitted, true},
		{"confirmed → held (mundur)", domain.StatusConfirmed, domain.StatusHeld, false},
		{"refunded → available", domain.StatusRefunded, domain.StatusAvailable, true},
		{"refunded → confirmed (tidak boleh)", domain.StatusRefunded, domain.StatusConfirmed, false},
		{"admitted → available (permanen)", domain.StatusAdmitted, domain.StatusAvailable, false},
		{"admitted → refunded (tidak bisa refund setelah masuk)", domain.StatusAdmitted, domain.StatusRefunded, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unit := &domain.TicketUnit{Status: tc.from}
			got := unit.CanTransitionTo(tc.to)
			if got != tc.want {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestTicketUnit_IsHoldExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute)
	future := time.Now().Add(5 * time.Minute)

	tests := []struct {
		name string
		unit domain.TicketUnit
		want bool
	}{
		{
			name: "held dan sudah expired",
			unit: domain.TicketUnit{Status: domain.StatusHeld, HeldUntil: &past},
			want: true,
		},
		{
			name: "held tapi belum expired",
			unit: domain.TicketUnit{Status: domain.StatusHeld, HeldUntil: &future},
			want: false,
		},
		{
			name: "held tapi held_until nil (data anomali)",
			unit: domain.TicketUnit{Status: domain.StatusHeld, HeldUntil: nil},
			want: false,
		},
		{
			name: "available (bukan held)",
			unit: domain.TicketUnit{Status: domain.StatusAvailable},
			want: false,
		},
		{
			name: "confirmed dengan held_until tersisa (tidak relevan)",
			unit: domain.TicketUnit{Status: domain.StatusConfirmed, HeldUntil: &past},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.unit.IsHoldExpired()
			if got != tc.want {
				t.Errorf("IsHoldExpired() = %v, want %v", got, tc.want)
			}
		})
	}
}
