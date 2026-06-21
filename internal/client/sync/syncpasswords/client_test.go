package syncpasswords

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/client/sync/syncer"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakePasswordClient struct {
	passwordv1.PasswordServiceClient

	changesResp *passwordv1.ChangesResponse
	changesErr  error

	addResp *passwordv1.AddResponse
	addErr  error
	gotAdd  *passwordv1.AddRequest

	updateResp *passwordv1.UpdateResponse
	updateErr  error
	gotUpdate  *passwordv1.UpdateRequest

	deleteErr error
	gotDelete *passwordv1.DeleteRequest
}

func (f *fakePasswordClient) Changes(_ context.Context, _ *passwordv1.ChangesRequest, _ ...grpc.CallOption) (*passwordv1.ChangesResponse, error) {
	return f.changesResp, f.changesErr
}

func (f *fakePasswordClient) Add(_ context.Context, in *passwordv1.AddRequest, _ ...grpc.CallOption) (*passwordv1.AddResponse, error) {
	f.gotAdd = in
	return f.addResp, f.addErr
}

func (f *fakePasswordClient) Update(_ context.Context, in *passwordv1.UpdateRequest, _ ...grpc.CallOption) (*passwordv1.UpdateResponse, error) {
	f.gotUpdate = in
	return f.updateResp, f.updateErr
}

func (f *fakePasswordClient) Delete(_ context.Context, in *passwordv1.DeleteRequest, _ ...grpc.CallOption) (*passwordv1.DeleteResponse, error) {
	f.gotDelete = in
	return &passwordv1.DeleteResponse{}, f.deleteErr
}

func TestChangesMapsAndComputesCursor(t *testing.T) {
	t1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	t2 := time.Date(2026, 6, 3, 4, 5, 6, 0, time.UTC)

	c1 := &passwordv1.PasswordChange{}
	c1.SetId("p1")
	c1.SetData([]byte("a"))
	c1.SetVersion(2)
	c1.SetUpdatedAt(timestamppb.New(t1))

	c2 := &passwordv1.PasswordChange{}
	c2.SetId("p2")
	c2.SetData([]byte("b"))
	c2.SetVersion(5)
	c2.SetDeleted(true)
	c2.SetUpdatedAt(timestamppb.New(t2))

	resp := &passwordv1.ChangesResponse{}
	resp.SetChanges([]*passwordv1.PasswordChange{c1, c2})

	changes, cursor, err := NewClient(&fakePasswordClient{changesResp: resp}).Changes(context.Background(), "old")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 2 || changes[0].ID != "p1" || !changes[1].Deleted {
		t.Fatalf("changes = %+v", changes)
	}
	want := t2.UTC().Format(time.RFC3339Nano)
	if cursor != want {
		t.Fatalf("cursor = %q, want %q", cursor, want)
	}
}

func TestChangesEmptyKeepsSince(t *testing.T) {
	changes, cursor, err := NewClient(&fakePasswordClient{changesResp: &passwordv1.ChangesResponse{}}).Changes(context.Background(), "s")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 0 || cursor != "s" {
		t.Fatalf("len=%d cursor=%q", len(changes), cursor)
	}
}

func TestChangesError(t *testing.T) {
	_, cursor, err := NewClient(&fakePasswordClient{changesErr: status.Error(codes.Internal, "x")}).Changes(context.Background(), "s")
	if err == nil || cursor != "s" {
		t.Fatalf("err=%v cursor=%q", err, cursor)
	}
}

func TestCreate(t *testing.T) {
	pw := &passwordv1.Password{}
	pw.SetVersion(9)
	resp := &passwordv1.AddResponse{}
	resp.SetPassword(pw)

	fc := &fakePasswordClient{addResp: resp}
	v, err := NewClient(fc).Create(context.Background(), "p1", []byte("d"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v != 9 || fc.gotAdd.GetId() != "p1" || string(fc.gotAdd.GetData()) != "d" {
		t.Fatalf("v=%d req=%+v", v, fc.gotAdd)
	}
}

func TestCreateError(t *testing.T) {
	if _, err := NewClient(&fakePasswordClient{addErr: status.Error(codes.Internal, "x")}).Create(context.Background(), "p1", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate(t *testing.T) {
	pw := &passwordv1.Password{}
	pw.SetVersion(4)
	resp := &passwordv1.UpdateResponse{}
	resp.SetPassword(pw)

	fc := &fakePasswordClient{updateResp: resp}
	v, err := NewClient(fc).Update(context.Background(), "p1", []byte("u"), 3)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v != 4 || fc.gotUpdate.GetId() != "p1" || fc.gotUpdate.GetVersion() != 3 {
		t.Fatalf("v=%d req=%+v", v, fc.gotUpdate)
	}
}

func TestUpdateConflict(t *testing.T) {
	if _, err := NewClient(&fakePasswordClient{updateErr: status.Error(codes.Aborted, "c")}).Update(context.Background(), "p1", nil, 1); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestDelete(t *testing.T) {
	fc := &fakePasswordClient{}
	if err := NewClient(fc).Delete(context.Background(), "p1", 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if fc.gotDelete.GetId() != "p1" {
		t.Fatalf("req = %+v", fc.gotDelete)
	}
}

func TestDeleteConflict(t *testing.T) {
	if err := NewClient(&fakePasswordClient{deleteErr: status.Error(codes.Aborted, "c")}).Delete(context.Background(), "p1", 1); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestMapErrPassthrough(t *testing.T) {
	other := status.Error(codes.NotFound, "nf")
	if err := NewClient(&fakePasswordClient{deleteErr: other}).Delete(context.Background(), "p1", 1); err != other {
		t.Fatalf("err = %v, want passthrough", err)
	}
}
