package main

import (
	"math/rand"
	"time"
)

type Strategy interface {
	GetNextNode([]uint64) uint64
	GetRandomBoolean() bool
	GetRandomInteger(int) int
}

type RandomStrategy struct {
	rand *rand.Rand
}

var _ Strategy = &RandomStrategy{}

func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RandomStrategy) GetNextNode(available []uint64) uint64 {
	randIndex := r.rand.Intn(len(available))
	return available[randIndex]
}

func (r *RandomStrategy) GetRandomBoolean() bool {
	return r.rand.Intn(2) == 0
}

func (r *RandomStrategy) GetRandomInteger(max int) int {
	return r.rand.Intn(max)
}

type RoundRobinStrategy struct {
	*RandomStrategy
	NumNodes int
	curNode  int
}

var _ Strategy = &RoundRobinStrategy{}

func NewRoundRobinStrategy(numNodes int) *RoundRobinStrategy {
	return &RoundRobinStrategy{
		RandomStrategy: NewRandomStrategy(),
		NumNodes:       numNodes,
		curNode:        0,
	}
}

func (r *RoundRobinStrategy) GetNextNode(available []uint64) uint64 {
	m := make(map[uint64]bool)
	for _, n := range available {
		m[n] = true
	}
	next := r.curNode
	for _, ok := m[uint64(next)]; !ok; next = (next + 1) % r.NumNodes {
	}
	r.curNode = (next + 1) % r.NumNodes
	return uint64(next)
}
