package model

import "errors"

// ErrNotFound is returned when an entity does not exist or its ID is malformed.
var ErrNotFound = errors.New("entity not found")
