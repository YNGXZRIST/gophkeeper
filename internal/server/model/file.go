package model

import "time"

type File struct {
	ID         string
	UserID     string
	Meta       []byte
	ChunkCount int
	Version    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type FileChunk struct {
	FileID string
	Idx    int
	Data   []byte
}
