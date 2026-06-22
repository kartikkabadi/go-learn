package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// CreateUser inserts a new user with the given email and password hash.
// Email comparison is case-insensitive (COLLATE NOCASE on the column).
func (s *SQLiteStore) CreateUser(email, passwordHash string) (User, error) {
	id := uuid.NewString()
	now := timeNowRFC3339()
	_, err := s.db.Exec(`
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES (?, ?, ?, ?)
	`, id, email, passwordHash, now)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: now}, nil
}

// GetUserByEmail returns the user with the given email, or nil if not found.
func (s *SQLiteStore) GetUserByEmail(email string) (*User, error) {
	var u User
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
func (s *SQLiteStore) GetUserByID(id string) (*User, error) {
	var u User
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
func (s *SQLiteStore) CreateSession(userID, expiresAt string) (Session, error) {
	token := uuid.NewString()
	now := timeNowRFC3339()
	_, err := s.db.Exec(`
		INSERT INTO sessions (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, token, userID, expiresAt, now)
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{Token: token, UserID: userID, ExpiresAt: expiresAt, CreatedAt: now}, nil
}

// GetSession returns the session with the given token if it exists and is unexpired.
func (s *SQLiteStore) GetSession(token string) (*Session, error) {
	var sess Session
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

// DeleteSession removes a session row, effectively logging the user out.
func (s *SQLiteStore) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
