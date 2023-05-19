//go:build ignore

package heap

/*
type BoundedInterface interface {
	heap.Interface
	Replace(i int, x any)
}

type Bounded struct { // rename to max bounded
	heap.Interface
	n int
}

func NewBoundedHeap(ifc heap.Interface, n int) *Bounded {
	return &Bounded{Interface: ifc, n: n}
}

func (b *Bounded) Init() {
	heap.Init(b.Interface)
}

func (b *Bounded) Push(x any) {

	heap.Push(b.Interface, x)
}

func Push(b Bounded, x any) {
	if !b.InBounds(x) {
		return
	}

	b.Push(x)
	siftUp(b, b.Len()-1)
}

func Pop(b Bounded) any {
	n := b.Len() - 1
	b.Swap(0, n)
	siftDown(b, 0, n)
	return b.Pop()
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (i * 2) + 1 }
func right(i int) int  { return left(i) + 1 }

func siftUp(b Bounded, i int) {
	for {
		p := parent(i)
		if i == p || b.Less(p, i) {
			//if h.values[p] == h.values[i] {
			//	fmt.Printf("duplicate: %v\n", h.values[p])
			//}
			break
		}
		b.Swap(p, i)
		i = p
	}
}

func siftDown(b Bounded, p, n int) bool {
	i := p
	for {
		l := left(i)
		if l >= n || l < 0 { // overflow
			break
		}
		// chose either the left or right sub-tree, depending
		// on which is smaller.
		t := l
		if r := right(i); r < n && b.Less(r, l) {
			t = r
		}
		if !b.Less(t, i) {
			break
		}
		b.Swap(i, t)
		i = t
	}
	return i > p
}

/*
// Heapify reorders the elements of h into a heap.
func Heapify[T any](h Interface[T]) {
	n := h.Len()
	for i := n/2 - 1; i >= 0; i-- {
		siftDown(h, i, n)
	}
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (2 * i) + 1 }
func right(i int) int  { return (2 * i) + 2 }

func siftDown[T any](h Interface[T], p, n int) bool {
	i := p
	for {
		l := left(i)
		if l >= n || l < 0 { // l < 0 guards against integer overflow
			break
		}
		// chose the smaller of the left or right subtree.
		t := l
		if r := right(i); r < n && h.Less(r, l) {
			t = r
		}
		if !h.Less(t, i) {
			break
		}
		h.Swap(i, t)
		i = t
	}
	return i > p
}

/*
type Ordered interface {
	~string | ~byte | ~int8 | ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64 | ~float32 | ~float64
}

type T[V Ordered] struct {
	values []V
	max    bool
}

// options:
// how much space to waste - cap() - len()
// dups

func newHeap[V Ordered](values []V, max bool) *T[V] {
	if values == nil {
		values = make([]V, 0)
	}
	return &T[V]{
		values: values[:0],
		max:    max,
	}
}

func NewMin[V Ordered](values []V) *T[V] {
	return newHeap(values, false)
}

func NewMax[V Ordered](values []V) *T[V] {
	return newHeap(values, true)
}

func (h *T[V]) Heapify() {
	h.heapify(0)
}

func (h *T[V]) Len() int { return len(h.values) }

func (h *T[V]) Cap() int { return cap(h.values) }

func (h *T[V]) Push(v V) {
	l := len(h.values)
	h.values = append(h.values, v)
	h.siftUp(l)
}

func (h *T[V]) Peek() V {
	return h.values[0]
}

func (h *T[V]) Pop() V {
	v := h.values[0]
	n := h.Len() - 1
	h.swap(0, n)
	h.siftDown(0)
	h.values = h.values[0 : n-1]
	return v
}

func (h *T[V]) Remove(i int) V {
	n := h.Len() - 1
	v := h.values[i]
	if n == i {
		h.values = h.values[0 : n-1]
		return v
	}
	h.swap(i, n)
	if !h.siftDown(i) {
		h.siftUp(i)
	}
	h.values = h.values[0 : len(h.values)-1]
	return v
}

func (h *T[V]) swap(i, j int) {
	h.values[i], h.values[j] = h.values[j], h.values[i]
}

func (h *T[V]) less(i, j int) bool {
	if h.max {
		return h.values[i] > h.values[j]
	}
	return h.values[i] < h.values[j]
}

func (h *T[V]) heapify(i int) {
	n := h.Len()
	for i := n/2 - 1; i > 0; i-- {
		h.siftDown(i)
	}
}



func (h *T[V]) siftUp(i int) {
	for {
		p := parent(i)
		if i == p || h.less(p, i) {
			//if h.values[p] == h.values[i] {
			//	fmt.Printf("duplicate: %v\n", h.values[p])
			//}
			break
		}
		h.swap(p, i)
		i = p
	}
}

func (h *T[V]) siftDown(p int) bool {
	i := p
	n := h.Len() - 1
	for {
		l := left(i)
		if l >= n || l < 0 { // overflow
			break
		}
		// chose either the left or right sub-tree, depending
		// on which is smaller.
		t := l
		if r := right(i); r < n && h.less(r, l) {
			t = r
		}
		if !h.less(t, i) {
			break
		}
		h.swap(i, t)
		i = t
	}
	return i > p
}

/*

// dups...

type Keyed[V comparable, D any] struct {
	values []V
	data   []D
}

func (h *Keyed[V, D]) Len() int { return len(h.data) }

func (h *Keyed[V, D]) Push(v V, d D) {
	h.values = append(h.values, v)
	h.data = append(h.data, d)

	// h.up(h.Len() - 1)
}

func (h *Keyed[V, D]) Pop() (V, D) {
	n := h.Len() - 1
	if n > 0 {
		//h.swap(0, n)
		//h.down()
	}
	v := h.values[n]
	d := h.data[n]
	h.values = h.values[0:n]
	h.data = h.data[0:n]
	return v, d
}

func (h *Keyed[V, D]) Peek() (V, D) {
	return h.values[0], h.data[0]
}

func (h *Keyed[V, D]) PeekN(n int) ([]V, []D) {
	vo := make([]V, n)
	do := make([]D, n)

	vo[0], do[0] = h.values[0], h.data[0]

	return vo, do
}

/*
type MapIndex[T comparable] map[T]int

func (mi MapIndex[T]) Insert(k T, v int) {
	mi[k] = v
}

func (mi MapIndex[T]) Lookup(k T) int {
	return mi[k]
}

type Index[T comparable] interface {
	Encode(T) int64
	Insert(k T, v int)
	Lookup(k T) (v int)
}

type Numeric[ValueT ArithmeticTypes, IndexT comparable] struct {
	order  Order
	total  ValueT
	values []ValueT
	index  Index[IndexT]
}

func NewNumericIndexed[ValueT ArithmeticTypes, IndexT comparable](order Order, index Index[IndexT]) *Numeric[ValueT, IndexT] {
	return &Numeric[ValueT, IndexT]{
		order:  order,
		values: make([]ValueT, 0),
		index:  index,
	}
}

/*
func NewNumeric[ValueT NumericTypes, DataT any](order Order) *Numeric[ValueT, DataT] {
	return &Heap[ValueT, DataT]{
		order:  order,
		values: make([]ValueT, 0),
		data:   make([]DataT, 0),
	}
}

func (h *Heap[V, D]) swap(i, j int) {
	h.values[i], h.values[j] = h.values[j], h.values[i]
	h.data[i], h.data[j] = h.data[j], h.data[i]
}

func (h *Heap[V, D]) Len() int { return len(h.data) }

func (h *Heap[V, D]) Push(v V, d D) {
	h.total += v
	h.values = append(h.values, v)
	h.data = append(h.data, d)
	h.up(h.Len() - 1)
}

func (h *Heap[V, D]) Pop() (V, D) {
	n := h.Len() - 1
	if n > 0 {
		h.swap(0, n)
		h.down()
	}
	v := h.values[n]
	d := h.data[n]
	h.values = h.values[0:n]
	h.data = h.data[0:n]
	return v, d
}

func (h *Heap[V, D]) Peek() (V, D) {
	return h.values[0], h.data[0]
}

func (h *Heap[V, D]) PeekN(n int) (V, D) {
	return h.values[0], h.data[0]
}

func (h *Heap[V, D]) up(jj int) {
	for {
		i := parent(jj)
		if i == jj || !h.comp(h.data[jj], h.data[i]) {
			break
		}
		h.swap(i, jj)
		jj = i
	}
}

func (h *Heap[V, D]) down() {
	n := h.Len() - 1
	i1 := 0
	for {
		j1 := left(i1)
		if j1 >= n || j1 < 0 {
			break
		}
		j := j1
		j2 := right(i1)
		if j2 < n && h.comp(h.data[j2], h.data[j1]) {
			j = j2
		}
		if !h.comp(h.data[j], h.data[i1]) {
			break
		}
		h.swap(i1, j)
		i1 = j
	}
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (i * 2) + 1 }
func right(i int) int  { return left(i) + 1 }
*/

/*


/*
// An IntHeap is a min-heap of ints.
type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *IntHeap) Push(x any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(int))
}

func (h *IntHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// This example inserts several ints into an IntHeap, checks the minimum,
// and removes them in order of priority.
func Example_intHeap() {
	h := &IntHeap{2, 1, 5}
	heap.Init(h)
	heap.Push(h, 3)
	fmt.Printf("minimum: %d\n", (*h)[0])
	for h.Len() > 0 {
		fmt.Printf("%d ", heap.Pop(h))
	}
	fmt.Printf("% 2v\n", h)
	// Output:
	// minimum: 1
	// 1 2 3 5
}

/*
func TestHeap(t *testing.T) {
	var min heap.MinHeap[int, int]

	for i := 0; i < 20; i++ {
		min.Push(i, i*10)
	}
}

/*
func (h *T[V]) verify(t *testing.T, p int) {
	n := h.Len()
	if r := right(p); r < n {
		if h.less(r, p) {
			t.Errorf("heap invariant invalidated [%d] = %v > [%d] = %v", p, h.values[p], r, h.values[r])
			return
		}
		h.verify(t, r)
	}
	if l := left(p); l < n {
		if h.less(l, p) {
			t.Errorf("heap invariant invalidated [%d] = %v > [%d] = %v", p, h.values[p], l, h.values[l])
			return
		}
		h.verify(t, l)
	}
}

func TestInit0(t *testing.T) {
	minh := NewMin(make([]int, 20))
	minh.Heapify()
	minh.verify(t, 0)

	for i := 1; minh.Len() > 0; i++ {
		x := minh.Pop()
		minh.verify(t, 0)
		if got, want := x, 0; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

}

func TestInit1(t *testing.T) {
	minh := NewMax(make([]int, 20))
	minh.Heapify()
	minh.verify(t, 0)
	minh = NewMin[int](nil)
	for i := 0; i < 20; i++ {
		minh.Push(i)
	}
	minh.Heapify()
	minh.verify(t, 0)

	for i := 1; minh.Len() > 0; i++ {
		x := minh.Pop()
		minh.verify(t, 0)
		if got, want := x, i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func BenchmarkDup(b *testing.B) {
	const n = 10000
	h := NewMin(make([]int, 0, n))
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			h.Push(0) // all elements are the same
		}
		for h.Len() > 0 {
			h.Pop()
		}
	}
}


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
