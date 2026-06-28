package model

import "time"

// FileMeta is the decrypted plaintext metadata of a stored file,
// serialized and encrypted before it leaves the client.
type FileMeta struct {
	Name string `json:"name"`
	Meta string `json:"meta"`
	Size int64  `json:"size"`
}

// File is the full client-side file: server-held metadata plus the decrypted FileMeta.
type File struct {
	ID         string
	Meta       FileMeta
	ChunkCount int
	Version    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
