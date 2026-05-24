package memory

import (
	"cmp"
	"container/heap"
	"context"
	"hash/fnv"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wildberries-test-task-22-05-2026/internal/domain"
)

const numOfBuckets = 6
const bucketSize = 256

type Shard struct {
	mu       sync.RWMutex
	counters map[string]int       // топ запросов
	seen     map[seenKey]struct{} // ключ - session_id + query, мапка нужна для того, чтобы не записывать дубликаты
}

type Bucket struct {
	bucket [bucketSize]Shard
}

type RingBuffer struct {
	timeBuff [numOfBuckets]Bucket // Будем записывать поминутно, гуляя от остатка от деления
}

type seenKey struct {
	sessionID string
	query     string
}

func NewRingBuffer() *RingBuffer {
	buffer := &RingBuffer{}

	for i := range buffer.timeBuff {
		for j := range buffer.timeBuff[i].bucket {
			buffer.timeBuff[i].bucket[j].counters = make(map[string]int)
			buffer.timeBuff[i].bucket[j].seen = map[seenKey]struct{}{}
		}
	}

	return buffer
}

func (r *RingBuffer) StartCleaner(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return // кейс для graceful shutdown
			case t := <-ticker.C:
				currentMin := t.UnixMilli() / 60000
				clearIdx := (currentMin + 1) % numOfBuckets

				for i := 0; i < bucketSize; i++ {
					shard := &r.timeBuff[clearIdx].bucket[i]
					shard.mu.Lock()
					clear(shard.counters)
					clear(shard.seen)
					shard.mu.Unlock()
				}
			}
		}
	}()
}

func (r *RingBuffer) AddSearchEvent(ctx context.Context, event domain.SearchEvent) error {
	minutes := event.Timestamp / 60000 % numOfBuckets // конвертирую миллисекунды в минуты и нормирую к 6 бакетам
	idx := hashStringToByte(event.NormalizedQuery)

	bucket := &r.timeBuff[minutes].bucket[idx]
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	key := seenKey{sessionID: event.SessionID, query: event.NormalizedQuery}
	if _, ok := bucket.seen[key]; ok {
		// Значит уже записали
		return nil
	}

	bucket.counters[event.NormalizedQuery]++
	bucket.seen[key] = struct{}{}
	return nil
}

func (r *RingBuffer) GetTopNTrends(ctx context.Context, n int) ([]domain.TrendQuery, error) {
	h := &TopHeap{}
	heap.Init(h)

	// мапа для мержа одного шарда по всем минутам
	localMerge := make(map[string]int)
	currentMin := time.Now().UnixMilli() / 60000

	for i := 0; i < bucketSize; i++ {
		clear(localMerge)

		for k := int64(0); k < int64(numOfBuckets-1); k++ {
			targetMin := currentMin - k
			bucketIdx := targetMin % numOfBuckets
			if bucketIdx < 0 {
				bucketIdx += numOfBuckets
			}

			shard := &r.timeBuff[bucketIdx].bucket[i]
			shard.mu.RLock()

			// Увеличиваем счетчик по конкретному запросу
			for q, c := range shard.counters {
				localMerge[q] += c
			}
			shard.mu.RUnlock()
		}

		for query, count := range localMerge {
			if h.Len() < n { // куча не заполнена, заполняем
				heap.Push(h, domain.TrendQuery{Query: query, NumOfReq: count})
			} else if count > (*h)[0].NumOfReq { // куча заполнена, добавляем новый элемент только если минимальный из кучи меньше нового элемента
				heap.Pop(h)
				heap.Push(h, domain.TrendQuery{Query: query, NumOfReq: count})
			}
		}
	}

	result := make([]domain.TrendQuery, 0, h.Len())

	for h.Len() > 0 {
		result = append(result, heap.Pop(h).(domain.TrendQuery))
	}

	slices.Reverse(result)

	slices.SortFunc(result, func(a, b domain.TrendQuery) int {
		if a.NumOfReq != b.NumOfReq {
			return cmp.Compare(b.NumOfReq, a.NumOfReq)
		}
		return strings.Compare(a.Query, b.Query)
	})

	return result, nil
}

func hashStringToByte(s string) byte {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return byte(h.Sum32() % 256)
}
