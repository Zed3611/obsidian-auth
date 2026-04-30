package storage

import "errors"

var (
	ErrUserExists      = errors.New("User already exists")
	ErrUserNotFound    = errors.New("User not found")
	ErrSessionNotFound = errors.New("Session not found")
)
