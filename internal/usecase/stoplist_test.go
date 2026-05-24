package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wildberries-test-task-22-05-2026/internal/usecase/mocks"
	"go.uber.org/mock/gomock"
)

func TestStopListManager_IsBlocked(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		mockSetup     func(repo *mocks.MockStopListRepository)
		expectedBlock bool
		expectedErr   bool
	}{
		{
			name:  "Запрос без стоп-слов",
			query: "купить свежие яблоки",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				gomock.InOrder(
					repo.EXPECT().IsBlocked(ctx, "купить").Return(false, nil),
					repo.EXPECT().IsBlocked(ctx, "свежие").Return(false, nil),
					repo.EXPECT().IsBlocked(ctx, "яблоки").Return(false, nil),
				)
			},
			expectedBlock: false,
			expectedErr:   false,
		},
		{
			name:  "Запрос содержит стоп-слово (прерывание поиска на втором слове)",
			query: "безобидное плохоеслово дальше не проверяем",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				gomock.InOrder(
					repo.EXPECT().IsBlocked(ctx, "безобидное").Return(false, nil),
					repo.EXPECT().IsBlocked(ctx, "плохоеслово").Return(true, nil),
				)
			},
			expectedBlock: true,
			expectedErr:   false,
		},
		{
			name:  "Запрос с двойными и начальными пробелами",
			query: "  одно   слово  ",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				gomock.InOrder(
					repo.EXPECT().IsBlocked(ctx, "одно").Return(false, nil),
					repo.EXPECT().IsBlocked(ctx, "слово").Return(false, nil),
				)
			},
			expectedBlock: false,
			expectedErr:   false,
		},
		{
			name:  "Ошибка от репозитория (БД устала и прилегла)",
			query: "тест",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().IsBlocked(ctx, "тест").Return(false, errors.New("timeout"))
			},
			expectedBlock: false,
			expectedErr:   true,
		},
		{
			name:          "Пустой запрос",
			query:         "   ",
			mockSetup:     func(repo *mocks.MockStopListRepository) {},
			expectedBlock: false,
			expectedErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockStopListRepository(ctrl)
			tc.mockSetup(mockRepo)

			manager := NewStopListManager(mockRepo)
			blocked, err := manager.IsBlocked(ctx, tc.query)

			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expectedBlock, blocked)
		})
	}
}

func TestStopListManager_AddWord(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		word        string
		mockSetup   func(repo *mocks.MockStopListRepository)
		expectedErr error
	}{
		{
			name: "Успешное добавление корректного слова",
			word: "запрещенка",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().AddWord(ctx, "запрещенка").Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "Слово с пробелами по краям (trim)",
			word: "  пробелы_по_бокам  ",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().AddWord(ctx, "пробелы_по_бокам").Return(nil)
			},
			expectedErr: nil,
		},
		{
			name:        "Пустое слово (после трима)",
			word:        "   ",
			mockSetup:   func(repo *mocks.MockStopListRepository) {},
			expectedErr: ErrEmptyStopWord,
		},
		{
			name:        "Слово содержит пробел внутри",
			word:        "два слова",
			mockSetup:   func(repo *mocks.MockStopListRepository) {},
			expectedErr: ErrInvalidStopWord,
		},
		{
			name: "Ошибка репозитория при добавлении",
			word: "ошибкабд",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().AddWord(ctx, "ошибкабд").Return(errors.New("db disconnected"))
			},
			expectedErr: errors.New("db disconnected"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockStopListRepository(ctrl)
			tc.mockSetup(mockRepo)

			manager := NewStopListManager(mockRepo)
			err := manager.AddWord(ctx, tc.word)

			if tc.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectedErr, ErrEmptyStopWord) || errors.Is(tc.expectedErr, ErrInvalidStopWord) {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStopListManager_DeleteWord(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		word        string
		mockSetup   func(repo *mocks.MockStopListRepository)
		expectedErr error
	}{
		{
			name: "Успешное удаление",
			word: "амнистия",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().DeleteWord(ctx, "амнистия").Return(nil)
			},
			expectedErr: nil,
		},
		{
			name:        "Попытка удалить некорректное слово (с пробелами)",
			word:        "нельзя удалять",
			mockSetup:   func(repo *mocks.MockStopListRepository) {},
			expectedErr: ErrInvalidStopWord,
		},
		{
			name:        "Попытка удалить пустое слово",
			word:        " ",
			mockSetup:   func(repo *mocks.MockStopListRepository) {},
			expectedErr: ErrEmptyStopWord,
		},
		{
			name: "Ошибка со стороны БД",
			word: "ошибка",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().DeleteWord(ctx, "ошибка").Return(errors.New("storage error"))
			},
			expectedErr: errors.New("storage error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockStopListRepository(ctrl)
			tc.mockSetup(mockRepo)

			manager := NewStopListManager(mockRepo)
			err := manager.DeleteWord(ctx, tc.word)

			if tc.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectedErr, ErrEmptyStopWord) || errors.Is(tc.expectedErr, ErrInvalidStopWord) {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStopListManager_GetAllWords(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		mockSetup     func(repo *mocks.MockStopListRepository)
		expectedWords []string
		expectedErr   bool
	}{
		{
			name: "Успешное получение списка",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().GetAllWords(ctx).Return([]string{"русскийрэп", "эпштейн", "дыня"}, nil)
			},
			expectedWords: []string{"русскийрэп", "эпштейн", "дыня"},
			expectedErr:   false,
		},
		{
			name: "Ошибка при получении списка",
			mockSetup: func(repo *mocks.MockStopListRepository) {
				repo.EXPECT().GetAllWords(ctx).Return(nil, errors.New("fail"))
			},
			expectedWords: nil,
			expectedErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockStopListRepository(ctrl)
			tc.mockSetup(mockRepo)

			manager := NewStopListManager(mockRepo)
			words, err := manager.GetAllWords(ctx)

			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedWords, words)
			}
		})
	}
}
