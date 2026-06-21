package grpc

import (
	"context"
	"errors"
	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/card/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CardServer struct {
	pb.UnimplementedCardServiceServer
	CardServerProp
}

func NewCardServer(cardServerProp CardServerProp) *CardServer {
	return &CardServer{CardServerProp: cardServerProp}
}

type CardServerProp struct {
	Service *service.CardService
	Logger  *zap.Logger
}

func (c *CardServer) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	card, err := c.Service.Add(ctx, userID, in.GetId(), in.GetData())
	if err != nil {
		c.Logger.Error("card add failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.AddResponse{}
	resp.SetCard(toPBCard(card))
	return resp, nil
}

func (c *CardServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	card, err := c.Service.Get(ctx, userID, in.GetId())
	if err != nil {
		if errors.Is(err, model.ErrCardNotFound) {
			return nil, status.Error(codes.NotFound, "card not found")
		}
		c.Logger.Error("card get failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.GetResponse{}
	resp.SetCard(toPBCard(card))
	return resp, nil
}

func (c *CardServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	cards, err := c.Service.List(ctx, userID, in.GetLastId(), int(in.GetLimit()), int(in.GetOffset()))
	if err != nil {
		c.Logger.Error("card list failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	pbCards := make([]*pb.Card, 0, len(cards))
	for _, card := range cards {
		pbCards = append(pbCards, toPBCard(card))
	}
	resp := &pb.ListResponse{}
	resp.SetCards(pbCards)
	return resp, nil
}

func (c *CardServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	card, err := c.Service.Update(ctx, userID, in.GetId(), in.GetData(), in.GetVersion())
	if err != nil {
		if errors.Is(err, model.ErrVersionConflict) {
			return nil, status.Error(codes.Aborted, "card version conflict")
		}
		c.Logger.Error("card update failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.UpdateResponse{}
	resp.SetCard(toPBCard(card))
	return resp, nil
}

func (c *CardServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	if err := c.Service.Delete(ctx, userID, in.GetId()); err != nil {
		if errors.Is(err, model.ErrCardNotFound) {
			return nil, status.Error(codes.NotFound, "card not found")
		}
		c.Logger.Error("card delete failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.DeleteResponse{}, nil
}

func (c *CardServer) Changes(ctx context.Context, in *pb.ChangesRequest) (*pb.ChangesResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
	}
	changes, err := c.Service.Changes(ctx, userID, in.GetSince())
	if err != nil {
		c.Logger.Error("card changes failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	pbChanges := make([]*pb.CardChange, 0, len(changes))
	for _, ch := range changes {
		pbChanges = append(pbChanges, toPbCardChange(ch))
	}
	resp := &pb.ChangesResponse{}
	resp.SetChanges(pbChanges)
	return resp, nil
}

func toPbCardChange(ch *model.CardChange) *pb.CardChange {
	pbChange := &pb.CardChange{}
	pbChange.SetId(ch.ID)
	pbChange.SetData(ch.Data)
	pbChange.SetVersion(ch.Version)
	pbChange.SetDeleted(ch.Deleted)
	pbChange.SetUpdatedAt(timestamppb.New(ch.UpdatedAt))
	return pbChange
}

func toPBCard(card *model.Card) *pb.Card {
	pbCard := &pb.Card{}
	pbCard.SetId(card.ID)
	pbCard.SetData(card.Data)
	pbCard.SetVersion(card.Version)
	pbCard.SetCreatedAt(timestamppb.New(card.CreatedAt))
	pbCard.SetUpdatedAt(timestamppb.New(card.UpdatedAt))
	return pbCard
}
