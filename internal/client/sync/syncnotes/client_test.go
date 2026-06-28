package syncnotes

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/client/sync/syncer"
	notev1 "gophkeeper/internal/shared/proto/note/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeNoteClient struct {
	notev1.NoteServiceClient

	changesResp *notev1.ChangesResponse
	changesErr  error

	addResp *notev1.AddResponse
	addErr  error
	gotAdd  *notev1.AddRequest

	updateResp *notev1.UpdateResponse
	updateErr  error
	gotUpdate  *notev1.UpdateRequest

	deleteErr error
	gotDelete *notev1.DeleteRequest
}

func (f *fakeNoteClient) Changes(_ context.Context, _ *notev1.ChangesRequest, _ ...grpc.CallOption) (*notev1.ChangesResponse, error) {
	return f.changesResp, f.changesErr
}

func (f *fakeNoteClient) Add(_ context.Context, in *notev1.AddRequest, _ ...grpc.CallOption) (*notev1.AddResponse, error) {
	f.gotAdd = in
	return f.addResp, f.addErr
}

func (f *fakeNoteClient) Update(_ context.Context, in *notev1.UpdateRequest, _ ...grpc.CallOption) (*notev1.UpdateResponse, error) {
	f.gotUpdate = in
	return f.updateResp, f.updateErr
}

func (f *fakeNoteClient) Delete(_ context.Context, in *notev1.DeleteRequest, _ ...grpc.CallOption) (*notev1.DeleteResponse, error) {
	f.gotDelete = in
	return &notev1.DeleteResponse{}, f.deleteErr
}

func TestChangesMapsAndComputesCursor(t *testing.T) {
	t1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	t2 := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

	c1 := &notev1.NoteChange{}
	c1.SetId("n1")
	c1.SetData([]byte("a"))
	c1.SetVersion(2)
	c1.SetDeleted(false)
	c1.SetUpdatedAt(timestamppb.New(t1))

	c2 := &notev1.NoteChange{}
	c2.SetId("n2")
	c2.SetData([]byte("b"))
	c2.SetVersion(5)
	c2.SetDeleted(true)
	c2.SetUpdatedAt(timestamppb.New(t2))

	resp := &notev1.ChangesResponse{}
	resp.SetChanges([]*notev1.NoteChange{c1, c2})

	fc := &fakeNoteClient{changesResp: resp}
	changes, cursor, err := NewClient(fc).Changes(context.Background(), "old")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("len = %d, want 2", len(changes))
	}
	if changes[0].ID != "n1" || string(changes[0].Data) != "a" || changes[0].Version != 2 || changes[0].Deleted {
		t.Fatalf("change[0] = %+v", changes[0])
	}
	if !changes[1].Deleted {
		t.Fatal("change[1] must be deleted")
	}
	want := t2.UTC().Format(time.RFC3339Nano)
	if cursor != want {
		t.Fatalf("cursor = %q, want %q", cursor, want)
	}
}

func TestChangesEmptyKeepsSince(t *testing.T) {
	resp := &notev1.ChangesResponse{}
	fc := &fakeNoteClient{changesResp: resp}
	changes, cursor, err := NewClient(fc).Changes(context.Background(), "since-token")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("len = %d, want 0", len(changes))
	}
	if cursor != "since-token" {
		t.Fatalf("cursor = %q, want since-token", cursor)
	}
}

func TestChangesError(t *testing.T) {
	fc := &fakeNoteClient{changesErr: status.Error(codes.Internal, "boom")}
	_, cursor, err := NewClient(fc).Changes(context.Background(), "s")
	if err == nil {
		t.Fatal("expected error")
	}
	if cursor != "s" {
		t.Fatalf("cursor = %q, want s", cursor)
	}
}

func TestCreate(t *testing.T) {
	note := &notev1.Note{}
	note.SetVersion(9)
	resp := &notev1.AddResponse{}
	resp.SetNote(note)

	fc := &fakeNoteClient{addResp: resp}
	v, err := NewClient(fc).Create(context.Background(), "n1", []byte("data"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v != 9 {
		t.Fatalf("version = %d, want 9", v)
	}
	if fc.gotAdd.GetId() != "n1" || string(fc.gotAdd.GetData()) != "data" {
		t.Fatalf("add req = %+v", fc.gotAdd)
	}
}

func TestCreateError(t *testing.T) {
	fc := &fakeNoteClient{addErr: status.Error(codes.Internal, "x")}
	_, err := NewClient(fc).Create(context.Background(), "n1", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate(t *testing.T) {
	note := &notev1.Note{}
	note.SetVersion(4)
	resp := &notev1.UpdateResponse{}
	resp.SetNote(note)

	fc := &fakeNoteClient{updateResp: resp}
	v, err := NewClient(fc).Update(context.Background(), "n1", []byte("upd"), 3)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v != 4 {
		t.Fatalf("version = %d, want 4", v)
	}
	if fc.gotUpdate.GetId() != "n1" || string(fc.gotUpdate.GetData()) != "upd" || fc.gotUpdate.GetVersion() != 3 {
		t.Fatalf("update req = %+v", fc.gotUpdate)
	}
}

func TestUpdateConflict(t *testing.T) {
	fc := &fakeNoteClient{updateErr: status.Error(codes.Aborted, "conflict")}
	_, err := NewClient(fc).Update(context.Background(), "n1", nil, 1)
	if err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestDelete(t *testing.T) {
	fc := &fakeNoteClient{}
	if err := NewClient(fc).Delete(context.Background(), "n1", 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if fc.gotDelete.GetId() != "n1" {
		t.Fatalf("delete req = %+v", fc.gotDelete)
	}
}

func TestDeleteConflict(t *testing.T) {
	fc := &fakeNoteClient{deleteErr: status.Error(codes.Aborted, "c")}
	if err := NewClient(fc).Delete(context.Background(), "n1", 2); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestMapErrPassthrough(t *testing.T) {
	other := status.Error(codes.NotFound, "nf")
	fc := &fakeNoteClient{deleteErr: other}
	if err := NewClient(fc).Delete(context.Background(), "n1", 1); err != other {
		t.Fatalf("err = %v, want passthrough", err)
	}
}
