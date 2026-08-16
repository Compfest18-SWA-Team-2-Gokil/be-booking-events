package application_test

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/domain"
)

type fakeCheckinRepo struct {
	units       map[string]*domain.CheckinTicket
	admittedIDs map[string]bool
	admitErr    error
	assignments map[string]bool // "gateOpID:eventID"
}

func newFakeCheckinRepo() *fakeCheckinRepo {
	return &fakeCheckinRepo{
		units:       make(map[string]*domain.CheckinTicket),
		admittedIDs: make(map[string]bool),
		assignments: make(map[string]bool),
	}
}

func (r *fakeCheckinRepo) GetConfirmedUnit(_ context.Context, unitID string) (*domain.CheckinTicket, error) {
	u, ok := r.units[unitID]
	if !ok {
		return nil, domain.ErrTicketNotConfirmed
	}
	return u, nil
}

func (r *fakeCheckinRepo) AdmitUnit(_ context.Context, unitID, _ string) error {
	if r.admitErr != nil {
		return r.admitErr
	}
	if r.admittedIDs[unitID] {
		return domain.ErrAlreadyAdmitted
	}
	r.admittedIDs[unitID] = true
	return nil
}

func (r *fakeCheckinRepo) IsGateOperatorAssigned(_ context.Context, gateOpID, eventID string) (bool, error) {
	return r.assignments[gateOpID+":"+eventID], nil
}

// fakeSigner adalah QRSigner deterministik untuk test: konten valid = validContent.
type fakeSigner struct {
	validContent string
	payload      *domain.QRPayload
	verifyErr    error
}

func (s *fakeSigner) Sign(p domain.QRPayload) (string, error) {
	return s.validContent, nil
}

func (s *fakeSigner) Verify(content string) (*domain.QRPayload, error) {
	if s.verifyErr != nil {
		return nil, s.verifyErr
	}
	if content != s.validContent {
		return nil, domain.ErrInvalidSignature
	}
	return s.payload, nil
}
