package memory

import "github.com/wildberries-test-task-22-05-2026/internal/domain"

type TopHeap []domain.TrendQuery

// sort.Interface
func (h TopHeap) Len() int           { return len(h) }
func (h TopHeap) Less(i, j int) bool { return h[i].NumOfReq < h[j].NumOfReq } // MinHeap
func (h TopHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// heap.Interface
func (h *TopHeap) Push(val any) { *h = append(*h, val.(domain.TrendQuery)) }
func (h *TopHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
