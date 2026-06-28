package syncfiles

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/client/sync/syncer"
	filev1 "gophkeeper/internal/shared/proto/file/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeFileClient struct {
	filev1.FileServiceClient

	changesResp *filev1.ChangesResponse
	changesErr  error

	updateResp *filev1.UpdateMetaResponse
	updateErr  error
	gotUpdate  *filev1.UpdateMetaRequest

	deleteErr error
	gotDelete *filev1.DeleteRequest
}

func (f *fakeFileClient) Changes(_ context.Context, _ *filev1.ChangesRequest, _ ...grpc.CallOption) (*filev1.ChangesResponse, error) {
	return f.changesResp, f.changesErr
}

func (f *fakeFileClient) UpdateMeta(_ context.Context, in *filev1.UpdateMetaRequest, _ ...grpc.CallOption) (*filev1.UpdateMetaResponse, error) {
	f.gotUpdate = in
	return f.updateResp, f.updateErr
}

func (f *fakeFileClient) Delete(_ context.Context, in *filev1.DeleteRequest, _ ...grpc.CallOption) (*filev1.DeleteResponse, error) {
	f.gotDelete = in
	return &filev1.DeleteResponse{}, f.deleteErr
}

func TestChangesMapsMetaToDataAndCursor(t *testing.T) {
	t1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	t2 := time.Date(2026, 4, 3, 4, 5, 6, 0, time.UTC)

	c1 := &filev1.FileChange{}
	c1.SetId("f1")
	c1.SetMeta([]byte("a"))
	c1.SetVersion(2)
	c1.SetUpdatedAt(timestamppb.New(t1))

	c2 := &filev1.FileChange{}
	c2.SetId("f2")
	c2.SetMeta([]byte("b"))
	c2.SetVersion(5)
	c2.SetDeleted(true)
	c2.SetUpdatedAt(timestamppb.New(t2))

	resp := &filev1.ChangesResponse{}
	resp.SetChanges([]*filev1.FileChange{c1, c2})

	changes, cursor, err := NewClient(&fakeFileClient{changesResp: resp}).Changes(context.Background(), "old")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 2 || changes[0].ID != "f1" || string(changes[0].Data) != "a" || !changes[1].Deleted {
		t.Fatalf("changes = %+v", changes)
	}
	want := t2.UTC().Format(time.RFC3339Nano)
	if cursor != want {
		t.Fatalf("cursor = %q, want %q", cursor, want)
	}
}

func TestChangesEmptyKeepsSince(t *testing.T) {
	changes, cursor, err := NewClient(&fakeFileClient{changesResp: &filev1.ChangesResponse{}}).Changes(context.Background(), "s")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 0 || cursor != "s" {
		t.Fatalf("len=%d cursor=%q", len(changes), cursor)
	}
}

func TestChangesError(t *testing.T) {
	_, cursor, err := NewClient(&fakeFileClient{changesErr: status.Error(codes.Internal, "x")}).Changes(context.Background(), "s")
	if err == nil || cursor != "s" {
		t.Fatalf("err=%v cursor=%q", err, cursor)
	}
}

func TestCreateNotSupported(t *testing.T) {
	v, err := NewClient(&fakeFileClient{}).Create(context.Background(), "f1", []byte("d"))
	if err == nil {
		t.Fatal("expected not-supported error")
	}
	if v != 0 {
		t.Fatalf("version = %d, want 0", v)
	}
}

func TestUpdateCallsUpdateMeta(t *testing.T) {
	file := &filev1.File{}
	file.SetVersion(4)
	resp := &filev1.UpdateMetaResponse{}
	resp.SetFile(file)

	fc := &fakeFileClient{updateResp: resp}
	v, err := NewClient(fc).Update(context.Background(), "f1", []byte("meta"), 3)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v != 4 {
		t.Fatalf("version = %d, want 4", v)
	}
	if fc.gotUpdate.GetId() != "f1" || string(fc.gotUpdate.GetMeta()) != "meta" || fc.gotUpdate.GetVersion() != 3 {
		t.Fatalf("update req = %+v", fc.gotUpdate)
	}
}

func TestUpdateConflict(t *testing.T) {
	if _, err := NewClient(&fakeFileClient{updateErr: status.Error(codes.Aborted, "c")}).Update(context.Background(), "f1", nil, 1); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestDelete(t *testing.T) {
	fc := &fakeFileClient{}
	if err := NewClient(fc).Delete(context.Background(), "f1", 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if fc.gotDelete.GetId() != "f1" {
		t.Fatalf("req = %+v", fc.gotDelete)
	}
}

func TestDeleteConflict(t *testing.T) {
	if err := NewClient(&fakeFileClient{deleteErr: status.Error(codes.Aborted, "c")}).Delete(context.Background(), "f1", 1); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestMapErrPassthrough(t *testing.T) {
	other := status.Error(codes.NotFound, "nf")
	if err := NewClient(&fakeFileClient{deleteErr: other}).Delete(context.Background(), "f1", 1); err != other {
		t.Fatalf("err = %v, want passthrough", err)
	}
}
