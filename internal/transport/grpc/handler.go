package grpc

import (
	"context"

	"github.com/wildberries-test-task-22-05-2026/internal/domain"
	pb "github.com/wildberries-test-task-22-05-2026/internal/transport/grpc/pb"
)

const (
	DefaultMinLimit = 10
	DefaultMaxLimit = 100
)

type TrendUseCase interface {
	GetTopNTrends(ctx context.Context, n int) ([]domain.TrendQuery, error)
}

type TrendHandler struct {
	pb.UnimplementedTrendServiceServer
	trendUC TrendUseCase
}

func NewTrendHandler(uc TrendUseCase) *TrendHandler {
	return &TrendHandler{trendUC: uc}
}

func (h *TrendHandler) GetTopN(ctx context.Context, req *pb.GetTopNRequest) (*pb.GetTopNResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = DefaultMinLimit
	}
	if limit > DefaultMaxLimit {
		limit = DefaultMaxLimit
	}

	topTrends, err := h.trendUC.GetTopNTrends(ctx, limit)
	if err != nil {
		return nil, err
	}

	response := make([]*pb.TrendItem, 0, len(topTrends))
	for _, item := range topTrends {
		response = append(response, mapTrendQueryDomainToTrendItemPB(item))
	}

	return &pb.GetTopNResponse{
		Trends: response,
	}, nil
}

// мапер TrendQuery из доменного слоя в protobuf TrendItem
func mapTrendQueryDomainToTrendItemPB(tq domain.TrendQuery) *pb.TrendItem {
	return &pb.TrendItem{
		Query: tq.Query,
		Count: int32(tq.NumOfReq),
	}
}
