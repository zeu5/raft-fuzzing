package main

import (
	"math/rand"
	"time"

	pb "github.com/zeu5/raft-fuzzing/raft/raftpb"
)

type Fuzzer struct {
	messageQueues               map[uint64]*Queue[pb.Message]
	nodes                       []uint64
	config                      *FuzzerConfig
	mutatedTracesQueue          *Queue[*List[*SchedulingChoice]]
	mutatedNodeChoices          *Queue[uint64]
	curEventTrace               *List[*Event]
	curTrace                    *List[*SchedulingChoice]
	mutatedRandomBooleanChoices *Queue[bool]
	mutatedRandomIntegerChoices *Queue[int]
	rand                        *rand.Rand
	raftEnvironment             *RaftEnvironment
}

type FuzzerConfig struct {
	Iterations            int
	Steps                 int
	Mutator               Mutator
	Guider                Guider
	Strategy              Strategy
	RaftEnvironmentConfig RaftEnvironmentConfig
	MutPerTrace           int
	InitialPopulation     int
}

func NewFuzzer(config *FuzzerConfig) *Fuzzer {
	f := &Fuzzer{
		config:                      config,
		nodes:                       make([]uint64, 0),
		messageQueues:               make(map[uint64]*Queue[pb.Message]),
		mutatedTracesQueue:          NewQueue[*List[*SchedulingChoice]](),
		mutatedNodeChoices:          NewQueue[uint64](),
		curEventTrace:               NewList[*Event](),
		curTrace:                    NewList[*SchedulingChoice](),
		mutatedRandomBooleanChoices: NewQueue[bool](),
		mutatedRandomIntegerChoices: NewQueue[int](),
		rand:                        rand.New(rand.NewSource(time.Now().UnixNano())),
		raftEnvironment:             NewRaftEnvironment(config.RaftEnvironmentConfig),
	}
	f.raftEnvironment.Setup(&FuzzContext{fuzzer: f})
	for i := 0; i <= f.config.RaftEnvironmentConfig.Replicas; i++ {
		f.nodes = append(f.nodes, uint64(i))
		f.messageQueues[uint64(i)] = NewQueue[pb.Message]()
	}
	return f
}

func (f *Fuzzer) GetRandomBoolean() (choice bool) {
	if f.mutatedRandomBooleanChoices.Size() > 0 {
		choice, _ = f.mutatedRandomBooleanChoices.Pop()
	} else {
		choice = f.config.Strategy.GetRandomBoolean()
	}
	f.curEventTrace.Append(&Event{
		Name: "RandomBooleanChoice",
		Params: map[string]interface{}{
			"choice": choice,
		},
	})
	f.curTrace.Append(&SchedulingChoice{
		Type:          RandomBoolean,
		BooleanChoice: choice,
	})
	return
}

func (f *Fuzzer) GetRandomInteger(max int) (choice int) {
	if f.mutatedRandomIntegerChoices.Size() > 0 {
		choice, _ = f.mutatedRandomIntegerChoices.Pop()
	} else {
		choice = f.config.Strategy.GetRandomInteger(max)
	}
	f.curEventTrace.Append(&Event{
		Name: "RandomIntegerChoice",
		Params: map[string]interface{}{
			"choice": choice,
		},
	})
	f.curTrace.Append(&SchedulingChoice{
		Type:          RandomInteger,
		IntegerChoice: choice,
	})
	return
}

func (f *Fuzzer) GetNextMessage() (message pb.Message, ok bool) {
	var nextNode uint64
	if f.mutatedNodeChoices.Size() > 0 {
		nextNode, _ = f.mutatedNodeChoices.Pop()
	} else {
		availableNodes := make([]uint64, 0)
		for node, q := range f.messageQueues {
			if q.Size() > 0 {
				availableNodes = append(availableNodes, node)
			}
		}
		if len(availableNodes) == 0 {
			ok = false
			return
		}
		nextNode = f.config.Strategy.GetNextNode(availableNodes)
	}
	message, ok = f.messageQueues[nextNode].Pop()
	if ok && message.To != 0 {
		f.curEventTrace.Append(&Event{
			Name: "DeliverMessage",
			Params: map[string]interface{}{
				"type":     message.Type.String(),
				"term":     message.Term,
				"from":     message.From,
				"to":       message.To,
				"log_term": message.LogTerm,
				"entries":  message.Entries,
				"index":    message.Index,
				"commit":   message.Commit,
				"vote":     message.Vote,
				"reject":   message.Reject,
			},
		})
	}
	f.curTrace.Append(&SchedulingChoice{
		Type:   Node,
		NodeID: nextNode,
	})
	return
}

func (f *Fuzzer) recordSend(message pb.Message) {
	f.curEventTrace.Append(&Event{
		Name: "SendMessage",
		Params: map[string]interface{}{
			"type":     message.Type.String(),
			"term":     message.Term,
			"from":     message.From,
			"to":       message.To,
			"log_term": message.LogTerm,
			"entries":  message.Entries,
			"index":    message.Index,
			"commit":   message.Commit,
			"vote":     message.Vote,
			"reject":   message.Reject,
		},
	})
}

func (f *Fuzzer) Run() error {
	for i := 0; i < f.config.Iterations; i++ {
		f.RunIteration(i)
	}
	return nil
}

func (f *Fuzzer) RunIteration(iteration int) {
	ctx := &FuzzContext{fuzzer: f}
	// Reset current trace
	f.curEventTrace.Reset()
	f.curTrace.Reset()
	for _, q := range f.messageQueues {
		q.Reset()
	}
	init := f.raftEnvironment.Reset()
	for _, m := range init {
		f.recordSend(m)
		f.messageQueues[m.To].Push(m)
	}

	for j := 0; j < f.config.Steps; j++ {
		message, ok := f.GetNextMessage()
		if !ok {
			continue
		}
		new := f.raftEnvironment.Step(ctx, message)
		for _, m := range new {
			if m.Type != pb.MsgProp {
				f.recordSend(m)
			}
			f.messageQueues[m.To].Push(m)
		}
	}
	f.mutatedNodeChoices.Reset()
	f.mutatedRandomBooleanChoices.Reset()
	f.mutatedRandomIntegerChoices.Reset()
	if ok := f.config.Guider.Check(f.curTrace, f.curEventTrace); ok {
		mutatedTraces := make([]*List[*SchedulingChoice], 0)
		for i := 0; i < f.config.MutPerTrace; i++ {
			new, ok := f.config.Mutator.Mutate(f.curTrace, f.curEventTrace)
			if ok {
				mutatedTraces = append(mutatedTraces, copyTrace(new, defaultCopyFilter()))
			}
		}
		if len(mutatedTraces) > 0 {
			f.mutatedTracesQueue.PushAll(mutatedTraces...)
		}
	}
	if iteration+1 > f.config.InitialPopulation {
		if f.mutatedTracesQueue.Size() > 0 {
			// fmt.Println("Picking mutated trace")
			mutatedTrace, _ := f.mutatedTracesQueue.Pop()
			for _, choice := range mutatedTrace.Iter() {
				switch choice.Type {
				case RandomBoolean:
					f.mutatedRandomBooleanChoices.Push(choice.BooleanChoice)
				case RandomInteger:
					f.mutatedRandomIntegerChoices.Push(choice.IntegerChoice)
				case Node:
					f.mutatedNodeChoices.Push(choice.NodeID)
				}
			}
		}
	}
}

type Mutator interface {
	Mutate(*List[*SchedulingChoice], *List[*Event]) (*List[*SchedulingChoice], bool)
}

type FuzzContext struct {
	fuzzer *Fuzzer
}

func (f *FuzzContext) AddEvent(e *Event) {
	f.fuzzer.curEventTrace.Append(e)
}

func (f *FuzzContext) RandomBooleanChoice() bool {
	return f.fuzzer.GetRandomBoolean()
}

func (f *FuzzContext) RandomIntegerChoice(max int) int {
	return f.fuzzer.GetRandomInteger(max)
}
