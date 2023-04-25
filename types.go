package main

import "encoding/json"

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
	BooleanChoice bool `json:",omitempty"`
	IntegerChoice int  `json:",omitempty"`
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

func (q *Queue[T]) PushAll(elems ...T) {
	q.q = append(q.q, elems...)
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

func (l *List[T]) Set(index int, elem T) bool {
	if len(l.l) <= index {
		return false
	}
	l.l[index] = elem
	return true
}

func (l *List[T]) Iter() []T {
	return l.l
}

func (l *List[T]) Reset() {
	l.l = make([]T, 0)
}

func (l *List[T]) AsList() []T {
	return l.l
}

func (l *List[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.l)
}

func (l *List[T]) Copy() *List[T] {
	return &List[T]{
		l: l.l,
	}
}

type State struct {
	Repr string
	Key  int64
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
