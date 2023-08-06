package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	pb "github.com/zeu5/raft-fuzzing/raft/raftpb"
)

type Fuzzer struct {
	messageQueues      map[uint64]*Queue[pb.Message]
	nodes              []uint64
	config             *FuzzerConfig
	mutatedTracesQueue *Queue[*List[*SchedulingChoice]]
	rand               *rand.Rand
	raftEnvironment    *RaftEnvironment

	stats map[string]interface{}
}

type traceCtx struct {
	trace          *List[*SchedulingChoice]
	mimicTrace     *List[*SchedulingChoice]
	eventTrace     *List[*Event]
	nodeChoices    *Queue[*SchedulingChoice]
	booleanChoices *Queue[bool]
	integerChoices *Queue[int]
	crashPoints    map[int]uint64
	startPoints    map[int]uint64
	clientRequests map[int]int
	rand           *rand.Rand

	fuzzer *Fuzzer
}

func (t *traceCtx) GetNextNodeChoice() (uint64, int) {
	var choice uint64
	var maxMessages int
	if t.nodeChoices.Size() > 0 {
		c, _ := t.nodeChoices.Pop()
		choice = c.NodeID
		maxMessages = c.MaxMessages
	} else {
		i := t.rand.Intn(len(t.fuzzer.nodes))
		choice = t.fuzzer.nodes[i]
		maxMessages = t.rand.Intn(t.fuzzer.config.MaxMessages)
	}
	t.trace.Append(&SchedulingChoice{
		Type:        Node,
		NodeID:      choice,
		MaxMessages: maxMessages,
	})

	return choice, maxMessages
}

func (t *traceCtx) GetRandomBoolean() (choice bool) {
	if t.booleanChoices.Size() > 0 {
		choice, _ = t.booleanChoices.Pop()
	} else {
		choice = t.rand.Intn(2) == 0
	}
	t.eventTrace.Append(&Event{
		Name: "RandomBooleanChoice",
		Params: map[string]interface{}{
			"choice": choice,
		},
	})
	t.trace.Append(&SchedulingChoice{
		Type:          RandomBoolean,
		BooleanChoice: choice,
	})
	return
}

func (t *traceCtx) GetRandomInteger(max int) (choice int) {
	if t.integerChoices.Size() > 0 {
		choice, _ = t.integerChoices.Pop()
	} else {
		choice = t.rand.Intn(max)
	}
	t.eventTrace.Append(&Event{
		Name: "RandomIntegerChoice",
		Params: map[string]interface{}{
			"choice": choice,
		},
	})
	t.trace.Append(&SchedulingChoice{
		Type:          RandomInteger,
		IntegerChoice: choice,
	})
	return
}

func (t *traceCtx) CanCrash(step int) (uint64, bool) {
	node, ok := t.crashPoints[step]
	if ok {
		t.eventTrace.Append(&Event{
			Name: "Remove",
			Node: node,
			Params: map[string]interface{}{
				"i": int(node),
			},
		})
		t.trace.Append(&SchedulingChoice{
			Type:   StopNode,
			NodeID: node,
			Step:   step,
		})
	}
	return node, ok
}

func (t *traceCtx) CanStart(step int) (uint64, bool) {
	node, ok := t.startPoints[step]
	if ok {
		t.eventTrace.Append(&Event{
			Name: "Add",
			Node: node,
			Params: map[string]interface{}{
				"i": int(node),
			},
		})
		t.trace.Append(&SchedulingChoice{
			Type:   StartNode,
			NodeID: node,
			Step:   step,
		})
	}
	return node, ok
}

func (t *traceCtx) IsClientRequest(step int) (int, bool) {
	req, ok := t.clientRequests[step]
	if ok {
		t.trace.Append(&SchedulingChoice{
			Type:    ClientRequest,
			Request: req,
		})
	}
	return req, ok
}

type FuzzerConfig struct {
	Iterations            int
	Steps                 int
	Checker               Checker
	Mutator               Mutator
	Guider                Guider
	Strategy              Strategy
	RaftEnvironmentConfig RaftEnvironmentConfig
	MutPerTrace           int
	SeedPopulationSize    int
	NumberRequests        int
	CrashQuota            int
	MaxMessages           int
	ReseedFrequency       int
}

func NewFuzzer(config *FuzzerConfig) *Fuzzer {
	f := &Fuzzer{
		config:             config,
		nodes:              make([]uint64, 0),
		messageQueues:      make(map[uint64]*Queue[pb.Message]),
		mutatedTracesQueue: NewQueue[*List[*SchedulingChoice]](),
		rand:               rand.New(rand.NewSource(time.Now().UnixNano())),
		raftEnvironment:    NewRaftEnvironment(config.RaftEnvironmentConfig),
		stats:              make(map[string]interface{}),
	}
	for i := 0; i <= f.config.RaftEnvironmentConfig.Replicas; i++ {
		f.nodes = append(f.nodes, uint64(i))
		f.messageQueues[uint64(i)] = NewQueue[pb.Message]()
	}
	f.stats["random_executions"] = 0
	f.stats["mutated_executions"] = 0
	f.stats["buggy_executions"] = 0
	return f
}

func (f *Fuzzer) Schedule(node uint64, maxMessages int) []pb.Message {
	queue, ok := f.messageQueues[node]
	if !ok || queue.Size() == 0 {
		return []pb.Message{}
	}
	messages := make([]pb.Message, 0)
	for i := 0; i < maxMessages; i++ {
		message, ok := queue.Pop()
		if !ok {
			break
		}
		messages = append(messages, message)
	}
	return messages
}

func recordReceive(message pb.Message, eventTrace *List[*Event]) {
	eventTrace.Append(&Event{
		Name: "DeliverMessage",
		Node: message.To,
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

func recordSend(message pb.Message, eventTrace *List[*Event]) {
	eventTrace.Append(&Event{
		Name: "SendMessage",
		Node: message.From,
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

func (f *Fuzzer) seed() {
	f.mutatedTracesQueue.Reset()
	for i := 0; i < f.config.SeedPopulationSize; i++ {
		trace, _ := f.RunIteration(fmt.Sprintf("pop_%d", i), nil)
		f.mutatedTracesQueue.Push(copyTrace(trace, defaultCopyFilter()))
	}
}

func (f *Fuzzer) Run() []CoverageStats {
	coverages := make([]CoverageStats, 0)
	for i := 0; i < f.config.Iterations; i++ {
		if i%f.config.ReseedFrequency == 0 {
			f.seed()
		}
		fmt.Printf("\rRunning iteration: %d/%d", i+1, f.config.Iterations)
		var mimic *List[*SchedulingChoice] = nil
		if f.mutatedTracesQueue.Size() > 0 {
			f.stats["mutated_executions"] = f.stats["mutated_executions"].(int) + 1
			mimic, _ = f.mutatedTracesQueue.Pop()
		} else {
			f.stats["random_executions"] = f.stats["random_executions"].(int) + 1
		}
		trace, eventTrace := f.RunIteration(fmt.Sprintf("fuzz_%d", i), mimic)
		if numNewStates, _ := f.config.Guider.Check(trace, eventTrace); numNewStates > 0 {
			numMutations := numNewStates * f.config.MutPerTrace
			for j := 0; j < numMutations; j++ {
				new, ok := f.config.Mutator.Mutate(trace, eventTrace)
				if ok {
					f.mutatedTracesQueue.Push(copyTrace(new, defaultCopyFilter()))
				}
			}
		}
		coverages = append(coverages, f.config.Guider.Coverage())
	}
	return coverages
}

func (f *Fuzzer) RunIteration(iteration string, mimic *List[*SchedulingChoice]) (*List[*SchedulingChoice], *List[*Event]) {
	// Setup the context for the iterations
	tCtx := &traceCtx{
		trace:          NewList[*SchedulingChoice](),
		eventTrace:     NewList[*Event](),
		nodeChoices:    NewQueue[*SchedulingChoice](),
		booleanChoices: NewQueue[bool](),
		integerChoices: NewQueue[int](),
		crashPoints:    make(map[int]uint64),
		startPoints:    make(map[int]uint64),
		clientRequests: make(map[int]int),
		rand:           f.rand,
		fuzzer:         f,
	}
	if mimic != nil {
		tCtx.mimicTrace = mimic
		for i := 0; i < mimic.Size(); i++ {
			ch, _ := mimic.Get(i)
			switch ch.Type {
			case Node:
				tCtx.nodeChoices.Push(ch.Copy())
			case RandomBoolean:
				tCtx.booleanChoices.Push(ch.BooleanChoice)
			case RandomInteger:
				tCtx.integerChoices.Push(ch.IntegerChoice)
			case StartNode:
				tCtx.startPoints[ch.Step] = ch.NodeID
			case StopNode:
				tCtx.crashPoints[ch.Step] = ch.NodeID
			case ClientRequest:
				tCtx.clientRequests[ch.Step] = ch.Request
			}
		}
	} else {
		for i := 0; i < f.config.Steps; i++ {
			var idx int = 0
			for idx == 0 {
				idx = f.rand.Intn(len(f.nodes))
			}
			tCtx.nodeChoices.Push(&SchedulingChoice{
				Type:        Node,
				NodeID:      f.nodes[idx],
				MaxMessages: f.rand.Intn(f.config.MaxMessages),
			})
		}
		choices := make([]int, f.config.Steps)
		for i := 0; i < f.config.Steps; i++ {
			choices[i] = i
		}
		for _, c := range sample(choices, f.config.CrashQuota, f.rand) {
			var idx int = 0
			for idx == 0 {
				idx = f.rand.Intn(len(f.nodes))
			}
			tCtx.crashPoints[c] = uint64(idx)
			s := sample(intRange(c, f.config.Steps), 1, f.rand)[0]
			tCtx.startPoints[s] = uint64(idx)
		}
		i := 1
		for _, req := range sample(choices, f.config.NumberRequests, f.rand) {
			tCtx.clientRequests[req] = i
			i++
		}
	}

	// Reset the queues and environment
	for _, q := range f.messageQueues {
		q.Reset()
	}
	f.raftEnvironment.Reset(&FuzzContext{traceCtx: tCtx})

	crashed := make(map[uint64]bool)
	fCtx := &FuzzContext{traceCtx: tCtx}
	for j := 0; j < f.config.Steps; j++ {
		if toCrash, ok := tCtx.CanCrash(j); ok {
			f.raftEnvironment.Stop(fCtx, toCrash)
			crashed[toCrash] = true
		}
		if toStart, ok := tCtx.CanStart(j); ok {
			_, isCrashed := crashed[toStart]
			if isCrashed {
				f.raftEnvironment.Start(fCtx, toStart)
				delete(crashed, toStart)
			}
		}
		toSchedule, maxMessages := tCtx.GetNextNodeChoice()
		if _, ok := crashed[toSchedule]; !ok {
			messages := f.Schedule(toSchedule, maxMessages)
			for _, m := range messages {
				recordReceive(m, tCtx.eventTrace)
				f.raftEnvironment.Step(fCtx, m)
			}
		}

		if reqNum, ok := tCtx.IsClientRequest(j); ok {
			req := pb.Message{
				Type: pb.MsgProp,
				From: uint64(0),
				Entries: []pb.Entry{
					{Data: []byte(strconv.Itoa(reqNum))},
				},
			}
			f.raftEnvironment.Step(fCtx, req)
		}

		for _, n := range f.raftEnvironment.Tick(fCtx) {
			recordSend(n, tCtx.eventTrace)
			f.messageQueues[n.To].Push(n)
		}
		if f.config.Checker != nil && !f.config.Checker(f.raftEnvironment) {
			f.stats["buggy_executions"] = f.stats["buggy_executions"].(int) + 1
		}
	}
	return tCtx.trace, tCtx.eventTrace
}

type Mutator interface {
	Mutate(*List[*SchedulingChoice], *List[*Event]) (*List[*SchedulingChoice], bool)
}

type FuzzContext struct {
	traceCtx *traceCtx
}

func (f *FuzzContext) AddEvent(e *Event) {
	f.traceCtx.eventTrace.Append(e)
}

func (f *FuzzContext) RandomBooleanChoice() bool {
	return f.traceCtx.GetRandomBoolean()
}

func (f *FuzzContext) RandomIntegerChoice(max int) int {
	return f.traceCtx.GetRandomInteger(max)
}
