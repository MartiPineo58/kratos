package sql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	kratos_sql "kratos/persistence/sql"
)

type mockDriver struct{}
type mockConn struct {
	shouldCancel bool
}
type mockStmt struct {
	shouldCancel bool
}
type mockRows struct {
	shouldCancel bool
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{shouldCancel: name == "cancel"}, nil
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{shouldCancel: c.shouldCancel}, nil
}

func (c *mockConn) Close() error { return nil }
func (c *mockConn) Begin() (driver.Tx, error) { return nil, nil }

func (s *mockStmt) Close() error { return nil }
func (s *mockStmt) NumInput() int { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRows{shouldCancel: s.shouldCancel}, nil
}

func (s *mockStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if s.shouldCancel {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &mockRows{shouldCancel: s.shouldCancel}, nil
}

func (r *mockRows) Columns() []string {
	return []string{"id", "active", "expires_at", "user_id"}
}

func (r *mockRows) Close() error { return nil }

func (r *mockRows) Next(dest []driver.Value) error {
	if r.shouldCancel {
		return context.Canceled
	}
	dest[0] = "session-123"
	dest[1] = true
	dest[2] = time.Now().Add(1 * time.Hour)
	dest[3] = "user-123"
	return nil
}

func init() {
	sql.Register("mock", &mockDriver{})
}

func TestSQLPersister_GetSession_ContextCancelled(t *testing.T) {
	db, err := sql.Open("mock", "cancel")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	persister := kratos_sql.NewSQLPersister(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sess, err := persister.GetSession(ctx, "session-123")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session, got %v", sess)
	}
}

func TestSQLPersister_GetSession_Success(t *testing.T) {
	db, err := sql.Open("mock", "success")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	persister := kratos_sql.NewSQLPersister(db)
	ctx := context.Background()

	sess, err := persister.GetSession(ctx, "session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if sess.ID != "session-123" || !sess.Active || sess.UserID != "user-123" {
		t.Errorf("unexpected session data: %+v", sess)
	}
}
