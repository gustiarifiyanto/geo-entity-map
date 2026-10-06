package model

import "errors"

var (
	// ErrNotFound is returned when an entity does not exist or its ID is malformed.
	ErrNotFound = errors.New("entity not found")
	// ErrEmailTaken is returned when registering an email that already has an account.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidCredentials is returned when the email or password is wrong.
	// It deliberately does not say which one.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrUnauthenticated is returned when a session token is missing, unknown or expired.
	ErrUnauthenticated = errors.New("not logged in")
)
