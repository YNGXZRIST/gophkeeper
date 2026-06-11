package model

import "errors"

// ErrLoginTaken is returned when a registration uses a login that already exists.
var ErrLoginTaken = errors.New("login already taken")
