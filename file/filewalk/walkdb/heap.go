package walkdb

import (
	"container/heap"
	"sync"
)

type Metric struct {
	Prefix string
	Size   int64
}

type sizeHeap struct {
	mu       sync.Mutex
	Metrics  []Metric
	Prefixes map[string]int
}

func newSizeHeap() sizeHeap {
	return sizeHeap{
		Prefixes: map[string]int{},
	}
}

func (h *sizeHeap) Len() int {
	return len(h.Metrics)
}
func (h *sizeHeap) Less(i, j int) bool {
	return h.Metrics[i].Size >= h.Metrics[j].Size
}

func (h *sizeHeap) Swap(i, j int) {
	h.Prefixes[h.Metrics[i].Prefix] = j
	h.Prefixes[h.Metrics[j].Prefix] = i
	h.Metrics[i], h.Metrics[j] = h.Metrics[j], h.Metrics[i]
}

func (h *sizeHeap) Push(x interface{}) {
	m := x.(Metric)
	h.Metrics = append(h.Metrics, m)
	h.Prefixes[m.Prefix] = len(h.Metrics) - 1
}

func (h *sizeHeap) Pop() interface{} {
	old := h.Metrics
	n := len(old)
	x := old[n-1]
	h.Metrics = old[0 : n-1]
	delete(h.Prefixes, x.Prefix)
	return x
}

func (h *sizeHeap) update(prefix string, value int64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := Metric{Prefix: prefix, Size: value}
	idx, ok := h.Prefixes[prefix]
	if !ok {
		heap.Push(h, m)
		return value
	}
	delta := m.Size - h.Metrics[idx].Size
	if delta == 0 {
		return 0
	}

	h.Metrics[idx] = m
	heap.Fix(h, idx)
	return delta
}

func (h *sizeHeap) init() {
	h.mu.Lock()
	defer h.mu.Unlock()
	heap.Init(h)
}

func (h *sizeHeap) TopN(n int) []Metric {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n >= len(h.Metrics) {
		n = len(h.Metrics) - 1
	}
	top := make([]Metric, n)
	for i := 0; i < n; i++ {
		top[i] = heap.Pop(h).(Metric)
	}
	return top
}
