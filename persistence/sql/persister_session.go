package sql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"kratos/session"
)

type SQLPersister struct {
	db *sql.DB
}

func NewSQLPersister(db *sql.DB) *SQLPersister {
	return &SQLPersister{db: db}
}

func (p *SQLPersister) GetSession(ctx context.Context, id string) (*session.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var sess session.Session
	var expiresAt time.Time

	err := p.db.QueryRowContext(ctx, "SELECT id, active, expires_at, user_id FROM sessions WHERE id = ?", id).Scan(
		&sess.ID, &sess.Active, &expiresAt, &sess.UserID,
	)

	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, session.ErrSessionNotFound
		}
		return nil, err
	}

	sess.ExpiresAt = expiresAt

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &sess, nil
}
