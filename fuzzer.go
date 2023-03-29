package main

import (
	"math/rand"
	"time"

	pb "go.etcd.io/raft/v3/raftpb"
)

type Fuzzer struct {
	messageQueues               map[uint64]*Queue[pb.Message]
	nodes                       []uint64
	config                      *FuzzerConfig
	mutator                     Mutator
	mutatedNodeChoices          *Queue[uint64]
	curEventTrace               *List[*Event]
	curTrace                    *List[*SchedulingChoice]
	mutatedRandomBooleanChoices *Queue[bool]
	mutatedRandomIntegerChoices *Queue[int]
	rand                        *rand.Rand
	raftEnvironment             *RaftEnvironment
	tlcClient                   *TLCClient
	statesMap                   map[int64]bool
}

type FuzzerConfig struct {
	Iterations            int
	Steps                 int
	TLCAddr               string
	Mutator               Mutator
	RaftEnvironmentConfig RaftEnvironmentConfig
}

func NewFuzzer(config *FuzzerConfig) *Fuzzer {
	f := &Fuzzer{
		config:                      config,
		nodes:                       make([]uint64, 0),
		messageQueues:               make(map[uint64]*Queue[pb.Message]),
		mutator:                     config.Mutator,
		mutatedNodeChoices:          NewQueue[uint64](),
		curEventTrace:               NewList[*Event](),
		curTrace:                    NewList[*SchedulingChoice](),
		mutatedRandomBooleanChoices: NewQueue[bool](),
		mutatedRandomIntegerChoices: NewQueue[int](),
		rand:                        rand.New(rand.NewSource(time.Now().UnixNano())),
		raftEnvironment:             NewRaftEnvironment(config.RaftEnvironmentConfig),
		tlcClient:                   NewTLCClient(config.TLCAddr),
		statesMap:                   make(map[int64]bool),
	}
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
		choice = f.rand.Intn(2) == 0
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
		choice = f.rand.Intn(max)
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
		randIndex := f.rand.Intn(len(availableNodes))
		nextNode = availableNodes[randIndex]
	}
	message, ok = f.messageQueues[nextNode].Pop()
	f.curEventTrace.Append(&Event{
		Name: "DeliverMessage",
		Params: map[string]interface{}{
			"type":     message.Type.String(),
			"term":     message.Term,
			"from":     message.From,
			"to":       message.To,
			"log_term": message.LogTerm,
			"index":    message.Index,
			"commit":   message.Commit,
			"vote":     message.Vote,
			"reject":   message.Reject,
		},
	})
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
			"index":    message.Index,
			"commit":   message.Commit,
			"vote":     message.Vote,
			"reject":   message.Reject,
		},
	})
}

func (f *Fuzzer) Run() error {
	ctx := &FuzzContext{fuzzer: f}
	for i := 0; i < f.config.Iterations; i++ {
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
		// Transmit trace to TLC and mutate
		if tlcStates, err := f.tlcClient.SendTrace(f.curEventTrace); err == nil {
			haveNewState := false
			for _, s := range tlcStates {
				_, ok := f.statesMap[s.Key]
				if !ok {
					haveNewState = true
					f.statesMap[s.Key] = true
				}
			}
			if haveNewState {
				mutatedTrace, ok := f.mutator.Mutate(f.curTrace, f.curEventTrace)
				if ok {
					for _, choice := range mutatedTrace.AsList() {
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

	}
	return nil
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
