package syncfiles

import (
	"context"
	"fmt"
	"gophkeeper/internal/client/sync/syncer"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	c filev1.FileServiceClient
}

func NewClient(c filev1.FileServiceClient) *Client {
	return &Client{c: c}
}

var _ syncer.Client = (*Client)(nil)

func (a *Client) Changes(ctx context.Context, since string) ([]syncer.Change, string, error) {
	req := &filev1.ChangesRequest{}
	req.SetSince(since)
	resp, err := a.c.Changes(ctx, req)
	if err != nil {
		return nil, since, err
	}
	items := resp.GetChanges()
	out := make([]syncer.Change, 0, len(items))
	var maxT time.Time
	for _, it := range items {
		out = append(out, syncer.Change{
			ID:      it.GetId(),
			Data:    it.GetMeta(),
			Version: it.GetVersion(),
			Deleted: it.GetDeleted(),
		})
		if ua := it.GetUpdatedAt(); ua != nil {
			if t := ua.AsTime(); t.After(maxT) {
				maxT = t
			}
		}
	}
	cursor := since
	if !maxT.IsZero() {
		cursor = maxT.UTC().Format(time.RFC3339Nano)
	}
	return out, cursor, nil
}

func (a *Client) Create(ctx context.Context, id string, data []byte) (int64, error) {
	return 0, fmt.Errorf("file create via sync is not supported")
}

func (a *Client) Update(ctx context.Context, id string, data []byte, version int64) (int64, error) {
	req := &filev1.UpdateMetaRequest{}
	req.SetId(id)
	req.SetMeta(data)
	req.SetVersion(version)
	resp, err := a.c.UpdateMeta(ctx, req)
	if err != nil {
		return 0, mapErr(err)
	}
	return resp.GetFile().GetVersion(), nil
}

func (a *Client) Delete(ctx context.Context, id string, version int64) error {
	req := &filev1.DeleteRequest{}
	req.SetId(id)
	_, err := a.c.Delete(ctx, req)
	return mapErr(err)
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if status.Code(err) == codes.Aborted {
		return syncer.ErrVersionConflict
	}
	return err
}
