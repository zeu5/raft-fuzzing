package main

import (
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/zeu5/raft-fuzzing/raft"
	pb "github.com/zeu5/raft-fuzzing/raft/raftpb"
)

// type RaftRand struct {
// 	rand *rand.Rand
// 	ctx  *FuzzContext
// 	lock *sync.Mutex
// }

// var _ raft.Rand = &RaftRand{}

// func NewRaftRand() *RaftRand {
// 	return &RaftRand{
// 		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
// 		ctx:  nil,
// 		lock: new(sync.Mutex),
// 	}
// }

// func (r *RaftRand) Intn(max int) int {
// 	r.lock.Lock()
// 	defer r.lock.Unlock()
// 	if r.ctx == nil {
// 		return r.rand.Intn(max)
// 	}
// 	return r.ctx.RandomIntegerChoice(max)
// }

// func (r *RaftRand) UpdateCtx(ctx *FuzzContext) {
// 	r.lock.Lock()
// 	defer r.lock.Unlock()
// 	r.ctx = ctx
// }

type RaftEnvironmentConfig struct {
	Replicas      int
	ElectionTick  int
	HeartbeatTick int
	TicksPerStep  int
}

type RaftEnvironment struct {
	config    RaftEnvironmentConfig
	nodes     map[uint64]*raft.RawNode
	storages  map[uint64]*raft.MemoryStorage
	curStates map[uint64]raft.Status
}

func NewRaftEnvironment(config RaftEnvironmentConfig) *RaftEnvironment {
	r := &RaftEnvironment{
		config:    config,
		nodes:     make(map[uint64]*raft.RawNode),
		storages:  make(map[uint64]*raft.MemoryStorage),
		curStates: make(map[uint64]raft.Status),
	}
	r.makeNodes()
	return r
}

func (r *RaftEnvironment) makeNodes() {
	confChanges := make([]pb.ConfChangeV2, r.config.Replicas)
	for i := 0; i < r.config.Replicas; i++ {
		confChanges[i] = pb.ConfChange{NodeID: uint64(i + 1), Type: pb.ConfChangeAddNode}.AsV2()
	}
	for i := 0; i < r.config.Replicas; i++ {
		storage := raft.NewMemoryStorage()
		nodeID := uint64(i + 1)
		r.storages[nodeID] = storage
		node, _ := raft.NewRawNode(&raft.Config{
			ID:                        nodeID,
			ElectionTick:              r.config.ElectionTick,
			HeartbeatTick:             r.config.HeartbeatTick,
			Storage:                   storage,
			MaxSizePerMsg:             1024 * 1024,
			MaxInflightMsgs:           256,
			Rand:                      nil,
			MaxUncommittedEntriesSize: 1 << 30,
			Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
			CheckQuorum:               true,
		})
		for _, c := range confChanges {
			node.ApplyConfChange(c)
		}
		r.curStates[nodeID] = node.Status()
		r.nodes[nodeID] = node
	}
}

func (r *RaftEnvironment) Reset(ctx *FuzzContext) {
	r.makeNodes()
}

func (r *RaftEnvironment) Step(ctx *FuzzContext, m pb.Message) {
	defer func(c *FuzzContext) {
		if r := recover(); r != nil {
			c.traceCtx.SetError(fmt.Errorf("panic in Step: %v", r))
		}
	}(ctx)
	if m.Type == pb.MsgProp {
		// TODO: handle proposal separately
		haveLeader := false
		leader := uint64(0)
		for id, node := range r.nodes {
			if node.Status().RaftState == raft.StateLeader {
				haveLeader = true
				leader = id
				break
			}
		}
		if haveLeader {
			m.To = leader
			request, _ := strconv.Atoi(string(m.Entries[0].Data))
			ctx.AddEvent(&Event{
				Name: "ClientRequest",
				Node: leader,
				Params: map[string]interface{}{
					"request": request,
					"leader":  leader,
				},
			})
			r.nodes[leader].Step(m)
		}
	} else {
		node, ok := r.nodes[m.To]
		if ok {
			node.Step(m)
		}
	}
}

func (r *RaftEnvironment) Tick(ctx *FuzzContext) []pb.Message {
	result := make([]pb.Message, 0)
	// Take random number of ticks and update node states
	for _, node := range r.nodes {
		for i := 0; i < r.config.TicksPerStep; i++ {
			node.Tick()
		}
	}
	r.updateStates(ctx)
	for id, node := range r.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				r.storages[id].ApplySnapshot(ready.Snapshot)
			}
			r.storages[id].Append(ready.Entries)
			result = append(result, ready.Messages...)
			if len(ready.CommittedEntries) > 0 {
				ctx.AddEvent(&Event{
					Name: "AdvanceCommitIndex",
					Node: id,
					Params: map[string]interface{}{
						"i": int(id),
					},
				})
			}
			node.Advance(ready)
		}
	}
	return result
}

func (r *RaftEnvironment) updateStates(ctx *FuzzContext) {
	for id, node := range r.nodes {
		newStatus := node.Status()
		// Compare state and add timeouts
		old := r.curStates[id].RaftState
		new := newStatus.RaftState
		oldTerm := r.curStates[id].Term
		newTerm := newStatus.Term
		if old != new && new == raft.StateLeader {
			ctx.AddEvent(&Event{
				Name: "BecomeLeader",
				Node: id,
				Params: map[string]interface{}{
					"node": id,
				},
			})
			ctx.AddEvent(&Event{
				Name: "ClientRequest",
				Node: id,
				Params: map[string]interface{}{
					"request": 0,
					"leader":  id,
				},
			})
		} else if (old != new && new == raft.StateCandidate) || (oldTerm < newTerm && old == new && new == raft.StateCandidate) {
			ctx.AddEvent(&Event{
				Name: "Timeout",
				Node: id,
				Params: map[string]interface{}{
					"node": id,
				},
			})
		}
		r.curStates[id] = newStatus
	}
}

func (r *RaftEnvironment) Stop(ctx *FuzzContext, node uint64) {
	delete(r.nodes, node)
}

func (r *RaftEnvironment) Start(ctx *FuzzContext, nodeID uint64) {
	if storage, ok := r.storages[nodeID]; ok {
		node, err := raft.NewRawNode(&raft.Config{
			ID:                        nodeID,
			ElectionTick:              r.config.ElectionTick,
			HeartbeatTick:             r.config.HeartbeatTick,
			Storage:                   storage,
			MaxSizePerMsg:             1024 * 1024,
			MaxInflightMsgs:           256,
			Rand:                      nil,
			MaxUncommittedEntriesSize: 1 << 30,
			Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
			CheckQuorum:               true,
		})
		r.nodes[nodeID] = node
		if err != nil {
			ctx.traceCtx.SetError(fmt.Errorf("error starting node: %v", err))
		}
	}
}
