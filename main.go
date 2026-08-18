package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"kratos/middleware"
	"kratos/session"
)

type dummyPersister struct{CustomField string}

func (d *dummyPersister) GetSession(ctx context.Context, id string) (*session.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &session.Session{
		ID:        id,
		Active:    true,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UserID:    "user-123",
	}, nil
}

func main() {
	persister := &dummyPersister{}
	manager := session.NewManager(persister, nil)
	mw := middleware.SessionMiddleware(manager)

	mux := http.NewServeMux()
	mux.Handle("/secure", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := r.Context().Value("session").(*session.Session)
		fmt.Fprintf(w, "Hello, user %s! Session %s is active.", sess.UserID, sess.ID)
	})))

	fmt.Println("Starting server on :8080...")
	_ = mux
}
