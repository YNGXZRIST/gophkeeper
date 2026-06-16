package model

// Card is the plaintext payload of a debit-card secret, serialized and
// encrypted before it leaves the client.
type Card struct {
	Number string `json:"number"`
	Holder string `json:"holder"`
	Expiry string `json:"expiry"`
	CVV    string `json:"cvv"`
	Meta   string `json:"meta"`
}
