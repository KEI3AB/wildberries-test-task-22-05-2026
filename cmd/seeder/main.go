package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	totalMessages = 1_000_000
	uniqueWords   = 10_000
)

func main() {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("127.0.0.1:9092"),
		kgo.AllowAutoTopicCreation(),
		kgo.ProducerLinger(5*time.Millisecond),
	)
	if err != nil {
		slog.Error("kafka connect error", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	slog.Info("waiting for startup")

	if err := client.Ping(pingCtx); err != nil {
		slog.Error("something went wrong with kafka", slog.Any("err", err))
		os.Exit(1)
	}
	slog.Info("kafka is ready")

	ctx := context.Background()
	var wg sync.WaitGroup

	slog.Info("started to seed messages", slog.Int("count", totalMessages))

	for i := 0; i < totalMessages; i++ {
		wg.Add(1)

		word := fmt.Sprintf("query_%d", rand.IntN(uniqueWords))

		payload := []byte(fmt.Sprintf(`{"session_id":"seed_%d","timestamp":%d,"normalized_query":"%s"}`,
			i, time.Now().UnixMilli(), word))

		record := &kgo.Record{
			Topic: "trend-search-events",
			Value: payload,
		}

		client.Produce(ctx, record, func(_ *kgo.Record, err error) {
			defer wg.Done()
			if err != nil {
				slog.Error("produce error", slog.Any("err", err))
			}
		})
	}

	wg.Wait()
	slog.Info("successfull data seed to kafka")
}
