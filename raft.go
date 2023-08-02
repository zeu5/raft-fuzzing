package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/zeu5/raft-fuzzing/raft"
	pb "github.com/zeu5/raft-fuzzing/raft/raftpb"
)

type RaftRand struct {
	rand *rand.Rand
	ctx  *FuzzContext
	lock *sync.Mutex
}

var _ raft.Rand = &RaftRand{}

func NewRaftRand() *RaftRand {
	return &RaftRand{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
		ctx:  nil,
		lock: new(sync.Mutex),
	}
}

func (r *RaftRand) Intn(max int) int {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.ctx == nil {
		return r.rand.Intn(max)
	}
	return r.ctx.RandomIntegerChoice(max)
}

func (r *RaftRand) UpdateCtx(ctx *FuzzContext) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.ctx = ctx
}

type RaftEnvironmentConfig struct {
	Replicas      int
	ElectionTick  int
	HeartbeatTick int
}

type RaftEnvironment struct {
	config         RaftEnvironmentConfig
	nodes          map[uint64]*raft.RawNode
	storages       map[uint64]*raft.MemoryStorage
	curStates      map[uint64]raft.Status
	raftRand       *RaftRand
	curCommitIndex uint64
}

func NewRaftEnvironment(config RaftEnvironmentConfig) *RaftEnvironment {
	r := &RaftEnvironment{
		config:         config,
		nodes:          make(map[uint64]*raft.RawNode),
		storages:       make(map[uint64]*raft.MemoryStorage),
		curStates:      make(map[uint64]raft.Status),
		curCommitIndex: 0,
		raftRand:       NewRaftRand(),
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
			Rand:                      r.raftRand,
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
	r.curCommitIndex = 0
}

func (r *RaftEnvironment) Reset(ctx *FuzzContext) {
	r.raftRand.UpdateCtx(ctx)
	r.makeNodes()
}

func (r *RaftEnvironment) Step(ctx *FuzzContext, m pb.Message) []pb.Message {
	result := make([]pb.Message, 0)
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
				Params: map[string]interface{}{
					"request": request,
					"leader":  leader,
				},
			})
			r.nodes[leader].Step(m)
		} else {
			result = append(result, m)
		}
	} else {
		node, ok := r.nodes[m.To]
		if ok {
			node.Step(m)
		}
	}

	// Take random number of ticks and update node states
	for _, node := range r.nodes {
		node.Tick()
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
				Params: map[string]interface{}{
					"node": id,
				},
			})
			ctx.AddEvent(&Event{
				Name: "ClientRequest",
				Params: map[string]interface{}{
					"request": 0,
					"leader":  id,
				},
			})
		} else if (old != new && new == raft.StateCandidate) || (oldTerm < newTerm && old == new && new == raft.StateCandidate) {
			ctx.AddEvent(&Event{
				Name: "Timeout",
				Params: map[string]interface{}{
					"node": id,
				},
			})
		}
		r.curStates[id] = newStatus
		ctx.AddEvent(&Event{
			Name: "StateUpdate",
			Params: map[string]interface{}{
				"node": id,
				"state": fmt.Sprintf(`{"id":"%x","term":%d,"vote":"%x","commit":%d,"lead":"%x","raftState":%q,"applied":%d}`,
					newStatus.ID, newStatus.Term, newStatus.Vote, newStatus.Commit, newStatus.Lead, newStatus.RaftState, newStatus.Applied),
			},
		})
	}
}

func (r *RaftEnvironment) Stop(ctx *FuzzContext, node uint64) {
	delete(r.nodes, node)
	// TODO: update this
	ctx.AddEvent(&Event{
		Name: "Remove",
		Params: map[string]interface{}{
			"i": node,
		},
	})
}

func (r *RaftEnvironment) Start(ctx *FuzzContext, nodeID uint64) {
	if storage, ok := r.storages[nodeID]; ok {
		node, _ := raft.NewRawNode(&raft.Config{
			ID:                        nodeID,
			ElectionTick:              r.config.ElectionTick,
			HeartbeatTick:             r.config.HeartbeatTick,
			Storage:                   storage,
			MaxSizePerMsg:             1024 * 1024,
			MaxInflightMsgs:           256,
			Rand:                      r.raftRand,
			MaxUncommittedEntriesSize: 1 << 30,
			Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
			CheckQuorum:               true,
		})
		r.nodes[nodeID] = node
		ctx.AddEvent(&Event{
			Name: "Add",
			Params: map[string]interface{}{
				"i": nodeID,
			},
		})
	}
}
