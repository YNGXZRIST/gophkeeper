package model

import "errors"

// ErrLoginTaken is returned when a registration uses a login that already exists.
var ErrLoginTaken = errors.New("login already taken")

// ErrInvalidCredentials is returned when a login fails because the user does not
// exist or the password does not match.
var ErrInvalidCredentials = errors.New("invalid login or password")

// ErrInvalidRefreshToken is returned when a refresh token is unknown or expired.
var ErrInvalidRefreshToken = errors.New("invalid refresh token")

// ErrCardNotFound is returned when a card is missing or not owned by the user.
var ErrCardNotFound = errors.New("card not found")

// ErrVersionConflict is returned when an update targets a stale card version.
var ErrVersionConflict = errors.New("card version conflict")
