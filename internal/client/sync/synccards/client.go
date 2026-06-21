package synccards

import (
	"context"
	"gophkeeper/internal/client/sync/syncer"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	c cardv1.CardServiceClient
}

func NewClient(c cardv1.CardServiceClient) *Client {
	return &Client{c: c}
}

var _ syncer.Client = (*Client)(nil)

func (a *Client) Changes(ctx context.Context, since string) ([]syncer.Change, string, error) {
	req := &cardv1.ChangesRequest{}
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
			Data:    it.GetData(),
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
	req := &cardv1.AddRequest{}
	req.SetId(id)
	req.SetData(data)
	resp, err := a.c.Add(ctx, req)
	if err != nil {
		return 0, mapErr(err)
	}
	return resp.GetCard().GetVersion(), nil
}

func (a *Client) Update(ctx context.Context, id string, data []byte, version int64) (int64, error) {
	req := &cardv1.UpdateRequest{}
	req.SetId(id)
	req.SetData(data)
	req.SetVersion(version)
	resp, err := a.c.Update(ctx, req)
	if err != nil {
		return 0, mapErr(err)
	}
	return resp.GetCard().GetVersion(), nil
}

func (a *Client) Delete(ctx context.Context, id string, version int64) error {
	req := &cardv1.DeleteRequest{}
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
