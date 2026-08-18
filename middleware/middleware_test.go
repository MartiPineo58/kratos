package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"kratos/middleware"
	"kratos/session"
)

type mockSessionManager struct {
	fetchSessionFunc func(ctx context.Context, id string) (*session.Session, error)
}

func (m *mockSessionManager) FetchSession(ctx context.Context, id string) (*session.Session, error) {
	return m.fetchSessionFunc(ctx, id)
}

func TestSessionMiddleware_Success(t *testing.T) {
	manager := &mockSessionManager{
		fetchSessionFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return &session.Session{ID: id, Active: true, UserID: "user-123"}, nil
		},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := r.Context().Value("session").(*session.Session)
		if !ok || sess == nil {
			t.Error("session not found in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.SessionMiddleware(manager)(nextHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Session-ID", "session-123")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestSessionMiddleware_ContextCancelled(t *testing.T) {
	manager := &mockSessionManager{
		fetchSessionFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return nil, context.Canceled
		},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	mw := middleware.SessionMiddleware(manager)(nextHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Session-ID", "session-123")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != 499 {
		t.Errorf("expected status 499, got %d", rec.Code)
	}
}

func TestSessionMiddleware_ContextTimeout(t *testing.T) {
	manager := &mockSessionManager{
		fetchSessionFunc: func(ctx context.Context, id string) (*session.Session, error) {
			return nil, context.DeadlineExceeded
		},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	mw := middleware.SessionMiddleware(manager)(nextHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Session-ID", "session-123")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", rec.Code)
	}
}
