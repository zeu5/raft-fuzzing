package main

import (
	"fmt"
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
			newTrace.Append(choice.Copy())
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
			newTrace.Append(choice.Copy())
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
	choices := numNodeChoiceIndices
	if s.NumSwaps < choices {
		choices = s.NumSwaps
	}
	toSwap := make(map[string]map[int]int)
	for len(toSwap) < choices {
		i := nodeChoiceIndices[s.rand.Intn(numNodeChoiceIndices)]
		j := nodeChoiceIndices[s.rand.Intn(numNodeChoiceIndices)]
		key := fmt.Sprintf("%d_%d", i, j)
		if _, ok := toSwap[key]; !ok {
			toSwap[key] = map[int]int{i: j}
		}
	}
	newTrace := copyTrace(trace, defaultCopyFilter())
	for _, v := range toSwap {
		for i, j := range v {
			first, _ := newTrace.Get(i)
			second, _ := newTrace.Get(j)
			newTrace.Set(i, second.Copy())
			newTrace.Set(j, first.Copy())
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
	newTrace := copyTrace(trace, defaultCopyFilter())
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
	newTrace := copyTrace(trace, defaultCopyFilter())
	for i := range toScaleDown {
		index := integerChoiceIndices[i]
		curChoice, ok := newTrace.Get(index)
		if !ok {
			continue
		}
		if curChoice.IntegerChoice > 0 {
			newChoice := &SchedulingChoice{
				Type:          RandomInteger,
				IntegerChoice: s.rand.Intn(curChoice.IntegerChoice),
			}
			newTrace.Set(index, newChoice)
		}
	}

	return newTrace, true
}

type ScaleUpIntChoiceMutator struct {
	NumPoints int
	Max       int
	rand      *rand.Rand
}

var _ Mutator = &ScaleUpIntChoiceMutator{}

func NewScaleUpIntChoiceMutator(numPoints, max int) *ScaleUpIntChoiceMutator {
	return &ScaleUpIntChoiceMutator{
		NumPoints: numPoints,
		Max:       max,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *ScaleUpIntChoiceMutator) Mutate(trace *List[*SchedulingChoice], _ *List[*Event]) (*List[*SchedulingChoice], bool) {
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
	toScaleUp := make(map[int]bool)
	choices := numIntegerChoiceIndices
	if s.NumPoints < numIntegerChoiceIndices {
		choices = s.NumPoints
	}
	for len(toScaleUp) < choices {
		next := s.rand.Intn(numIntegerChoiceIndices)
		toScaleUp[next] = true
	}
	newTrace := copyTrace(trace, defaultCopyFilter())
	for i := range toScaleUp {
		index := integerChoiceIndices[i]
		curChoice, ok := newTrace.Get(index)
		if !ok {
			continue
		}
		newChoice := &SchedulingChoice{
			Type:          RandomInteger,
			IntegerChoice: min(s.Max, curChoice.IntegerChoice*2),
		}
		newTrace.Set(index, newChoice)
	}

	return newTrace, true
}

func copyTrace(t *List[*SchedulingChoice], filter func(*SchedulingChoice) bool) *List[*SchedulingChoice] {
	newL := NewList[*SchedulingChoice]()
	for _, e := range t.Iter() {
		if filter(e) {
			newL.Append(e.Copy())
		}
	}
	return newL
}

func defaultCopyFilter() func(*SchedulingChoice) bool {
	return func(sc *SchedulingChoice) bool {
		return true
	}
}

func typeCopyFilter(t SchedulingChoiceType) func(*SchedulingChoice) bool {
	return func(sc *SchedulingChoice) bool {
		return sc.Type == t
	}
}

type combinedMutator struct {
	mutators []Mutator
}

func (c *combinedMutator) Mutate(trace *List[*SchedulingChoice], eventTrace *List[*Event]) (*List[*SchedulingChoice], bool) {
	curTrace := copyTrace(trace, defaultCopyFilter())
	for _, m := range c.mutators {
		nextTrace, ok := m.Mutate(curTrace, eventTrace)
		if !ok {
			return nil, false
		}
		curTrace = nextTrace
	}
	return curTrace, true
}

func CombineMutators(mutators ...Mutator) Mutator {
	return &combinedMutator{
		mutators: mutators,
	}
}

type SwapCrashNodeMutator struct {
	r *rand.Rand
}

var _ Mutator = &SwapCrashNodeMutator{}

func NewSwapCrashNodeMutator() *SwapCrashNodeMutator {
	return &SwapCrashNodeMutator{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SwapCrashNodeMutator) Mutate(trace *List[*SchedulingChoice], eventTrace *List[*Event]) (*List[*SchedulingChoice], bool) {
	crashPoints := make(map[int]uint64)
	crashPointIndices := make([]int, 0)

	for i, ch := range trace.Iter() {
		if ch.Type == StopNode {
			crashPointIndices = append(crashPointIndices, i)
			crashPoints[i] = ch.NodeID
		}
	}

	sp := sample(crashPointIndices, 2, s.r)
	first := sp[0]
	second := sp[1]

	newTrace := copyTrace(trace, defaultCopyFilter())
	for i, ch := range newTrace.Iter() {
		if i == first {
			ch.NodeID = crashPoints[second]
		} else if i == second {
			ch.NodeID = crashPoints[first]
		}
	}
	return newTrace, true
}

type SwapMaxMessagesMutator struct {
	NumSwaps int
	r        *rand.Rand
}

var _ Mutator = &SwapMaxMessagesMutator{}

func NewSwapMaxMessagesMutator(swaps int) *SwapMaxMessagesMutator {
	return &SwapMaxMessagesMutator{
		NumSwaps: swaps,
		r:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SwapMaxMessagesMutator) Mutate(trace *List[*SchedulingChoice], eventTrace *List[*Event]) (*List[*SchedulingChoice], bool) {
	swaps := make(map[int]int)

	nodeChoices := make([]int, 0)
	for i, ch := range trace.Iter() {
		if ch.Type == Node {
			nodeChoices = append(nodeChoices, i)
		}
	}

	for len(swaps) < s.NumSwaps {
		sp := sample(nodeChoices, 2, s.r)
		swaps[sp[0]] = sp[1]
	}

	newTrace := copyTrace(trace, defaultCopyFilter())
	for i, j := range swaps {
		iCh, _ := newTrace.Get(i)
		jCh, _ := newTrace.Get(j)

		iChNew := iCh.Copy()
		iChNew.MaxMessages = jCh.MaxMessages
		jChNew := jCh.Copy()
		jChNew.MaxMessages = iCh.MaxMessages

		newTrace.Set(i, iChNew)
		newTrace.Set(j, jChNew)
	}
	return newTrace, true
}
