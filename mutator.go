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
	if numIndices == 0 {
		return nil, false
	}
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
	if numNodeChoiceIndices == 0 {
		return nil, false
	}
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
	return newTrace, true
}

type SwapNodeMutator struct {
	NumSwaps int
	rand     *rand.Rand
}

var _ Mutator = &SwapNodeMutator{}

func NewSwapNodeMutator(swaps int) *SwapNodeMutator {
	return &SwapNodeMutator{
		NumSwaps: swaps,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SwapNodeMutator) Mutate(trace *List[*SchedulingChoice], _ *List[*Event]) (*List[*SchedulingChoice], bool) {
	nodeChoiceIndices := make([]int, 0)
	for i, choice := range trace.Iter() {
		if choice.Type == Node {
			nodeChoiceIndices = append(nodeChoiceIndices, i)
		}
	}
	numNodeChoiceIndices := len(nodeChoiceIndices)
	if numNodeChoiceIndices == 0 {
		return nil, false
	}
	toSwap := make(map[int]map[int]bool)
	for len(toSwap) < s.NumSwaps {
		i := nodeChoiceIndices[s.rand.Intn(numNodeChoiceIndices)]
		j := nodeChoiceIndices[s.rand.Intn(numNodeChoiceIndices)]
		if _, ok := toSwap[i]; !ok {
			toSwap[i] = map[int]bool{j: true}
		}
	}
	newTrace := trace
	for i, v := range toSwap {
		for j := range v {
			first, _ := newTrace.Get(i)
			second, _ := newTrace.Get(j)
			newTrace.Set(i, second)
			newTrace.Set(j, first)
		}
	}
	return newTrace, true
}

type SwapIntegerChoiceMutator struct {
	NumSwaps int
	rand     *rand.Rand
}

func NewSwapIntegerChoiceMutator(numswaps int) *SwapIntegerChoiceMutator {
	return &SwapIntegerChoiceMutator{
		NumSwaps: numswaps,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SwapIntegerChoiceMutator) Mutate(trace *List[*SchedulingChoice], _ *List[*Event]) (*List[*SchedulingChoice], bool) {
	integerChoiceIndices := make([]int, 0)
	for i, choice := range trace.Iter() {
		if choice.Type == RandomInteger {
			integerChoiceIndices = append(integerChoiceIndices, i)
		}
	}
	numIntegerChoiceIndices := len(integerChoiceIndices)
	if numIntegerChoiceIndices == 0 {
		return nil, false
	}
	toSwap := make(map[int]map[int]bool)
	for len(toSwap) < s.NumSwaps {
		i := integerChoiceIndices[s.rand.Intn(numIntegerChoiceIndices)]
		j := integerChoiceIndices[s.rand.Intn(numIntegerChoiceIndices)]
		if _, ok := toSwap[i]; !ok {
			toSwap[i] = map[int]bool{j: true}
		}
	}
	newTrace := trace
	for i, v := range toSwap {
		for j := range v {
			first, _ := newTrace.Get(i)
			second, _ := newTrace.Get(j)
			newTrace.Set(i, second)
			newTrace.Set(j, first)
		}
	}
	return newTrace, true
}

type ScaleDownIntChoiceMutator struct {
	NumPoints int
	rand      *rand.Rand
}

var _ Mutator = &ScaleDownIntChoiceMutator{}

func NewScaleDownIntChoiceMutator(numPoints int) *ScaleDownIntChoiceMutator {
	return &ScaleDownIntChoiceMutator{
		NumPoints: numPoints,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *ScaleDownIntChoiceMutator) Mutate(trace *List[*SchedulingChoice], _ *List[*Event]) (*List[*SchedulingChoice], bool) {
	integerChoiceIndices := make([]int, 0)
	for i, choice := range trace.Iter() {
		if choice.Type == RandomInteger {
			integerChoiceIndices = append(integerChoiceIndices, i)
		}
	}
	numIntegerChoiceIndices := len(integerChoiceIndices)
	if numIntegerChoiceIndices == 0 {
		return nil, false
	}
	toScaleDown := make(map[int]bool)
	for len(toScaleDown) < s.NumPoints {
		next := s.rand.Intn(numIntegerChoiceIndices)
		toScaleDown[next] = true
	}
	newTrace := trace
	for i, _ := range toScaleDown {
		curChoice, ok := trace.Get(i)
		if !ok {
			continue
		}
		newChoice := &SchedulingChoice{
			Type:          RandomInteger,
			IntegerChoice: s.rand.Intn(curChoice.IntegerChoice),
		}
		newTrace.Set(i, newChoice)
	}

	return newTrace, true
}
