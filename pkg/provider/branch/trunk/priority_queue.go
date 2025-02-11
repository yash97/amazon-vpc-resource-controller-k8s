package trunk

import (
	"container/heap"
	"fmt"
)

type ENIHeap interface {
	Push(item *ENIDetails) error
	Pop() (*ENIDetails, bool)
	Peek() (*ENIDetails, bool)
	Items() []ENIDetails
	Len() int
}

type eniMinHeap []*ENIDetails

func (h eniMinHeap) Len() int { return len(h) }
func (h eniMinHeap) Less(i, j int) bool {
	return h[i].nextDueTime.Before(h[j].nextDueTime)
}

func (h eniMinHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *eniMinHeap) Push(x interface{}) {
	*h = append(*h, x.(*ENIDetails))
}

func (h *eniMinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

type eniHeapImpl struct {
	heap eniMinHeap
}

var _ ENIHeap = (*eniHeapImpl)(nil)

func NewENIHeap() ENIHeap {
	impl := &eniHeapImpl{
		heap: make(eniMinHeap, 0),
	}
	heap.Init(&impl.heap)
	return impl
}

func (e *eniHeapImpl) Push(item *ENIDetails) error {
	if item.nextDueTime.IsZero() {
		return fmt.Errorf("eni next due time is not setup")
	}
	fmt.Println("pushing element ", item)
	heap.Push(&e.heap, item)
	return nil
}

func (e *eniHeapImpl) Pop() (*ENIDetails, bool) {
	if e.Len() == 0 {
		return nil, false
	}
	item := heap.Pop(&e.heap).(*ENIDetails)
	fmt.Println("ppoping element ", item)
	return item, true
}

func (e *eniHeapImpl) Len() int {
	return len(e.heap)
}

func (e *eniHeapImpl) Peek() (*ENIDetails, bool) {
	if e.Len() == 0 {
		return nil, false
	}
	return e.heap[0], true
}

func (e *eniHeapImpl) Items() []ENIDetails {
	items := make([]ENIDetails, len(e.heap))
	for i, ptr := range e.heap {
		items[i] = *ptr
	}
	return items
}
