package trunk

import (
	"errors"
	"sync"
)

// QueueInterface defines the methods that a Queue should implement
type QueueInterface interface {
	Enqueue(value *ENIDetails)
	EnqueueFront(value *ENIDetails)
	Dequeue() (*ENIDetails, error)
	First() (*ENIDetails, error)
	Last() (*ENIDetails, error)
	Size() int
	Capacity() int
	Elements() []*ENIDetails
}

type Queue struct {
	mu       sync.Mutex
	data     []*ENIDetails
	head     int
	tail     int
	count    int
	capacity int
}

func NewQueue(cap int) QueueInterface {
	if cap <= 0 {
		cap = 1
	}
	return &Queue{
		data:     make([]*ENIDetails, cap),
		head:     0,
		tail:     0,
		count:    0,
		capacity: cap,
	}
}

func (q *Queue) Enqueue(value *ENIDetails) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == q.capacity {
		q.resize()
	}

	q.data[q.tail] = value
	q.tail = (q.tail + 1) % q.capacity
	q.count++
}

func (q *Queue) EnqueueFront(value *ENIDetails) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.count == q.capacity {
		q.resize()
	}
	q.head = (q.head - 1 + q.capacity) % q.capacity
	q.data[q.head] = value
	q.count++
}

// Dequeue removes and returns the front element of the queue.
// If the queue is empty, it returns an error.
func (q *Queue) Dequeue() (*ENIDetails, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zeroVal *ENIDetails
	if q.count == 0 {
		return zeroVal, errors.New("queue is empty")
	}

	val := q.data[q.head]
	// Clear the element at head (not strictly necessary, but good practice)
	var zeroValue *ENIDetails
	q.data[q.head] = zeroValue
	q.head = (q.head + 1) % q.capacity
	q.count--

	return val, nil
}

// First returns the first element without removing it.
// If empty, returns an error.
func (q *Queue) First() (*ENIDetails, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zeroVal *ENIDetails
	if q.count == 0 {
		return zeroVal, errors.New("queue is empty")
	}

	return q.data[q.head], nil
}

// Last returns the last element without removing it.
// If empty, returns an error.
func (q *Queue) Last() (*ENIDetails, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zeroVal *ENIDetails
	if q.count == 0 {
		return zeroVal, errors.New("queue is empty")
	}

	lastIndex := (q.tail - 1 + q.capacity) % q.capacity
	return q.data[lastIndex], nil
}

// Size returns the current number of elements in the queue.
func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

// Capacity returns the current capacity of the queue.
func (q *Queue) Capacity() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.capacity
}

func (q *Queue) Elements() []*ENIDetails {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Create a slice to hold the snapshot of elements
	elems := make([]*ENIDetails, q.count)
	for i := 0; i < q.count; i++ {
		elems[i] = q.data[(q.head+i)%q.capacity]
	}

	return elems
}

// resize doubles the capacity of the queue. This is done when the queue is full.
func (q *Queue) resize() {
	newCapacity := q.capacity * 2
	newData := make([]*ENIDetails, newCapacity)

	// Copy elements to newData in order
	for i := 0; i < q.count; i++ {
		newData[i] = q.data[(q.head+i)%q.capacity]
	}

	q.data = newData
	q.head = 0
	q.tail = q.count
	q.capacity = newCapacity
}
