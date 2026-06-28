package model

import "time"

// NoteData is the decrypted plaintext payload of a note secret,
// serialized and encrypted before it leaves the client.
type NoteData struct {
	Text string `json:"text"`
	Meta string `json:"meta"`
}

// Note is the full client-side note: server-held metadata plus the decrypted payload.
type Note struct {
	ID        string
	Data      NoteData
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
