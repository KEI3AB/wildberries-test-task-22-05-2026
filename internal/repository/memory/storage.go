package memory

import (
	"cmp"
	"container/heap"
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wildberries-test-task-22-05-2026/internal/domain"
)

const numOfBuckets = 6
const bucketSize = 256
const maxCachedTrends = 100

type Shard struct {
	mu       sync.RWMutex
	counters map[string]int       // топ запросов
	seen     map[seenKey]struct{} // ключ - session_id + query, мапка нужна для того, чтобы не записывать дубликаты
}

type Bucket struct {
	bucket [bucketSize]Shard
}

type RingBuffer struct {
	timeBuff  [numOfBuckets]Bucket // Будем записывать поминутно, гуляя от остатка от деления
	cacheMu   sync.RWMutex
	cachedTop []domain.TrendQuery
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

func (r *RingBuffer) StartTopCalculator(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.recalculateTop()
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
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	limit := n
	if limit > len(r.cachedTop) {
		limit = len(r.cachedTop)
	}

	result := make([]domain.TrendQuery, limit)
	copy(result, r.cachedTop[:limit])

	return result, nil
}

func hashStringToByte(s string) byte {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return byte(h % 256)
}

func (r *RingBuffer) recalculateTop() {
	h := &TopHeap{}
	heap.Init(h)

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

			for q, c := range shard.counters {
				localMerge[q] += c
			}
			shard.mu.RUnlock()
		}

		for query, count := range localMerge {
			if h.Len() < maxCachedTrends {
				heap.Push(h, domain.TrendQuery{Query: query, NumOfReq: count})
			} else if count > (*h)[0].NumOfReq {
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

	r.cacheMu.Lock()
	r.cachedTop = result
	r.cacheMu.Unlock()
}
