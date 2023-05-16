//go:build ignore

package heap

/*
type Slice[V Ordered] []V

func (s Slice[V]) Len() int { return len(s) }

func (s Slice[V]) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s Slice[V]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Slice[V]) Push(x any) {
	s = append(s, x.(V))
}

func (s Slice[V]) Pop() V {
	n := len(s) - 1
	v := s[n]
	s = s[0 : n-1]
	return v
}

/*
type MinHeap[K Ordered, V any] struct {
	keys []K
	vals []V
}

func (s *MinHeap[K, V]) Len() int { return len(s.keys) }

func (s *MinHeap[K, V]) Less(i, j int) bool {
	return s.keys[i] < s.keys[j]
}

func (s *MinHeap[K, V]) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.vals[i], s.vals[j] = s.vals[j], s.vals[i]
}

func (s *MinHeap[K, V]) Push(x any) {
	s.keys = append(s.keys, k)
	s.vals = append(s.vals, v)
}

func (s *MinHeap[K, V]) Pop() (K, V) {
	n := len(s.keys) - 1
	k, v := s.keys[n], s.vals[n]
	s.keys = s.keys[0 : n-1]
	s.vals = s.vals[0 : n-1]
	return k, v
}

func (s *MinHeap[K, V]) Peek() (K, V) {
	return s.keys[0], s.vals[0]
}

type MaxHeap[K Ordered, V any] struct {
	MinHeap[K, V]
}

func (s *MaxHeap[K, V]) Less(i, j int) bool {
	return s.keys[i] > s.keys[j]
}

type SH[K Ordered, V any] struct {
	MinHeap[K, V]
}

func (h *SH[K, V]) Push(k K, v V) {
	heap.Push(&h.MinHeap, k, v)
}

/*
type item[K Ordered, V any] struct {
	key K
	val V
}

type slice[K Ordered, V any] []item[K, V]

func popSlice[K Ordered, V any](s *[]item[K, V]) (K, V) {
	n := len(*s) - 1
	k, v := (*s)[n].key, (*s)[n].val
	*s = (*s)[0 : n-1]
	return k, v
}

type MinHeap[K Ordered, V any] []item[K, V]

func (s *MinHeap[K, V]) Len() int { return len(*s) }
func (s *MinHeap[K, V]) Less(i, j int) bool {
	return (*s)[i].key < (*s)[j].key
}

func (s *MinHeap[K, V]) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *MinHeap[K, V]) Push(k K, v V) {
	*s = append(*s, item[K, V]{key: k, val: v})
}

func (s *MinHeap[K, V]) Pop() (K, V) {
	return popSlice((*[]item[K, V])(s))
}

func (s *MinHeap[K, V]) Peek() (K, V) {
	return (*s)[0].key, (*s)[0].val
}

type MaxHeap[K Ordered, V any] []item[K, V]

func (s *MaxHeap[K, V]) Len() int { return len(*s) }
func (s *MaxHeap[K, V]) Less(i, j int) bool {
	return (*s)[i].key > (*s)[j].key
}

func (s *MaxHeap[K, V]) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *MaxHeap[K, V]) Push(k K, v V) {
	*s = append(*s, item[K, V]{key: k, val: v})
}

func (s *MaxHeap[K, V]) Pop() (K, V) {
	return popSlice((*[]item[K, V])(s))
}

func (s *MaxHeap[K, V]) Peek() (K, V) {
	return (*s)[0].key, (*s)[0].val
}
*/
