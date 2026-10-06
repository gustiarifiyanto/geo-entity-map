package model

import "errors"

var (
	// ErrNotFound is returned when an entity does not exist or its ID is malformed.
	ErrNotFound = errors.New("entity not found")
	// ErrUserNotFound is returned by storage when no user has the given email.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailTaken is returned when registering an email that already has an account.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidCredentials is returned when the email or password is wrong.
	// It deliberately does not say which one.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrUnauthenticated is returned when a session token is missing, unknown or expired.
	ErrUnauthenticated = errors.New("not logged in")
	// ErrPhotoNotFound is returned when a photo does not exist or its ID is malformed.
	ErrPhotoNotFound = errors.New("photo not found")
	// ErrUnsupportedPhoto is returned when an upload is not a JPEG, PNG or WebP image.
	ErrUnsupportedPhoto = errors.New("unsupported photo type")
	// ErrTooManyPhotos is returned when an entity already has MaxPhotosPerEntity photos.
	ErrTooManyPhotos = errors.New("too many photos")
	// ErrNoInstallation is returned for installation requests on an entity
	// whose type does not have the installation capability.
	ErrNoInstallation = errors.New("this entity type has no installation data")
)
