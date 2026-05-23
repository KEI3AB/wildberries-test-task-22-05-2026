package stoplist

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

type RedisStopList struct {
	redisClient rueidis.Client
}

func NewRedisRepository(client rueidis.Client) *RedisStopList {
	return &RedisStopList{redisClient: client}
}

func (r *RedisStopList) IsBlocked(ctx context.Context, token string) (bool, error) {
	cmd := r.redisClient.B().Get().Key("stopword:" + token).Cache()

	res := r.redisClient.DoCache(ctx, cmd, 24*time.Hour)

	if err := res.Error(); err != nil {
		if rueidis.IsRedisNil(err) { // не нашли по ключу, значит нет в стоп-листе
			return false, nil
		}

		return false, fmt.Errorf("redis error: %w", err)
	}

	return true, nil
}

func (r *RedisStopList) AddWord(ctx context.Context, word string) error {
	cmd := r.redisClient.B().Set().Key("stopword:" + word).Value("1").Build()

	if err := r.redisClient.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to add stopword: %w", err)
	}

	return nil
}

func (r *RedisStopList) DeleteWord(ctx context.Context, word string) error {
	cmd := r.redisClient.B().Del().Key("stopword:" + word).Build()

	if err := r.redisClient.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to remove stopword: %w", err)
	}

	return nil
}
