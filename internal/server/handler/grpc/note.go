package grpc

import (
	"context"
	"errors"
	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/note/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NoteServer struct {
	pb.UnimplementedNoteServiceServer
	NoteServerProp
}

func NewNoteServer(noteServiceProp NoteServerProp) *NoteServer {
	return &NoteServer{NoteServerProp: noteServiceProp}
}

type NoteServerProp struct {
	Service *service.NoteService
	Logger  *zap.Logger
}

func (n *NoteServer) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	note, err := n.Service.Add(ctx, userID, in.GetData())
	if err != nil {
		n.Logger.Error("note add failed", zap.Error(err))
	}
	resp := &pb.AddResponse{}
	resp.SetNote(toPbNote(note))
	return resp, nil

}

func (n *NoteServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	note, err := n.Service.Get(ctx, userID, in.GetId())
	if err != nil {
		if errors.Is(err, model.ErrNoteNotFound) {
			return nil, status.Error(codes.NotFound, "note not found")
		}
		n.Logger.Error("note get failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.GetResponse{}
	resp.SetNote(toPbNote(note))
	return resp, nil
}

func (n *NoteServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	notes, err := n.Service.List(ctx, userID, in.GetLastId(), int(in.GetLimit()), int(in.GetOffset()))
	if err != nil {
		n.Logger.Error("note list failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	pbNotes := make([]*pb.Note, 0, len(notes))
	for _, note := range notes {
		pbNotes = append(pbNotes, toPbNote(note))
	}
	resp := &pb.ListResponse{}
	resp.SetNotes(pbNotes)
	return resp, nil
}

func (n *NoteServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	note, err := n.Service.Update(ctx, userID, in.GetId(), in.GetData(), in.GetVersion())
	if err != nil {
		if errors.Is(err, model.ErrVersionConflict) {
			return nil, status.Error(codes.Aborted, "note version conflict")
		}
		n.Logger.Error("note update failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.UpdateResponse{}
	resp.SetNote(toPbNote(note))
	return resp, nil
}

func (n *NoteServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	if err := n.Service.Delete(ctx, userID, in.GetId()); err != nil {
		if errors.Is(err, model.ErrNoteNotFound) {
			return nil, status.Error(codes.NotFound, "note not found")
		}
		n.Logger.Error("note delete failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.DeleteResponse{}, nil

}
func toPbNote(note *model.Note) *pb.Note {
	pbNote := &pb.Note{}
	pbNote.SetId(note.ID)
	pbNote.SetData(note.Data)
	pbNote.SetVersion(note.Version)
	pbNote.SetCreatedAt(timestamppb.New(note.CreatedAt))
	pbNote.SetUpdatedAt(timestamppb.New(note.UpdatedAt))
	return pbNote

}
