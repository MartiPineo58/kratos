package middleware

import (
	"context"
	"errors"
	"net/http"

	"kratos/session"
)

type SessionManager interface {
	FetchSession(ctx context.Context, id string) (*session.Session, error)
}

func SessionMiddleware(manager SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			sessionID := r.Header.Get("X-Session-ID")
			if sessionID == "" {
				http.Error(w, "Unauthorized: missing session ID", http.StatusUnauthorized)
				return
			}

			sess, err := manager.FetchSession(ctx, sessionID)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					w.WriteHeader(499)
					w.Write([]byte("Client Closed Request"))
					return
				}
				if errors.Is(err, context.DeadlineExceeded) {
					http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
					return
				}
				if errors.Is(err, session.ErrSessionNotFound) {
					http.Error(w, "Unauthorized: invalid session", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			ctx = context.WithValue(ctx, "session", sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
