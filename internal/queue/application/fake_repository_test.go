package application_test

import (
	"context"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/queue/domain"
)

type fakeQueueRepo struct {
	queues       map[string][]string // eventID → ordered userIDs
	tokens       map[string]string   // "eventID:userID" → token
	activeQueues map[string]bool
	rates        map[string]int64
}

func newFakeQueueRepo() *fakeQueueRepo {
	return &fakeQueueRepo{
		queues:       make(map[string][]string),
		tokens:       make(map[string]string),
		activeQueues: make(map[string]bool),
		rates:        make(map[string]int64),
	}
}

func (r *fakeQueueRepo) Join(_ context.Context, eventID, userID string) (int64, error) {
	q := r.queues[eventID]
	for i, id := range q {
		if id == userID {
			return int64(i), nil
		}
	}
	r.queues[eventID] = append(q, userID)
	return int64(len(r.queues[eventID]) - 1), nil
}

func (r *fakeQueueRepo) Position(_ context.Context, eventID, userID string) (int64, error) {
	for i, id := range r.queues[eventID] {
		if id == userID {
			return int64(i), nil
		}
	}
	return -1, nil
}

func (r *fakeQueueRepo) PopTop(_ context.Context, eventID string, n int64) ([]string, error) {
	q := r.queues[eventID]
	if int64(len(q)) < n {
		n = int64(len(q))
	}
	popped := q[:n]
	r.queues[eventID] = q[n:]
	return popped, nil
}

func (r *fakeQueueRepo) QueueSize(_ context.Context, eventID string) (int64, error) {
	return int64(len(r.queues[eventID])), nil
}

func (r *fakeQueueRepo) SetToken(_ context.Context, eventID, userID, token string, _ time.Duration) error {
	r.tokens[eventID+":"+userID] = token
	return nil
}

func (r *fakeQueueRepo) GetToken(_ context.Context, eventID, userID string) (string, error) {
	return r.tokens[eventID+":"+userID], nil
}

func (r *fakeQueueRepo) AddActiveQueue(_ context.Context, eventID string) error {
	r.activeQueues[eventID] = true
	return nil
}

func (r *fakeQueueRepo) GetActiveQueues(_ context.Context) ([]string, error) {
	var ids []string
	for id := range r.activeQueues {
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *fakeQueueRepo) RemoveActiveQueue(_ context.Context, eventID string) error {
	delete(r.activeQueues, eventID)
	return nil
}

func (r *fakeQueueRepo) IncrRequestRate(_ context.Context, eventID string) (int64, error) {
	r.rates[eventID]++
	return r.rates[eventID], nil
}

// fakeTokenSigner adalah TokenSigner deterministik untuk test.
type fakeTokenSigner struct {
	token     *domain.QueueToken
	verifyErr error
}

func (s *fakeTokenSigner) Sign(t domain.QueueToken) (string, error) {
	return "signed:" + t.UserID + ":" + t.EventID, nil
}

func (s *fakeTokenSigner) Verify(_ string) (*domain.QueueToken, error) {
	if s.verifyErr != nil {
		return nil, s.verifyErr
	}
	return s.token, nil
}
