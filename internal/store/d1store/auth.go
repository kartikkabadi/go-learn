//go:build js && wasm

package d1store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kartikkabadi/go-learn/internal/store"
)

// CreateUser inserts a new user with the given email and password hash.
func (s *Store) CreateUser(email, passwordHash string) (store.User, error) {
	id := uuid.NewString()
	now := timeNowRFC3339()
	_, err := s.db.Exec(`
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES (?, ?, ?, ?)
	`, id, email, passwordHash, now)
	if err != nil {
		return store.User{}, fmt.Errorf("create user: %w", err)
	}
	return store.User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: now}, nil
}

// GetUserByEmail returns the user with the given email, or nil if not found.
func (s *Store) GetUserByEmail(email string) (*store.User, error) {
	var u store.User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

// GetUserByID returns the user with the given ID, or nil if not found.
func (s *Store) GetUserByID(id string) (*store.User, error) {
	var u store.User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// CreateSession inserts a new session row for a user with the given expiry.
func (s *Store) CreateSession(userID, expiresAt string) (store.Session, error) {
	token := uuid.NewString()
	now := timeNowRFC3339()
	_, err := s.db.Exec(`
		INSERT INTO sessions (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, token, userID, expiresAt, now)
	if err != nil {
		return store.Session{}, fmt.Errorf("create session: %w", err)
	}
	return store.Session{Token: token, UserID: userID, ExpiresAt: expiresAt, CreatedAt: now}, nil
}

// GetSession returns the session with the given token if it exists and is unexpired.
func (s *Store) GetSession(token string) (*store.Session, error) {
	var sess store.Session
	err := s.db.QueryRow(`
		SELECT token, user_id, expires_at, created_at
		FROM sessions WHERE token = ? AND expires_at > datetime('now')
	`, token).Scan(&sess.Token, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &sess, nil
}

// DeleteSession removes a session row.
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func timeNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
