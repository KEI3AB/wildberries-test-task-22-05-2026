package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wildberries-test-task-22-05-2026/internal/domain"
)

func TestRingBuffer_AddAndGetTopN(t *testing.T) {
	baseTime := time.Now().UnixMilli()

	tests := []struct {
		name           string
		events         []domain.SearchEvent
		topN           int
		expectedResult []domain.TrendQuery
	}{
		{
			name: "Успешная дедуплекация и выдача топа",
			events: []domain.SearchEvent{
				{SessionID: "user1", Timestamp: baseTime, NormalizedQuery: "apple"},
				{SessionID: "user1", Timestamp: baseTime, NormalizedQuery: "apple"}, // дубликат
				{SessionID: "user2", Timestamp: baseTime, NormalizedQuery: "apple"},

				{SessionID: "user3", Timestamp: baseTime, NormalizedQuery: "banana"},
			},
			topN: 2,
			expectedResult: []domain.TrendQuery{
				{Query: "apple", NumOfReq: 2},
				{Query: "banana", NumOfReq: 1},
			},
		},
		{
			name: "Успешная выдача топа при разных минутных отрезках",
			events: []domain.SearchEvent{
				// 4 минуты подряд
				{SessionID: "user1", Timestamp: baseTime, NormalizedQuery: "apple"},
				{SessionID: "user2", Timestamp: baseTime - 1*int64(time.Minute), NormalizedQuery: "apple"},
				{SessionID: "user3", Timestamp: baseTime - 2*int64(time.Minute), NormalizedQuery: "apple"},
				{SessionID: "user4", Timestamp: baseTime - 3*int64(time.Minute), NormalizedQuery: "apple"},
				// через 1 минуту
				{SessionID: "user2", Timestamp: baseTime - 1*int64(time.Minute), NormalizedQuery: "banana"},
				{SessionID: "user4", Timestamp: baseTime - 3*int64(time.Minute), NormalizedQuery: "banana"},
			},
			topN: 2,
			expectedResult: []domain.TrendQuery{
				{Query: "apple", NumOfReq: 4},
				{Query: "banana", NumOfReq: 2},
			},
		},
		{
			name: "Запросили больше, чем есть в базе вообще",
			events: []domain.SearchEvent{
				{SessionID: "user1", Timestamp: baseTime, NormalizedQuery: "apple"},
			},
			topN: 3,
			expectedResult: []domain.TrendQuery{
				{Query: "apple", NumOfReq: 1},
			},
		},
		{
			name:           "Пустой буфер",
			events:         []domain.SearchEvent{},
			topN:           5,
			expectedResult: []domain.TrendQuery{},
		},
		{
			name: "Сортировка при одинаковом количестве запросов",
			events: []domain.SearchEvent{
				{SessionID: "user1", Timestamp: baseTime, NormalizedQuery: "banana"},
				{SessionID: "user2", Timestamp: baseTime, NormalizedQuery: "apple"},
				{SessionID: "user3", Timestamp: baseTime, NormalizedQuery: "iphone"},
			},
			topN: 3,
			expectedResult: []domain.TrendQuery{
				{Query: "apple", NumOfReq: 1},
				{Query: "banana", NumOfReq: 1},
				{Query: "iphone", NumOfReq: 1},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buffer := NewRingBuffer()
			ctx := context.Background()

			for _, ev := range tc.events {
				err := buffer.AddSearchEvent(ctx, ev)
				require.NoError(t, err)
			}

			result, err := buffer.GetTopNTrends(ctx, tc.topN)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestRingBuffer_Concurrency(t *testing.T) {
	buffer := NewRingBuffer()
	ctx := context.Background()
	baseTime := time.Now().UnixMilli()

	goroutinesCount := 1000
	var wg sync.WaitGroup
	wg.Add(goroutinesCount)

	for i := 0; i < goroutinesCount; i++ {
		go func(sessionID int) {
			defer wg.Done()

			session := string(rune(sessionID))

			ev := domain.SearchEvent{
				SessionID:       session,
				Timestamp:       baseTime,
				NormalizedQuery: "iphone",
			}
			_ = buffer.AddSearchEvent(ctx, ev)
		}(i)
	}

	wg.Wait()

	result, err := buffer.GetTopNTrends(ctx, 1)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "iphone", result[0].Query)
	assert.Equal(t, goroutinesCount, result[0].NumOfReq)
}
