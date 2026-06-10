package auth

type Session struct {
	Login   string
	Access  Token
	Refresh Token
}

func NewSession(login string, access Token, refresh Token) *Session {
	return &Session{Login: login, Access: access, Refresh: refresh}
}
