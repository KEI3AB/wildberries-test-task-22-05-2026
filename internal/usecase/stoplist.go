package usecase

import (
	"context"
	"errors"
	"strings"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/mock_$GOFILE -package=mocks

var (
	ErrInvalidStopWord = errors.New("stopword must be a single word without spaces")
	ErrEmptyStopWord   = errors.New("stopword cannot be empty")
)

type StopListRepository interface {
	IsBlocked(ctx context.Context, token string) (bool, error)
	AddWord(ctx context.Context, word string) error
	DeleteWord(ctx context.Context, word string) error
	GetAllWords(ctx context.Context) ([]string, error)
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

		blocked, err := uc.slRepo.IsBlocked(ctx, word)
		if blocked {
			return true, nil
		} else if err != nil {
			return false, err
		}
	}

	return false, nil
}

func (uc *StopListManager) AddWord(ctx context.Context, word string) error {
	validWord, err := validateWord(word)
	if err != nil {
		return err
	}
	return uc.slRepo.AddWord(ctx, validWord)
}

func (uc *StopListManager) DeleteWord(ctx context.Context, word string) error {
	validWord, err := validateWord(word)
	if err != nil {
		return err
	}
	return uc.slRepo.DeleteWord(ctx, validWord)
}

func (uc *StopListManager) GetAllWords(ctx context.Context) ([]string, error) {
	return uc.slRepo.GetAllWords(ctx)
}

func validateWord(word string) (string, error) {
	w := strings.TrimSpace(word)
	if w == "" {
		return "", ErrEmptyStopWord
	}
	if strings.ContainsRune(w, ' ') {
		return "", ErrInvalidStopWord
	}
	return w, nil
}
