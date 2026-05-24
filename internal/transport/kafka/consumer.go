package kafka

//go:generate easyjson $GOFILE

import (
	"context"
	"log/slog"
	"sync"

	"github.com/mailru/easyjson"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/wildberries-test-task-22-05-2026/internal/domain"
)

//easyjson:json
type SearchEventPayload struct {
	SessionID       string `json:"session_id"`
	Timestamp       int64  `json:"timestamp"`
	NormalizedQuery string `json:"normalized_query"`
}

type TrendUseCase interface {
	AddSearchEvent(ctx context.Context, event domain.SearchEvent) error
}

type Consumer struct {
	client  *kgo.Client
	trendUC TrendUseCase
}

func NewConsumer(kafkaClient *kgo.Client, uc TrendUseCase) *Consumer {
	return &Consumer{
		client:  kafkaClient,
		trendUC: uc,
	}
}

func (tr *Consumer) Start(ctx context.Context) {
	for {
		fetches := tr.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			if ctx.Err() != nil {
				break
			}
			slog.Error("batch fetching error", "err", err)
			continue
		}

		iter := fetches.RecordIter()

		var wg sync.WaitGroup
		sem := make(chan struct{}, 256)

		for !iter.Done() {
			record := iter.Next()

			wg.Add(1)
			sem <- struct{}{}

			go func(rec *kgo.Record) {
				defer wg.Done()
				defer func() { <-sem }()

				var dto SearchEventPayload
				if err := easyjson.Unmarshal(rec.Value, &dto); err != nil {
					slog.Error("kafka DTO easyjson unmarshalling error", "err", err)
					return
				}

				err := tr.trendUC.AddSearchEvent(ctx, searchEventPayloadDTOToSearchEventDomain(dto))
				if err != nil {
					slog.Error("failed to process event", "err", err)
				}
			}(record)
		}
		wg.Wait()
	}
}

// мапер DTO в доменную структуру
func searchEventPayloadDTOToSearchEventDomain(dto SearchEventPayload) domain.SearchEvent {
	return domain.SearchEvent{
		SessionID:       dto.SessionID,
		Timestamp:       dto.Timestamp,
		NormalizedQuery: dto.NormalizedQuery,
	}
}
