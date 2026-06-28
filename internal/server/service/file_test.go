package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
)

type fileRepoStub struct {
	createFile   func(context.Context, *model.User, []byte, int) (*model.File, error)
	insertChunk  func(context.Context, string, int, []byte) error
	getByUser    func(context.Context, *model.User, string, int, int) ([]*model.File, error)
	getMeta      func(context.Context, *model.User, string) (*model.File, error)
	streamChunks func(context.Context, string, func(idx int, data []byte) error) error
	updateMeta   func(context.Context, *model.User, string, []byte, int64) (*model.File, error)
	del          func(context.Context, *model.User, string) error
	changes      func(context.Context, *model.User, time.Time) ([]*model.FileChange, error)
}

func (s fileRepoStub) CreateFile(ctx context.Context, u *model.User, meta []byte, n int) (*model.File, error) {
	return s.createFile(ctx, u, meta, n)
}
func (s fileRepoStub) InsertChunk(ctx context.Context, fileID string, idx int, data []byte) error {
	return s.insertChunk(ctx, fileID, idx, data)
}
func (s fileRepoStub) GetByUser(ctx context.Context, u *model.User, lastID string, limit, offset int) ([]*model.File, error) {
	return s.getByUser(ctx, u, lastID, limit, offset)
}
func (s fileRepoStub) GetMeta(ctx context.Context, u *model.User, id string) (*model.File, error) {
	return s.getMeta(ctx, u, id)
}
func (s fileRepoStub) StreamChunks(ctx context.Context, fileID string, fn func(idx int, data []byte) error) error {
	return s.streamChunks(ctx, fileID, fn)
}
func (s fileRepoStub) UpdateMeta(ctx context.Context, u *model.User, id string, meta []byte, version int64) (*model.File, error) {
	return s.updateMeta(ctx, u, id, meta, version)
}
func (s fileRepoStub) Delete(ctx context.Context, u *model.User, id string) error {
	return s.del(ctx, u, id)
}
func (s fileRepoStub) Changes(ctx context.Context, u *model.User, since time.Time) ([]*model.FileChange, error) {
	return s.changes(ctx, u, since)
}

type txStub struct{}

func (txStub) WithinTx(ctx context.Context, _ *sql.TxOptions, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestFileServiceCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chunks := [][]byte{[]byte("a"), []byte("b")}
		var inserted [][]byte
		svc := NewFileService(fileRepoStub{
			createFile: func(_ context.Context, _ *model.User, _ []byte, _ int) (*model.File, error) {
				return &model.File{ID: "f1"}, nil
			},
			insertChunk: func(_ context.Context, _ string, _ int, data []byte) error {
				inserted = append(inserted, data)
				return nil
			},
		}, txStub{})

		i := 0
		next := func() ([]byte, error) {
			if i >= len(chunks) {
				return nil, io.EOF
			}
			c := chunks[i]
			i++
			return c, nil
		}

		id, err := svc.Create(context.Background(), "u1", []byte("meta"), 2, next)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "f1" {
			t.Errorf("id = %q, want f1", id)
		}
		if len(inserted) != 2 {
			t.Fatalf("inserted = %d chunks, want 2", len(inserted))
		}
	})

	t.Run("create error", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{
			createFile: func(_ context.Context, _ *model.User, _ []byte, _ int) (*model.File, error) {
				return nil, errors.New("db down")
			},
		}, txStub{})
		_, err := svc.Create(context.Background(), "u1", nil, 0, func() ([]byte, error) { return nil, io.EOF })
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("next error", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{
			createFile: func(_ context.Context, _ *model.User, _ []byte, _ int) (*model.File, error) {
				return &model.File{ID: "f1"}, nil
			},
		}, txStub{})
		_, err := svc.Create(context.Background(), "u1", nil, 1, func() ([]byte, error) {
			return nil, errors.New("read failed")
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("insert error", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{
			createFile: func(_ context.Context, _ *model.User, _ []byte, _ int) (*model.File, error) {
				return &model.File{ID: "f1"}, nil
			},
			insertChunk: func(_ context.Context, _ string, _ int, _ []byte) error {
				return errors.New("insert failed")
			},
		}, txStub{})
		_, err := svc.Create(context.Background(), "u1", nil, 1, func() ([]byte, error) {
			return []byte("a"), nil
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFileServiceList(t *testing.T) {
	want := []*model.File{{ID: "f1"}, {ID: "f2"}}
	var gotUser *model.User
	var gotLast string
	var gotLimit, gotOffset int
	svc := NewFileService(fileRepoStub{
		getByUser: func(_ context.Context, u *model.User, lastID string, limit, offset int) ([]*model.File, error) {
			gotUser, gotLast, gotLimit, gotOffset = u, lastID, limit, offset
			return want, nil
		},
	}, txStub{})

	got, err := svc.List(context.Background(), "u1", "f0", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if gotUser.ID != "u1" || gotLast != "f0" || gotLimit != 10 || gotOffset != 5 {
		t.Errorf("args = (%q,%q,%d,%d), want (u1,f0,10,5)", gotUser.ID, gotLast, gotLimit, gotOffset)
	}
}

func TestFileServiceDownload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var gotMeta []byte
		var gotChunks int
		svc := NewFileService(fileRepoStub{
			getMeta: func(_ context.Context, _ *model.User, _ string) (*model.File, error) {
				return &model.File{ID: "f1", Meta: []byte("meta")}, nil
			},
			streamChunks: func(_ context.Context, _ string, fn func(idx int, data []byte) error) error {
				return fn(0, []byte("a"))
			},
		}, txStub{})

		err := svc.Download(context.Background(), "u1", "f1",
			func(m []byte) error { gotMeta = m; return nil },
			func(_ int, _ []byte) error { gotChunks++; return nil },
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(gotMeta) != "meta" {
			t.Errorf("meta = %q, want meta", gotMeta)
		}
		if gotChunks != 1 {
			t.Errorf("chunks = %d, want 1", gotChunks)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{
			getMeta: func(_ context.Context, _ *model.User, _ string) (*model.File, error) {
				return nil, model.ErrFileNotFound
			},
		}, txStub{})
		err := svc.Download(context.Background(), "u1", "missing",
			func([]byte) error { return nil },
			func(int, []byte) error { return nil },
		)
		if !errors.Is(err, model.ErrFileNotFound) {
			t.Fatalf("err = %v, want ErrFileNotFound", err)
		}
	})

	t.Run("send meta error", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{
			getMeta: func(_ context.Context, _ *model.User, _ string) (*model.File, error) {
				return &model.File{ID: "f1"}, nil
			},
		}, txStub{})
		sendErr := errors.New("send failed")
		err := svc.Download(context.Background(), "u1", "f1",
			func([]byte) error { return sendErr },
			func(int, []byte) error { return nil },
		)
		if !errors.Is(err, sendErr) {
			t.Fatalf("err = %v, want sendErr", err)
		}
	})
}

func TestFileServiceUpdateMeta(t *testing.T) {
	want := &model.File{ID: "f1", Version: 2}
	var gotID string
	var gotVersion int64
	svc := NewFileService(fileRepoStub{
		updateMeta: func(_ context.Context, _ *model.User, id string, _ []byte, version int64) (*model.File, error) {
			gotID, gotVersion = id, version
			return want, nil
		},
	}, txStub{})

	got, err := svc.UpdateMeta(context.Background(), "u1", "f1", []byte("new"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("file = %v, want %v", got, want)
	}
	if gotID != "f1" || gotVersion != 1 {
		t.Errorf("args = (%q,%d), want (f1,1)", gotID, gotVersion)
	}
}

func TestFileServiceDelete(t *testing.T) {
	var gotUser *model.User
	var gotID string
	svc := NewFileService(fileRepoStub{
		del: func(_ context.Context, u *model.User, id string) error {
			gotUser, gotID = u, id
			return nil
		},
	}, txStub{})

	if err := svc.Delete(context.Background(), "u1", "f1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser.ID != "u1" || gotID != "f1" {
		t.Errorf("args = (%q,%q), want (u1,f1)", gotUser.ID, gotID)
	}
}

func TestFileServiceChanges(t *testing.T) {
	t.Run("empty since", func(t *testing.T) {
		var gotSince time.Time
		svc := NewFileService(fileRepoStub{
			changes: func(_ context.Context, _ *model.User, since time.Time) ([]*model.FileChange, error) {
				gotSince = since
				return []*model.FileChange{{ID: "f1"}}, nil
			},
		}, txStub{})
		got, err := svc.Changes(context.Background(), "u1", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotSince.IsZero() {
			t.Errorf("since = %v, want zero", gotSince)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
	})

	t.Run("valid since", func(t *testing.T) {
		want := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
		var gotSince time.Time
		svc := NewFileService(fileRepoStub{
			changes: func(_ context.Context, _ *model.User, since time.Time) ([]*model.FileChange, error) {
				gotSince = since
				return nil, nil
			},
		}, txStub{})
		_, err := svc.Changes(context.Background(), "u1", want.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotSince.Equal(want) {
			t.Errorf("since = %v, want %v", gotSince, want)
		}
	})

	t.Run("invalid since", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{}, txStub{})
		_, err := svc.Changes(context.Background(), "u1", "not-a-time")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := NewFileService(fileRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.FileChange, error) {
				return nil, errors.New("db down")
			},
		}, txStub{})
		_, err := svc.Changes(context.Background(), "u1", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
