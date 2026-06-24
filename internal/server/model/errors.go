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

// ErrVersionConflict is returned when an update targets a stale version.
var ErrVersionConflict = errors.New("version conflict")

// ErrPasswordNotFound is returned when a password is missing or not owned by the user.
var ErrPasswordNotFound = errors.New("password not found")

// ErrNoteNotFound is returned when a note is missing or not owned by the user.
var ErrNoteNotFound = errors.New("note not found")

// ErrFileNotFound is returned when a file is missing or not owned by the user.
var ErrFileNotFound = errors.New("file not found")

// ErrEntryNotFound is returned when an entry is missing or not owned by the
// user. It is the unified not-found sentinel for the EntryRepo store.
var ErrEntryNotFound = errors.New("entry not found")

// ErrInternalServerError is returned when server have error
var ErrInternalServerError = errors.New("internal server error")
