package main

import (
	"math/rand"
	"time"
)

type EmptyMutator struct {
}

var _ Mutator = &EmptyMutator{}

func (e *EmptyMutator) Mutate(schedulerTrace *List[*SchedulingChoice], trace *List[*Event]) (*List[*SchedulingChoice], bool) {
	return nil, false
}

type ChoiceMutator struct {
	NumFlips int
	rand     *rand.Rand
}

func NewChoiceMutator(flips int) *ChoiceMutator {
	return &ChoiceMutator{
		NumFlips: flips,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

var _ Mutator = &ChoiceMutator{}

func (c *ChoiceMutator) Mutate(trace *List[*SchedulingChoice], _ *List[*Event]) (*List[*SchedulingChoice], bool) {
	booleanChoiceIndices := make([]int, 0)
	for i, choice := range trace.Iter() {
		if choice.Type == RandomBoolean {
			booleanChoiceIndices = append(booleanChoiceIndices, i)
		}
	}
	toFlip := make(map[int]bool)
	numIndices := len(booleanChoiceIndices)
	for len(toFlip) < c.NumFlips {
		next := booleanChoiceIndices[c.rand.Intn(numIndices)]
		if _, ok := toFlip[next]; !ok {
			toFlip[next] = true
		}
	}

	newTrace := NewList[*SchedulingChoice]()
	for i, choice := range trace.Iter() {
		if _, ok := toFlip[i]; ok {
			newTrace.Append(&SchedulingChoice{
				Type:          choice.Type,
				BooleanChoice: !choice.BooleanChoice,
			})
		} else {
			newTrace.Append(choice)
		}
	}

	return newTrace, true
}

type SkipNodeMutator struct {
	NumSkips int
	rand     *rand.Rand
}

var _ Mutator = &SkipNodeMutator{}

func NewSkipNodeMutator(skips int) *SkipNodeMutator {
	return &SkipNodeMutator{
		NumSkips: skips,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (d *SkipNodeMutator) Mutate(trace *List[*SchedulingChoice], _ *List[*Event]) (*List[*SchedulingChoice], bool) {
	nodeChoiceIndices := make([]int, 0)
	for i, choice := range trace.Iter() {
		if choice.Type == Node {
			nodeChoiceIndices = append(nodeChoiceIndices, i)
		}
	}
	numNodeChoiceIndices := len(nodeChoiceIndices)
	toSkip := make(map[int]bool)
	for len(toSkip) < d.NumSkips {
		next := nodeChoiceIndices[d.rand.Intn(numNodeChoiceIndices)]
		if _, ok := toSkip[next]; !ok {
			toSkip[next] = true
		}
	}
	newTrace := NewList[*SchedulingChoice]()
	for i, choice := range trace.Iter() {
		if _, ok := toSkip[i]; !ok {
			newTrace.Append(choice)
		}
	}
	return newTrace, false
}
