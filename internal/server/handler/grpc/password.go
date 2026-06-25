package grpc

import (
	"context"
	"errors"
	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/password/v1"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PasswordServer struct {
	pb.UnimplementedPasswordServiceServer
	PasswordServerProp
}

type PasswordServerProp struct {
	Service *service.EntryService
	Logger  *slog.Logger
}

func NewPasswordServer(passwordServerProp PasswordServerProp) *PasswordServer {
	return &PasswordServer{PasswordServerProp: passwordServerProp}
}

func (p *PasswordServer) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	entry, err := p.Service.Add(ctx, userID, in.GetId(), in.GetData())
	if err != nil {
		p.Logger.Error("password add failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, model.ErrInternalServerError.Error())
	}
	resp := &pb.AddResponse{}
	resp.SetPassword(toPBPassword(entry))
	return resp, nil
}

func (p *PasswordServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	entry, err := p.Service.Get(ctx, userID, in.GetId())
	if err != nil {
		if errors.Is(err, model.ErrEntryNotFound) {
			return nil, status.Error(codes.NotFound, model.ErrPasswordNotFound.Error())
		}
		p.Logger.Error("password get failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, model.ErrInternalServerError.Error())
	}
	resp := &pb.GetResponse{}
	resp.SetPassword(toPBPassword(entry))
	return resp, nil
}

func (p *PasswordServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	passwordEntries, err := p.Service.List(ctx, userID, in.GetLastId(), int(in.GetLimit()), int(in.GetOffset()))
	if err != nil {
		p.Logger.Error("password list failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, model.ErrInternalServerError.Error())
	}
	pbPasswords := make([]*pb.Password, 0, len(passwordEntries))
	for _, pass := range passwordEntries {
		pbPasswords = append(pbPasswords, toPBPassword(pass))
	}
	resp := &pb.ListResponse{}
	resp.SetPasswords(pbPasswords)
	return resp, nil
}

func (p *PasswordServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	entry, err := p.Service.Update(ctx, userID, in.GetId(), in.GetData(), in.GetVersion())
	if err != nil {
		if errors.Is(err, model.ErrVersionConflict) {
			return nil, status.Error(codes.Aborted, "password version conflict")
		}
		p.Logger.Error("password update failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, model.ErrInternalServerError.Error())
	}
	resp := &pb.UpdateResponse{}
	resp.SetPassword(toPBPassword(entry))
	return resp, nil
}

func (p *PasswordServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	if err := p.Service.Delete(ctx, userID, in.GetId()); err != nil {
		if errors.Is(err, model.ErrEntryNotFound) {
			return nil, status.Error(codes.NotFound, model.ErrPasswordNotFound.Error())
		}
		p.Logger.Error("password delete failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, model.ErrInternalServerError.Error())
	}
	return &pb.DeleteResponse{}, nil
}

func (p *PasswordServer) Changes(ctx context.Context, in *pb.ChangesRequest) (*pb.ChangesResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	changes, err := p.Service.Changes(ctx, userID, in.GetSince())
	if err != nil {
		p.Logger.Error("password changes failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, model.ErrInternalServerError.Error())
	}
	pbChanges := make([]*pb.PasswordChange, 0, len(changes))
	for _, ch := range changes {
		pbChanges = append(pbChanges, toPBPasswordChange(ch))
	}
	resp := &pb.ChangesResponse{}
	resp.SetChanges(pbChanges)
	return resp, nil
}

func toPBPasswordChange(ch *model.EntryChange) *pb.PasswordChange {
	pbChange := &pb.PasswordChange{}
	pbChange.SetId(ch.ID)
	pbChange.SetData(ch.Data)
	pbChange.SetVersion(ch.Version)
	pbChange.SetDeleted(ch.Deleted)
	pbChange.SetUpdatedAt(timestamppb.New(ch.UpdatedAt))
	return pbChange
}

func toPBPassword(pass *model.Entry) *pb.Password {
	pbPass := &pb.Password{}
	pbPass.SetId(pass.ID)
	pbPass.SetData(pass.Data)
	pbPass.SetVersion(pass.Version)
	pbPass.SetCreatedAt(timestamppb.New(pass.CreatedAt))
	pbPass.SetUpdatedAt(timestamppb.New(pass.UpdatedAt))
	return pbPass
}
