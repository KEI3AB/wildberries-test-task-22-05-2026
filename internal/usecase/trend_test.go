package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wildberries-test-task-22-05-2026/internal/domain"
	"github.com/wildberries-test-task-22-05-2026/internal/usecase/mocks"
	"go.uber.org/mock/gomock"
)

func TestTrendManager_AddSearchEvent(t *testing.T) {
	ctx := context.Background()

	nowMilli := func() int64 { return time.Now().UnixMilli() }
	oldMilli := func() int64 { return time.Now().Add(-10 * time.Minute).UnixMilli() }

	tests := []struct {
		name        string
		setupEvent  func() domain.SearchEvent
		mockSetup   func(repo *mocks.MockTrendRepository, checker *mocks.MockStopListChecker, event domain.SearchEvent)
		expectedErr bool
	}{
		{
			name: "Успешное добавление (слово не в стоп-листе, запрос свежий)",
			setupEvent: func() domain.SearchEvent {
				return domain.SearchEvent{
					NormalizedQuery: "айфон 17",
					Timestamp:       nowMilli(),
				}
			},
			mockSetup: func(repo *mocks.MockTrendRepository, checker *mocks.MockStopListChecker, event domain.SearchEvent) {
				checker.EXPECT().IsBlocked(ctx, event.NormalizedQuery).Return(false, nil)
				repo.EXPECT().AddSearchEvent(ctx, event).Return(nil)
			},
			expectedErr: false,
		},
		{
			name: "Игнорирование запроса (слово найдено в стоп-листе)",
			setupEvent: func() domain.SearchEvent {
				return domain.SearchEvent{
					NormalizedQuery: "плохоеслово",
					Timestamp:       nowMilli(),
				}
			},
			mockSetup: func(repo *mocks.MockTrendRepository, checker *mocks.MockStopListChecker, event domain.SearchEvent) {
				checker.EXPECT().IsBlocked(ctx, event.NormalizedQuery).Return(true, nil)
			},
			expectedErr: false,
		},
		{
			name: "Игнорирование запроса (возраст запроса старше 5 минут)",
			setupEvent: func() domain.SearchEvent {
				return domain.SearchEvent{
					NormalizedQuery: "старый запрос",
					Timestamp:       oldMilli(),
				}
			},
			mockSetup: func(repo *mocks.MockTrendRepository, checker *mocks.MockStopListChecker, event domain.SearchEvent) {
				checker.EXPECT().IsBlocked(ctx, event.NormalizedQuery).Return(false, nil)
			},
			expectedErr: false,
		},
		{
			name: "Продолжение работы при ошибке стоп-листа",
			setupEvent: func() domain.SearchEvent {
				return domain.SearchEvent{
					NormalizedQuery: "нормальное слово",
					Timestamp:       nowMilli(),
				}
			},
			mockSetup: func(repo *mocks.MockTrendRepository, checker *mocks.MockStopListChecker, event domain.SearchEvent) {
				checker.EXPECT().IsBlocked(ctx, event.NormalizedQuery).Return(false, errors.New("checker error"))
				repo.EXPECT().AddSearchEvent(ctx, event).Return(nil)
			},
			expectedErr: false,
		},
		{
			name: "Ошибка при сохранении в репозиторий",
			setupEvent: func() domain.SearchEvent {
				return domain.SearchEvent{
					NormalizedQuery: "айфон",
					Timestamp:       nowMilli(),
				}
			},
			mockSetup: func(repo *mocks.MockTrendRepository, checker *mocks.MockStopListChecker, event domain.SearchEvent) {
				checker.EXPECT().IsBlocked(ctx, event.NormalizedQuery).Return(false, nil)
				repo.EXPECT().AddSearchEvent(ctx, event).Return(errors.New("db timeout"))
			},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockRepo := mocks.NewMockTrendRepository(ctrl)
			mockChecker := mocks.NewMockStopListChecker(ctrl)

			event := tc.setupEvent()
			tc.mockSetup(mockRepo, mockChecker, event)

			manager := NewTrendManager(mockRepo, mockChecker)
			err := manager.AddSearchEvent(ctx, event)

			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

}

func TestTrendManager_GetTopNTrends(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		n              int
		mockSetup      func(repo *mocks.MockTrendRepository)
		expectedResult []domain.TrendQuery
		expectedErr    bool
	}{
		{
			name: "Успешное получение топ трендов",
			n:    5,
			mockSetup: func(repo *mocks.MockTrendRepository) {
				expectedData := []domain.TrendQuery{
					{Query: "носки", NumOfReq: 100},
					{Query: "футболка", NumOfReq: 50},
				}
				repo.EXPECT().GetTopNTrends(ctx, 5).Return(expectedData, nil)
			},
			expectedResult: []domain.TrendQuery{
				{Query: "носки", NumOfReq: 100},
				{Query: "футболка", NumOfReq: 50},
			},
			expectedErr: false,
		},
		{
			name: "Ошибка от репозитория",
			n:    10,
			mockSetup: func(repo *mocks.MockTrendRepository) {
				repo.EXPECT().GetTopNTrends(ctx, 10).Return(nil, errors.New("db error"))
			},
			expectedResult: nil,
			expectedErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockTrendRepository(ctrl)
			mockChecker := mocks.NewMockStopListChecker(ctrl)

			tc.mockSetup(mockRepo)

			manager := NewTrendManager(mockRepo, mockChecker)
			res, err := manager.GetTopNTrends(ctx, tc.n)

			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResult, res)
			}
		})
	}
}
