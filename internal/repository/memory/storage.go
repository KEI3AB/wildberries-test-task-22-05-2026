package memory

import "sync"

type Shard struct {
	mu       sync.RWMutex
	counters map[string]int      // топ запросов
	seen     map[string]struct{} // ключ - session_id + query, мапка нужна для того, чтобы не записывать дубликаты
}

type Bucket struct {
	bucket [256]Shard
}

type RingBuffer struct {
	timeBuff [6]Bucket // Будем записывать поминутно, гуляя от остатка от деления
}

func NewRingBuffer() *RingBuffer {
	buffer := &RingBuffer{}

	for i := range buffer.timeBuff {
		for j := range buffer.timeBuff[i].bucket {
			buffer.timeBuff[i].bucket[j].counters = make(map[string]int)
			buffer.timeBuff[i].bucket[j].seen = map[string]struct{}{}
		}
	}

	return buffer
}
