package main

import (
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
	NumRequests   int
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

func (r *RaftEnvironment) Setup(ctx *FuzzContext) {
	r.raftRand.UpdateCtx(ctx)
}

func (r *RaftEnvironment) makeNodes() {
	peers := make([]raft.Peer, r.config.Replicas)
	for i := 0; i < r.config.Replicas; i++ {
		peers[i] = raft.Peer{ID: uint64(i + 1)}
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
		})
		node.Bootstrap(peers)
		r.curStates[nodeID] = node.Status()
		r.nodes[nodeID] = node
	}
	r.curCommitIndex = 0
}

func (r *RaftEnvironment) Reset() []pb.Message {
	messages := make([]pb.Message, r.config.NumRequests)
	for i := 0; i < r.config.NumRequests; i++ {
		messages[i] = pb.Message{
			Type: pb.MsgProp,
			From: uint64(0),
			Entries: []pb.Entry{
				{Data: []byte(strconv.Itoa(i + 1))},
			},
		}
	}
	r.makeNodes()
	return messages
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
		node := r.nodes[m.To]
		node.Step(m)
	}

	// Take random number of ticks and update node states
	for _, node := range r.nodes {
		ticks := max(1, int(r.config.ElectionTick/5)) // ctx.RandomIntegerChoice(r.config.ElectionTick)
		for i := 0; i < ticks; i++ {
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
			node.Advance(ready)
		}
	}
	return result
}

func (r *RaftEnvironment) updateStates(ctx *FuzzContext) {
	for id, node := range r.nodes {
		// Compare state and add timeouts
		old := r.curStates[id].RaftState
		new := node.Status().RaftState
		if old != new && new == raft.StateLeader {
			ctx.AddEvent(&Event{
				Name: "BecomeLeader",
				Params: map[string]interface{}{
					"node": id,
				},
			})
		} else if old != new && new == raft.StateCandidate {
			ctx.AddEvent(&Event{
				Name: "Timeout",
				Params: map[string]interface{}{
					"node": id,
				},
			})
		}
		// Compare commit index of leader and add advance commit index
		if new == raft.StateLeader {
			oldCommitIndex := r.curStates[id].Commit
			newCommitIndex := node.Status().Commit
			if newCommitIndex > oldCommitIndex {
				ctx.AddEvent(&Event{
					Name: "AdvanceCommitIndex",
					Params: map[string]interface{}{
						"node": id,
					},
				})
			}
		}
		r.curStates[id] = node.Status()
	}
}
