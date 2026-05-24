package stoplist

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redis/rueidis"
)

type RedisStopList struct {
	redisClient rueidis.Client
	mu          sync.RWMutex
	wordsCache  map[string]struct{}
}

func NewRedisRepository(client rueidis.Client) *RedisStopList {
	return &RedisStopList{
		redisClient: client,
		wordsCache:  make(map[string]struct{}),
	}
}

func (r *RedisStopList) StartSync(ctx context.Context) {
	r.syncCache(ctx)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.syncCache(ctx)
			}
		}
	}()
}

func (r *RedisStopList) IsBlocked(ctx context.Context, token string) (bool, error) {
	r.mu.RLock()
	_, blocked := r.wordsCache[token]
	r.mu.RUnlock()

	return blocked, nil
}

func (r *RedisStopList) AddWord(ctx context.Context, word string) error {
	cmd := r.redisClient.B().Set().Key("stopword:" + word).Value("1").Build()

	if err := r.redisClient.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to add stopword: %w", err)
	}

	r.mu.Lock()
	r.wordsCache[word] = struct{}{}
	r.mu.Unlock()

	return nil
}

func (r *RedisStopList) DeleteWord(ctx context.Context, word string) error {
	cmd := r.redisClient.B().Del().Key("stopword:" + word).Build()

	if err := r.redisClient.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to remove stopword: %w", err)
	}

	r.mu.Lock()
	delete(r.wordsCache, word)
	r.mu.Unlock()

	return nil
}

func (r *RedisStopList) GetAllWords(ctx context.Context) ([]string, error) {
	var words []string
	var cursor uint64 = 0

	for {
		cmd := r.redisClient.B().Scan().Cursor(cursor).Match("stopword:*").Count(1000).Build()

		resp, err := r.redisClient.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			return nil, fmt.Errorf("redis scan error: %w", err)
		}

		for _, key := range resp.Elements {
			words = append(words, strings.TrimPrefix(key, "stopword:"))
		}

		cursor = resp.Cursor
		if cursor == 0 {
			break // обошли базу
		}
	}

	slices.Sort(words)
	return words, nil
}

func (r *RedisStopList) syncCache(ctx context.Context) {
	words, err := r.GetAllWords(ctx)
	if err != nil {
		slog.Error("failed to sync stop words cache", slog.Any("err", err))
		return
	}

	newCache := make(map[string]struct{}, len(words))
	for _, w := range words {
		newCache[w] = struct{}{}
	}

	r.mu.Lock()
	r.wordsCache = newCache
	r.mu.Unlock()
}
