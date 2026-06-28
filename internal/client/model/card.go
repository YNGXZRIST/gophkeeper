package model

import "time"

// CardData is the decrypted plaintext payload of a debit-card secret,
// serialized and encrypted before it leaves the client.
type CardData struct {
	Number string `json:"number"`
	Holder string `json:"holder"`
	Expiry string `json:"expiry"`
	CVV    string `json:"cvv"`
	Meta   string `json:"meta"`
}

// Card is the full client-side card: server-held metadata plus the
// decrypted payload.
type Card struct {
	ID        string
	Data      CardData
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
