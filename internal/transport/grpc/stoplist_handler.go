package grpc

import (
	"context"

	pb "github.com/wildberries-test-task-22-05-2026/internal/transport/grpc/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type StopListUsecase interface {
	AddWord(ctx context.Context, word string) error
	DeleteWord(ctx context.Context, word string) error
	GetAllWords(ctx context.Context) ([]string, error)
}

type StopListHandler struct {
	pb.UnimplementedStopListServiceServer
	stopListUC StopListUsecase
}

func NewStopListHandler(uc StopListUsecase) *StopListHandler {
	return &StopListHandler{stopListUC: uc}
}

func (h *StopListHandler) AddWord(ctx context.Context, req *pb.AddWordRequest) (*emptypb.Empty, error) {
	err := h.stopListUC.AddWord(ctx, req.Word)
	return &emptypb.Empty{}, err
}

func (h *StopListHandler) DeleteWord(ctx context.Context, req *pb.DeleteWordRequest) (*emptypb.Empty, error) {
	err := h.stopListUC.DeleteWord(ctx, req.Word)
	return &emptypb.Empty{}, err
}

func (h *StopListHandler) GetAllWords(ctx context.Context, _ *emptypb.Empty) (*pb.GetAllWordsResponse, error) {
	resp, err := h.stopListUC.GetAllWords(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetAllWordsResponse{
		Words: resp,
	}, nil
}
