package grpc

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/file/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if s.changes == nil {
		return nil, nil
	}
	return s.changes(ctx, u, since)
}

type fileTxStub struct{}

func (fileTxStub) WithinTx(ctx context.Context, _ *sql.TxOptions, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newFileServer(repo fileRepoStub) *FileServer {
	return NewFileServer(FileServerProp{
		Service: service.NewFileService(repo, fileTxStub{}),
		Logger:  zap.NewNop(),
	})
}

type uploadStream struct {
	grpc.ServerStream
	ctx  context.Context
	reqs []*pb.UploadRequest
	idx  int
	resp *pb.UploadResponse
}

func (s *uploadStream) Context() context.Context { return s.ctx }

func (s *uploadStream) Recv() (*pb.UploadRequest, error) {
	if s.idx >= len(s.reqs) {
		return nil, io.EOF
	}
	r := s.reqs[s.idx]
	s.idx++
	return r, nil
}

func (s *uploadStream) SendAndClose(resp *pb.UploadResponse) error {
	s.resp = resp
	return nil
}

func headerReq(meta []byte, chunkCount int32) *pb.UploadRequest {
	h := &pb.FileHeader{}
	h.SetMeta(meta)
	h.SetChunkCount(chunkCount)
	r := &pb.UploadRequest{}
	r.SetHeader(h)
	return r
}

func chunkReq(data []byte) *pb.UploadRequest {
	c := &pb.FileChunk{}
	c.SetData(data)
	r := &pb.UploadRequest{}
	r.SetChunk(c)
	return r
}

type downloadStream struct {
	grpc.ServerStream
	ctx  context.Context
	sent []*pb.DownloadResponse
}

func (s *downloadStream) Context() context.Context { return s.ctx }

func (s *downloadStream) Send(resp *pb.DownloadResponse) error {
	s.sent = append(s.sent, resp)
	return nil
}

func TestFileServerUpload(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		err := srv.Upload(&uploadStream{ctx: context.Background()})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("missing header", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		stream := &uploadStream{
			ctx:  authed("u1"),
			reqs: []*pb.UploadRequest{chunkReq([]byte("a"))},
		}
		err := srv.Upload(stream)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
		}
	})

	t.Run("empty stream", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		err := srv.Upload(&uploadStream{ctx: authed("u1")})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
		}
	})

	t.Run("create error", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			createFile: func(_ context.Context, _ *model.User, _ []byte, _ int) (*model.File, error) {
				return nil, errors.New("db down")
			},
		})
		stream := &uploadStream{
			ctx:  authed("u1"),
			reqs: []*pb.UploadRequest{headerReq([]byte("meta"), 0)},
		}
		err := srv.Upload(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		var inserted int
		srv := newFileServer(fileRepoStub{
			createFile: func(_ context.Context, _ *model.User, _ []byte, _ int) (*model.File, error) {
				return &model.File{ID: "f1"}, nil
			},
			insertChunk: func(_ context.Context, _ string, _ int, _ []byte) error {
				inserted++
				return nil
			},
		})
		stream := &uploadStream{
			ctx: authed("u1"),
			reqs: []*pb.UploadRequest{
				headerReq([]byte("meta"), 2),
				chunkReq([]byte("a")),
				chunkReq([]byte("b")),
			},
		}
		if err := srv.Upload(stream); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stream.resp.GetId() != "f1" {
			t.Fatalf("id = %q, want f1", stream.resp.GetId())
		}
		if inserted != 2 {
			t.Fatalf("inserted = %d, want 2", inserted)
		}
	})
}

func TestFileServerDownload(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		err := srv.Download(&pb.DownloadRequest{}, &downloadStream{ctx: context.Background()})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			getMeta: func(_ context.Context, _ *model.User, _ string) (*model.File, error) {
				return nil, model.ErrFileNotFound
			},
		})
		err := srv.Download(&pb.DownloadRequest{}, &downloadStream{ctx: authed("u1")})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			getMeta: func(_ context.Context, _ *model.User, _ string) (*model.File, error) {
				return nil, errors.New("db down")
			},
		})
		err := srv.Download(&pb.DownloadRequest{}, &downloadStream{ctx: authed("u1")})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			getMeta: func(_ context.Context, _ *model.User, _ string) (*model.File, error) {
				return &model.File{ID: "f1", Meta: []byte("meta")}, nil
			},
			streamChunks: func(_ context.Context, _ string, fn func(idx int, data []byte) error) error {
				return fn(0, []byte("a"))
			},
		})
		stream := &downloadStream{ctx: authed("u1")}
		if err := srv.Download(&pb.DownloadRequest{}, stream); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stream.sent) != 2 {
			t.Fatalf("sent = %d, want 2 (meta + chunk)", len(stream.sent))
		}
		if string(stream.sent[0].GetMeta()) != "meta" {
			t.Fatalf("meta = %q, want meta", stream.sent[0].GetMeta())
		}
	})
}

func TestFileServerList(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		_, err := srv.List(context.Background(), &pb.ListRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.File, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.List(authed("u1"), &pb.ListRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.File, error) {
				return []*model.File{{ID: "f1"}, {ID: "f2"}}, nil
			},
		})
		resp, err := srv.List(authed("u1"), &pb.ListRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.GetFiles()) != 2 {
			t.Fatalf("files = %d, want 2", len(resp.GetFiles()))
		}
	})
}

func TestFileServerUpdateMeta(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		_, err := srv.UpdateMeta(context.Background(), &pb.UpdateMetaRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("version conflict", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			updateMeta: func(_ context.Context, _ *model.User, _ string, _ []byte, _ int64) (*model.File, error) {
				return nil, model.ErrVersionConflict
			},
		})
		_, err := srv.UpdateMeta(authed("u1"), &pb.UpdateMetaRequest{})
		if status.Code(err) != codes.Aborted {
			t.Fatalf("code = %v, want Aborted", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			updateMeta: func(_ context.Context, _ *model.User, _ string, _ []byte, _ int64) (*model.File, error) {
				return nil, model.ErrFileNotFound
			},
		})
		_, err := srv.UpdateMeta(authed("u1"), &pb.UpdateMetaRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			updateMeta: func(_ context.Context, _ *model.User, _ string, _ []byte, _ int64) (*model.File, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.UpdateMeta(authed("u1"), &pb.UpdateMetaRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			updateMeta: func(_ context.Context, _ *model.User, _ string, _ []byte, _ int64) (*model.File, error) {
				return &model.File{ID: "f1", Version: 2}, nil
			},
		})
		resp, err := srv.UpdateMeta(authed("u1"), &pb.UpdateMetaRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetFile().GetVersion() != 2 {
			t.Fatalf("version = %d, want 2", resp.GetFile().GetVersion())
		}
	})
}

func TestFileServerDelete(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		_, err := srv.Delete(context.Background(), &pb.DeleteRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return model.ErrFileNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return errors.New("db down") },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestFileServerChanges(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{})
		_, err := srv.Changes(context.Background(), &pb.ChangesRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.FileChange, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Changes(authed("u1"), &pb.ChangesRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newFileServer(fileRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.FileChange, error) {
				return []*model.FileChange{{ID: "f1"}}, nil
			},
		})
		resp, err := srv.Changes(authed("u1"), &pb.ChangesRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.GetChanges()) != 1 {
			t.Fatalf("changes = %d, want 1", len(resp.GetChanges()))
		}
	})
}
