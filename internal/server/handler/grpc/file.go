package grpc

import (
	"context"
	"errors"
	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/file/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// maxChunkSize caps a single encrypted chunk: 1 MiB plaintext plus room for the
// encryption envelope. Larger frames are rejected before touching the database.
const maxChunkSize = 2 << 20

type FileServer struct {
	pb.UnimplementedFileServiceServer
	FileServerProp
}

func NewFileServer(prop FileServerProp) *FileServer {
	return &FileServer{FileServerProp: prop}
}

type FileServerProp struct {
	Service *service.FileService
	Logger  *zap.Logger
}

func (f *FileServer) Upload(stream pb.FileService_UploadServer) error {
	ctx := stream.Context()
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing user identity")
	}
	first, err := stream.Recv()
	if err != nil {
		return status.Error(codes.InvalidArgument, "expected file header")
	}
	header := first.GetHeader()
	if header == nil {
		return status.Error(codes.InvalidArgument, "first message must be a header")
	}
	next := func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		chunk := req.GetChunk()
		if chunk == nil {
			return nil, status.Error(codes.InvalidArgument, "expected a chunk")
		}
		if len(chunk.GetData()) > maxChunkSize {
			return nil, status.Error(codes.InvalidArgument, "chunk too large")
		}
		return chunk.GetData(), nil
	}
	id, err := f.Service.Create(ctx, userID, header.GetMeta(), int(header.GetChunkCount()), next)
	if err != nil {
		f.Logger.Error("file upload failed", zap.Error(err))
		return status.Error(codes.Internal, "internal error")
	}
	resp := &pb.UploadResponse{}
	resp.SetId(id)
	return stream.SendAndClose(resp)
}

func (f *FileServer) Download(in *pb.DownloadRequest, stream pb.FileService_DownloadServer) error {
	ctx := stream.Context()
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing user identity")
	}
	sendMeta := func(meta []byte) error {
		resp := &pb.DownloadResponse{}
		resp.SetMeta(meta)
		return stream.Send(resp)
	}
	sendChunk := func(idx int, data []byte) error {
		chunk := &pb.FileChunk{}
		chunk.SetIdx(int32(idx))
		chunk.SetData(data)
		resp := &pb.DownloadResponse{}
		resp.SetChunk(chunk)
		return stream.Send(resp)
	}
	err := f.Service.Download(ctx, userID, in.GetId(), sendMeta, sendChunk)
	if err != nil {
		if errors.Is(err, model.ErrFileNotFound) {
			return status.Error(codes.NotFound, "file not found")
		}
		f.Logger.Error("file download failed", zap.Error(err))
		return status.Error(codes.Internal, "internal error")
	}
	return nil
}

func (f *FileServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	files, err := f.Service.List(ctx, userID, in.GetLastId(), int(in.GetLimit()), int(in.GetOffset()))
	if err != nil {
		f.Logger.Error("file list failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	pbFiles := make([]*pb.File, 0, len(files))
	for _, file := range files {
		pbFiles = append(pbFiles, toPbFile(file))
	}
	resp := &pb.ListResponse{}
	resp.SetFiles(pbFiles)
	return resp, nil
}

func (f *FileServer) UpdateMeta(ctx context.Context, in *pb.UpdateMetaRequest) (*pb.UpdateMetaResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	file, err := f.Service.UpdateMeta(ctx, userID, in.GetId(), in.GetMeta(), in.GetVersion())
	if err != nil {
		if errors.Is(err, model.ErrVersionConflict) {
			return nil, status.Error(codes.Aborted, "file version conflict")
		}
		if errors.Is(err, model.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		f.Logger.Error("file update meta failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.UpdateMetaResponse{}
	resp.SetFile(toPbFile(file))
	return resp, nil
}

func (f *FileServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	if err := f.Service.Delete(ctx, userID, in.GetId()); err != nil {
		if errors.Is(err, model.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		f.Logger.Error("file delete failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.DeleteResponse{}, nil
}

func toPbFile(file *model.File) *pb.File {
	pbFile := &pb.File{}
	pbFile.SetId(file.ID)
	pbFile.SetMeta(file.Meta)
	pbFile.SetChunkCount(int32(file.ChunkCount))
	pbFile.SetVersion(file.Version)
	pbFile.SetCreatedAt(timestamppb.New(file.CreatedAt))
	pbFile.SetUpdatedAt(timestamppb.New(file.UpdatedAt))
	return pbFile
}
