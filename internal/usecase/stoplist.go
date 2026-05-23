package usecase

import (
	"context"
	"strings"
)

type StopListRepository interface {
	IsBlocked(ctx context.Context, token string) (bool, error)
	AddWord(ctx context.Context, word string) error
	DeleteWord(ctx context.Context, word string) error
}

type StopListManager struct {
	slRepo StopListRepository
}

func NewStopListManager(repo StopListRepository) *StopListManager {
	return &StopListManager{slRepo: repo}
}

func (uc *StopListManager) IsBlocked(ctx context.Context, query string) (bool, error) {
	for len(query) > 0 {
		idx := strings.IndexByte(query, ' ')

		var word string
		if idx == -1 { // пробелов нет
			word = query
			query = ""
		} else {
			word = query[:idx]
			query = query[idx+1:]
		}

		if word == "" {
			continue // из-за двойного пробела может быть
		}

		blocked, err := uc.slRepo.IsBlocked(ctx, query[0:idx])
		if blocked {
			return true, nil
		} else if err != nil {
			return false, err
		}
	}

	return false, nil
}

func (uc *StopListManager) AddWord(ctx context.Context, word string) error {
	return uc.slRepo.AddWord(ctx, word)
}

func (uc *StopListManager) DeleteWord(ctx context.Context, word string) error {
	return uc.slRepo.DeleteWord(ctx, word)
}
