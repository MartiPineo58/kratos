package session_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kratos/session"
)

type mockPersister struct {
	getSessionFunc func(ctx context.Context, id string) (*session.Session, error)
}

func (m *mockPersister) GetSession(ctx context.Context, id string) (*session.Session, error) {
	return m.getSessionFunc(ctx, id)
}

type mockCache struct {
	getFunc func(ctx context.Context, id string) (*session.Session, error)
	setFunc func(ctx context.Context, id string, sess *session.Session, ttl time.Duration) error
}

func (m *mockCache) Get(ctx context.Context, id string) (*session.Session, error) {
	return m.getFunc(ctx, id)
}

func (m *mockCache) Set(ctx context.Context, id string, sess *session.Session, ttl time.Duration) error {
	return m.setFunc(ctx, id, sess, ttl)
}

func TestFetchSession_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	persister := &mockPersister{
		getSessionFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return &session.Session{ID: id, Active: true}, nil
		},
	}

	manager := session.NewManager(persister, nil)
	sess, err := manager.FetchSession(ctx, "session-123")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session, got %v", sess)
	}
}

func TestFetchSession_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	persister := &mockPersister{
		getSessionFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return &session.Session{ID: id, Active: true}, nil
		},
	}

	manager := session.NewManager(persister, nil)
	sess, err := manager.FetchSession(ctx, "session-123")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded error, got %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session, got %v", sess)
	}
}

func TestFetchSession_CacheNotPopulatedOnCancellation(t *testing.T) {
	persister := &mockPersister{
		getSessionFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return nil, context.Canceled
		},
	}

	cacheSetCalled := false
	cache := &mockCache{
		getFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return nil, errors.New("cache miss")
		},
		setFunc: func(ctx context.Context, id string, sess *session.Session, ttl time.Duration) error {
			cacheSetCalled = true
			return nil
		},
	}

	manager := session.NewManager(persister, cache)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sess, err := manager.FetchSession(ctx, "session-123")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session, got %v", sess)
	}
	if cacheSetCalled {
		t.Error("expected cache.Set not to be called on context cancellation")
	}
}
