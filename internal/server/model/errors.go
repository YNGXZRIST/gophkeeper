package model

import "errors"

// ErrLoginTaken is returned when a registration uses a login that already exists.
var ErrLoginTaken = errors.New("login already taken")

// ErrInvalidCredentials is returned when a login fails because the user does not
// exist or the password does not match.
var ErrInvalidCredentials = errors.New("invalid login or password")

// ErrInvalidRefreshToken is returned when a refresh token is unknown or expired.
var ErrInvalidRefreshToken = errors.New("invalid refresh token")
