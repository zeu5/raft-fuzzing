package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"time"

	pb "go.etcd.io/raft/v3/raftpb"
)

type Fuzzer struct {
	messageQueues               map[uint64]*Queue[pb.Message]
	nodes                       []uint64
	config                      *FuzzerConfig
	mutator                     Mutator
	mutatedTracesQueue          *Queue[*List[*SchedulingChoice]]
	mutatedNodeChoices          *Queue[uint64]
	curEventTrace               *List[*Event]
	curTrace                    *List[*SchedulingChoice]
	mutatedRandomBooleanChoices *Queue[bool]
	mutatedRandomIntegerChoices *Queue[int]
	rand                        *rand.Rand
	raftEnvironment             *RaftEnvironment
	tlcClient                   *TLCClient
	statesMap                   map[int64]bool
	tracesMap                   map[string]bool
	statesTracesMap             map[string]bool
}

type FuzzerConfig struct {
	Iterations            int
	Steps                 int
	TLCAddr               string
	Mutator               Mutator
	RaftEnvironmentConfig RaftEnvironmentConfig
	MutPerTrace           int
}

func NewFuzzer(config *FuzzerConfig) *Fuzzer {
	f := &Fuzzer{
		config:                      config,
		nodes:                       make([]uint64, 0),
		messageQueues:               make(map[uint64]*Queue[pb.Message]),
		mutator:                     config.Mutator,
		mutatedTracesQueue:          NewQueue[*List[*SchedulingChoice]](),
		mutatedNodeChoices:          NewQueue[uint64](),
		curEventTrace:               NewList[*Event](),
		curTrace:                    NewList[*SchedulingChoice](),
		mutatedRandomBooleanChoices: NewQueue[bool](),
		mutatedRandomIntegerChoices: NewQueue[int](),
		rand:                        rand.New(rand.NewSource(time.Now().UnixNano())),
		raftEnvironment:             NewRaftEnvironment(config.RaftEnvironmentConfig),
		tlcClient:                   NewTLCClient(config.TLCAddr),
		statesMap:                   make(map[int64]bool),
		tracesMap:                   make(map[string]bool),
		statesTracesMap:             make(map[string]bool),
	}
	for i := 0; i <= f.config.RaftEnvironmentConfig.Replicas; i++ {
		f.nodes = append(f.nodes, uint64(i))
		f.messageQueues[uint64(i)] = NewQueue[pb.Message]()
	}
	return f
}

type CoverageStats struct {
	UniqueStates      int
	UniqueTraces      int
	UniqueStateTraces int
}

func (f *Fuzzer) Coverage() CoverageStats {
	return CoverageStats{
		UniqueStates:      len(f.statesMap),
		UniqueTraces:      len(f.tracesMap),
		UniqueStateTraces: len(f.statesTracesMap),
	}
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
			"entries":  message.Entries,
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

func (f *Fuzzer) RunIteration(_ int) {
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
	bs, _ := json.Marshal(f.curTrace)
	sum := sha256.Sum256(bs)
	hash := hex.EncodeToString(sum[:])
	if _, ok := f.tracesMap[hash]; !ok {
		f.tracesMap[hash] = true
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
		bs, _ := json.Marshal(tlcStates)
		sum := sha256.Sum256(bs)
		stateTraceHash := hex.EncodeToString(sum[:])
		if _, ok := f.statesTracesMap[stateTraceHash]; !ok {
			f.statesTracesMap[stateTraceHash] = true
		}
		if haveNewState {
			mutatedTraces := make([]*List[*SchedulingChoice], 0)
			for i := 0; i < f.config.MutPerTrace; i++ {
				new, ok := f.mutator.Mutate(f.curTrace, f.curEventTrace)
				if ok {
					mutatedTraces = append(mutatedTraces, new)
				}
			}
			if len(mutatedTraces) > 0 {
				f.mutatedTracesQueue.PushAll(mutatedTraces...)
			}
		}
		if f.mutatedTracesQueue.Size() > 0 {
			mutatedTrace, _ := f.mutatedTracesQueue.Pop()
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
