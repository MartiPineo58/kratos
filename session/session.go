package session

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type Session struct {
	ID        string    `json:"id"`
	Active    bool      `json:"active"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    string    `json:"user_id"`
}

type Persister interface {
	GetSession(ctx context.Context, id string) (*Session, error)
}

type Cache interface {
	Get(ctx context.Context, id string) (*Session, error)
	Set(ctx context.Context, id string, sess *Session, ttl time.Duration) error
}

type Manager struct {
	persister Persister
	cache     Cache
}

func NewManager(persister Persister, cache Cache) *Manager {
	return &Manager{
		persister: persister,
		cache:     cache,
	}
}

func (m *Manager) FetchSession(ctx context.Context, id string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if m.cache != nil {
		sess, err := m.cache.Get(ctx, id)
		if err == nil && sess != nil {
			return sess, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	sess, err := m.persister.GetSession(ctx, id)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if m.cache != nil && sess != nil {
		if err := m.cache.Set(ctx, id, sess, 5*time.Minute); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
	}

	return sess, nil
}
