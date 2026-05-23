package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/wildberries-test-task-22-05-2026/internal/domain"
)

type TrendRepository interface {
	// Добавление поискового запроса в хранилище
	AddSearchEvent(ctx context.Context, event domain.SearchEvent) error
	// Чтение топ-N запросов за последние 5 минут
	GetTopNTrends(ctx context.Context, n int) ([]domain.TrendQuery, error)
}

type StopListChecker interface {
	IsBlocked(ctx context.Context, query string) (bool, error)
}

type TrendManager struct {
	repo    TrendRepository
	checker StopListChecker
}

func NewTrendManager(repo TrendRepository, checker StopListChecker) *TrendManager {
	return &TrendManager{
		repo:    repo,
		checker: checker,
	}
}

func (uc *TrendManager) AddSearchEvent(ctx context.Context, event domain.SearchEvent) error {
	if ok, err := uc.checker.IsBlocked(ctx, event.NormalizedQuery); ok {
		return nil
	} else if err != nil {
		slog.Error("stop list error", "err", err)
	}

	currTime := time.Now().UnixMilli()
	if (currTime-event.Timestamp)/60000 > 5 {
		return nil // Запрос старше 5 минут
	}

	return uc.repo.AddSearchEvent(ctx, event)
}

func (uc *TrendManager) GetTopNTrends(ctx context.Context, n int) ([]domain.TrendQuery, error) {
	return uc.repo.GetTopNTrends(ctx, n)
}
