package synccards

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/client/sync/syncer"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeCardClient struct {
	cardv1.CardServiceClient

	changesResp *cardv1.ChangesResponse
	changesErr  error

	addResp *cardv1.AddResponse
	addErr  error
	gotAdd  *cardv1.AddRequest

	updateResp *cardv1.UpdateResponse
	updateErr  error
	gotUpdate  *cardv1.UpdateRequest

	deleteErr error
	gotDelete *cardv1.DeleteRequest
}

func (f *fakeCardClient) Changes(_ context.Context, _ *cardv1.ChangesRequest, _ ...grpc.CallOption) (*cardv1.ChangesResponse, error) {
	return f.changesResp, f.changesErr
}

func (f *fakeCardClient) Add(_ context.Context, in *cardv1.AddRequest, _ ...grpc.CallOption) (*cardv1.AddResponse, error) {
	f.gotAdd = in
	return f.addResp, f.addErr
}

func (f *fakeCardClient) Update(_ context.Context, in *cardv1.UpdateRequest, _ ...grpc.CallOption) (*cardv1.UpdateResponse, error) {
	f.gotUpdate = in
	return f.updateResp, f.updateErr
}

func (f *fakeCardClient) Delete(_ context.Context, in *cardv1.DeleteRequest, _ ...grpc.CallOption) (*cardv1.DeleteResponse, error) {
	f.gotDelete = in
	return &cardv1.DeleteResponse{}, f.deleteErr
}

func TestChangesMapsAndComputesCursor(t *testing.T) {
	t1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	t2 := time.Date(2026, 5, 3, 4, 5, 6, 0, time.UTC)

	c1 := &cardv1.CardChange{}
	c1.SetId("c1")
	c1.SetData([]byte("a"))
	c1.SetVersion(2)
	c1.SetUpdatedAt(timestamppb.New(t1))

	c2 := &cardv1.CardChange{}
	c2.SetId("c2")
	c2.SetData([]byte("b"))
	c2.SetVersion(5)
	c2.SetDeleted(true)
	c2.SetUpdatedAt(timestamppb.New(t2))

	resp := &cardv1.ChangesResponse{}
	resp.SetChanges([]*cardv1.CardChange{c1, c2})

	changes, cursor, err := NewClient(&fakeCardClient{changesResp: resp}).Changes(context.Background(), "old")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 2 || changes[0].ID != "c1" || !changes[1].Deleted {
		t.Fatalf("changes = %+v", changes)
	}
	want := t2.UTC().Format(time.RFC3339Nano)
	if cursor != want {
		t.Fatalf("cursor = %q, want %q", cursor, want)
	}
}

func TestChangesEmptyKeepsSince(t *testing.T) {
	changes, cursor, err := NewClient(&fakeCardClient{changesResp: &cardv1.ChangesResponse{}}).Changes(context.Background(), "s")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(changes) != 0 || cursor != "s" {
		t.Fatalf("len=%d cursor=%q", len(changes), cursor)
	}
}

func TestChangesError(t *testing.T) {
	_, cursor, err := NewClient(&fakeCardClient{changesErr: status.Error(codes.Internal, "x")}).Changes(context.Background(), "s")
	if err == nil || cursor != "s" {
		t.Fatalf("err=%v cursor=%q", err, cursor)
	}
}

func TestCreate(t *testing.T) {
	card := &cardv1.Card{}
	card.SetVersion(9)
	resp := &cardv1.AddResponse{}
	resp.SetCard(card)

	fc := &fakeCardClient{addResp: resp}
	v, err := NewClient(fc).Create(context.Background(), "c1", []byte("d"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v != 9 || fc.gotAdd.GetId() != "c1" || string(fc.gotAdd.GetData()) != "d" {
		t.Fatalf("v=%d req=%+v", v, fc.gotAdd)
	}
}

func TestCreateError(t *testing.T) {
	if _, err := NewClient(&fakeCardClient{addErr: status.Error(codes.Internal, "x")}).Create(context.Background(), "c1", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate(t *testing.T) {
	card := &cardv1.Card{}
	card.SetVersion(4)
	resp := &cardv1.UpdateResponse{}
	resp.SetCard(card)

	fc := &fakeCardClient{updateResp: resp}
	v, err := NewClient(fc).Update(context.Background(), "c1", []byte("u"), 3)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v != 4 || fc.gotUpdate.GetId() != "c1" || fc.gotUpdate.GetVersion() != 3 {
		t.Fatalf("v=%d req=%+v", v, fc.gotUpdate)
	}
}

func TestUpdateConflict(t *testing.T) {
	if _, err := NewClient(&fakeCardClient{updateErr: status.Error(codes.Aborted, "c")}).Update(context.Background(), "c1", nil, 1); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestDelete(t *testing.T) {
	fc := &fakeCardClient{}
	if err := NewClient(fc).Delete(context.Background(), "c1", 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if fc.gotDelete.GetId() != "c1" {
		t.Fatalf("req = %+v", fc.gotDelete)
	}
}

func TestDeleteConflict(t *testing.T) {
	if err := NewClient(&fakeCardClient{deleteErr: status.Error(codes.Aborted, "c")}).Delete(context.Background(), "c1", 1); err != syncer.ErrVersionConflict {
		t.Fatalf("err = %v, want ErrVersionConflict", err)
	}
}

func TestMapErrPassthrough(t *testing.T) {
	other := status.Error(codes.NotFound, "nf")
	if err := NewClient(&fakeCardClient{deleteErr: other}).Delete(context.Background(), "c1", 1); err != other {
		t.Fatalf("err = %v, want passthrough", err)
	}
}
