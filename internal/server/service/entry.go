package service

import (
	"context"
	"gophkeeper/internal/server/model"
	"iter"
	"time"
)

// entryStore is the persistence contract EntryService needs: one table's worth
// of encrypted entries. The bound table is decided by the injected repository.
type entryStore interface {
	GetByUser(ctx context.Context, uid, lastID string, limit, offset int) iter.Seq2[*model.Entry, error]
	GetByID(ctx context.Context, uid, id string) (*model.Entry, error)
	Create(ctx context.Context, uid string, entry *model.Entry) (*model.Entry, error)
	Update(ctx context.Context, uid string, entry *model.Entry) (*model.Entry, error)
	Delete(ctx context.Context, uid, id string) error
	Changes(ctx context.Context, uid string, since time.Time) iter.Seq2[*model.EntryChange, error]
}

// EntryService is the business layer for one kind of secret. Notes, passwords
// and cards reuse it; the kind is determined by the repository it is built with.
type EntryService struct {
	repo entryStore
}

func NewEntryService(repo entryStore) *EntryService {
	return &EntryService{repo: repo}
}

// Add stores a new entry for the user.
func (s *EntryService) Add(ctx context.Context, userID, id string, data []byte) (*model.Entry, error) {
	return s.repo.Create(ctx, userID, &model.Entry{ID: id, Data: data})
}

// List returns a chunk of the user's entries, draining the repository's
// streaming iterator and stopping on the first error it surfaces.
func (s *EntryService) List(ctx context.Context, userID, lastID string, limit, offset int) ([]*model.Entry, error) {
	var entries []*model.Entry
	for entry, err := range s.repo.GetByUser(ctx, userID, lastID, limit, offset) {
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Get returns a single entry owned by the user.
func (s *EntryService) Get(ctx context.Context, userID, id string) (*model.Entry, error) {
	return s.repo.GetByID(ctx, userID, id)
}

// Update overwrites an entry owned by the user.
func (s *EntryService) Update(ctx context.Context, userID, id string, data []byte, version int64) (*model.Entry, error) {
	return s.repo.Update(ctx, userID, &model.Entry{ID: id, Data: data, Version: version})
}

// Delete removes an entry owned by the user.
func (s *EntryService) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, userID, id)
}

// Changes returns the user's entries changed since the given cursor (RFC3339, empty = all).
func (s *EntryService) Changes(ctx context.Context, userID, since string) ([]*model.EntryChange, error) {
	var t time.Time
	if since != "" {
		parsed, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return nil, err
		}
		t = parsed
	}
	var changes []*model.EntryChange
	for ch, err := range s.repo.Changes(ctx, userID, t) {
		if err != nil {
			return nil, err
		}
		changes = append(changes, ch)
	}
	return changes, nil
}
