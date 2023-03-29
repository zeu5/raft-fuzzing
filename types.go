package main

type Event struct {
	Name   string
	Params map[string]interface{}
	Reset  bool
}

var (
	Node          SchedulingChoiceType = "Node"
	RandomBoolean SchedulingChoiceType = "RandomBoolean"
	RandomInteger SchedulingChoiceType = "RandomInteger"
)

type SchedulingChoiceType string

type SchedulingChoice struct {
	Type          SchedulingChoiceType
	NodeID        uint64
	BooleanChoice bool
	IntegerChoice int
}

type Queue[T any] struct {
	q []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		q: make([]T, 0),
	}
}

func (q *Queue[T]) Push(elem T) {
	q.q = append(q.q, elem)
}

func (q *Queue[T]) Pop() (elem T, ok bool) {
	if len(q.q) < 1 {
		ok = false
		return
	}
	elem = q.q[0]
	q.q = q.q[1:]
	ok = true
	return
}

func (q *Queue[T]) Size() int {
	return len(q.q)
}

func (q *Queue[T]) Reset() {
	q.q = make([]T, 0)
}

type List[T any] struct {
	l []T
}

func NewList[T any]() *List[T] {
	return &List[T]{
		l: make([]T, 0),
	}
}

func (l *List[T]) Append(elem T) {
	l.l = append(l.l, elem)
}

func (l *List[T]) Size() int {
	return len(l.l)
}

func (l *List[T]) Get(index int) (elem T, ok bool) {
	if len(l.l) <= index {
		ok = false
		return
	}
	elem = l.l[index]
	ok = true
	return
}

func (l *List[T]) Reset() {
	l.l = make([]T, 0)
}

func (l *List[T]) AsList() []T {
	return l.l
}

type State struct {
	Repr string
	Key  int64
}
