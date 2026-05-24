package stoplist

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/rueidis/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRedisStopList_IsBlocked(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		word          string
		setup         func(repo *RedisStopList)
		expectedBlock bool
	}{
		{
			name: "Слово есть в стоп-листе",
			word: "676767676767",
			setup: func(repo *RedisStopList) {
				repo.wordsCache["676767676767"] = struct{}{}
			},
			expectedBlock: true,
		},
		{
			name: "Слова нет в стоп-листе",
			word: "бурмалда",
			setup: func(repo *RedisStopList) {
			},
			expectedBlock: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock.NewClient(ctrl)

			repo := NewRedisRepository(mockClient)
			tc.setup(repo)

			blocked, err := repo.IsBlocked(ctx, tc.word)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedBlock, blocked)
		})
	}
}

func TestRedisStopList_AddWord(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		word        string
		mockSetup   func(mockClient *mock.Client)
		expectedErr bool
	}{
		{
			name: "Успешное добавление слова",
			word: "альтушка",
			mockSetup: func(mockClient *mock.Client) {
				mockClient.EXPECT().
					Do(ctx, mock.Match("SET", "stopword:альтушка", "1")).
					Return(mock.Result(mock.RedisString("OK")))
			},
			expectedErr: false,
		},
		{
			name: "Ошибка сети - РКН установил кривое вредоносное ПО на наши сервера",
			word: "дешевыеБилетыНаАвиасейлс",
			mockSetup: func(mockClient *mock.Client) {
				mockClient.EXPECT().
					Do(ctx, mock.Match("SET", "stopword:дешевыеБилетыНаАвиасейлс", "1")).
					Return(mock.ErrorResult(errors.New("timeout")))
			},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock.NewClient(ctrl)
			tc.mockSetup(mockClient)

			repo := NewRedisRepository(mockClient)
			err := repo.AddWord(ctx, tc.word)

			if tc.expectedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to add stopword")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRedisStopList_DeleteWord(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		word        string
		mockSetup   func(mockClient *mock.Client)
		expectedErr bool
	}{
		{
			name: "Успешное удаление слова",
			word: "тунгТунгТунгСахур",
			mockSetup: func(mockClient *mock.Client) {
				mockClient.EXPECT().
					Do(ctx, mock.Match("DEL", "stopword:тунгТунгТунгСахур")).
					Return(mock.Result(mock.RedisInt64(1)))
			},
			expectedErr: false,
		},
		{
			name: "Ошибка со стороны Redis при удалении",
			word: "израиль",
			mockSetup: func(mockClient *mock.Client) {
				mockClient.EXPECT().
					Do(ctx, mock.Match("DEL", "stopword:израиль")).
					Return(mock.ErrorResult(errors.New("connection timeout")))
			},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock.NewClient(ctrl)
			tc.mockSetup(mockClient)

			repo := NewRedisRepository(mockClient)
			err := repo.DeleteWord(ctx, tc.word)

			if tc.expectedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to remove stopword")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRedisStopList_GetAllWords(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		mockSetup     func(mockClient *mock.Client)
		expectedWords []string
		expectedErr   bool
	}{
		{
			name: "Успешное получение слов в один проход",
			mockSetup: func(m *mock.Client) {
				m.EXPECT().
					Do(ctx, mock.Match("SCAN", "0", "MATCH", "stopword:*", "COUNT", "1000")).
					Return(mock.Result(mock.RedisArray(
						mock.RedisString("0"),
						mock.RedisArray(
							mock.RedisString("stopword:эпштейн"),
							mock.RedisString("stopword:чудоОстров"),
						),
					)))
			},
			expectedWords: []string{"чудоОстров", "эпштейн"},
			expectedErr:   false,
		},
		{
			name: "Успешное получение слов в несколько проходов (пагинация курсора)",
			mockSetup: func(m *mock.Client) {
				gomock.InOrder(
					m.EXPECT().
						Do(ctx, mock.Match("SCAN", "0", "MATCH", "stopword:*", "COUNT", "1000")).
						Return(mock.Result(mock.RedisArray(
							mock.RedisString("42"),
							mock.RedisArray(mock.RedisString("stopword:SIX_SEVEN")),
						))),

					m.EXPECT().
						Do(ctx, mock.Match("SCAN", "42", "MATCH", "stopword:*", "COUNT", "1000")).
						Return(mock.Result(mock.RedisArray(
							mock.RedisString("0"),
							mock.RedisArray(mock.RedisString("stopword:SIX_NINE")),
						))),
				)
			},
			expectedWords: []string{"SIX_NINE", "SIX_SEVEN"},
			expectedErr:   false,
		},
		{
			name: "Ошибка сети - в Беларуси бобры перегрызли интернет-кабели",
			mockSetup: func(m *mock.Client) {
				m.EXPECT().
					Do(ctx, mock.Match("SCAN", "0", "MATCH", "stopword:*", "COUNT", "1000")).
					Return(mock.ErrorResult(errors.New("scan failed")))
			},
			expectedWords: nil,
			expectedErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock.NewClient(ctrl)
			tc.mockSetup(mockClient)

			repo := NewRedisRepository(mockClient)
			words, err := repo.GetAllWords(ctx)

			if tc.expectedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "redis scan error")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedWords, words)
			}
		})
	}
}
