package main

import (
	"encoding/json"
	"math/rand"
)

type Event struct {
	Name   string
	Node   uint64 `json:"-"`
	Params map[string]interface{}
	Reset  bool
}

var (
	Node          SchedulingChoiceType = "Node"
	RandomBoolean SchedulingChoiceType = "RandomBoolean"
	RandomInteger SchedulingChoiceType = "RandomInteger"
	StartNode     SchedulingChoiceType = "StartNode"
	StopNode      SchedulingChoiceType = "StopNode"
	ClientRequest SchedulingChoiceType = "ClientRequest"
)

type SchedulingChoiceType string

type SchedulingChoice struct {
	Type          SchedulingChoiceType
	NodeID        uint64
	MaxMessages   int
	BooleanChoice bool `json:",omitempty"`
	IntegerChoice int  `json:",omitempty"`
	Step          int  `json:",omitempty"`
	Request       int  `json:",omitempty"`
}

func (s *SchedulingChoice) Copy() *SchedulingChoice {
	return &SchedulingChoice{
		Type:          s.Type,
		NodeID:        s.NodeID,
		MaxMessages:   s.MaxMessages,
		BooleanChoice: s.BooleanChoice,
		IntegerChoice: s.IntegerChoice,
		Step:          s.Step,
		Request:       s.Request,
	}
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

func (l *List[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.l)
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

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func sample(l []int, size int, r *rand.Rand) []int {
	if size >= len(l) {
		return l
	}
	indexes := make(map[int]bool)
	for len(indexes) < size {
		i := r.Intn(len(l))
		indexes[i] = true
	}
	samples := make([]int, size)
	i := 0
	for k := range indexes {
		samples[i] = l[k]
		i++
	}
	return samples
}

func intRange(start, end int) []int {
	res := make([]int, end-start)
	for i := start; i < end; i++ {
		res[i-start] = i
	}
	return res
}
