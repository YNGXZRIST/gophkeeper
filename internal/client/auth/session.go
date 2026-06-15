package auth

type Session struct {
	Login      string
	Access     Token
	Refresh    Token
	EncSalt    []byte
	WrappedDek []byte
}

func NewSession(login string, access Token, refresh Token, encSalt, wrappedDek []byte) *Session {
	return &Session{Login: login, Access: access, Refresh: refresh, EncSalt: encSalt, WrappedDek: wrappedDek}
}

// Credentials carries the raw values returned by the server that must be
// persisted as the current session: the login, the token pair and the
// encryption keys used to unwrap the user's data key.
type Credentials struct {
	Login        string
	AccessToken  string
	RefreshToken string
	EncSalt      []byte
	WrappedDek   []byte
}
