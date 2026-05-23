package usecase

import (
	"context"

	"github.com/wildberries-test-task-22-05-2026/internal/domain"
)

type TrendRepository interface {
	// Добавление поискового запроса в хранилище
	AddSearchEvent(ctx context.Context, event domain.SearchEvent) error
	// Чтение топ-N запросов за последние 5 минут
	GetTopNTrends(ctx context.Context, n int) ([]domain.TrendQuery, error)
}

type TrendManager struct {
	repo TrendRepository
}
