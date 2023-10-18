package main

import (
	"bytes"

	"github.com/zeu5/raft-fuzzing/raft"
	pb "github.com/zeu5/raft-fuzzing/raft/raftpb"
)

type Checker func(*RaftEnvironment) bool

func SerializabilityChecker() func(*RaftEnvironment) bool {
	return func(re *RaftEnvironment) bool {
		minCommit := 100
		for _, state := range re.curStates {
			if state.Commit < uint64(minCommit) {
				minCommit = int(state.Commit)
			}
		}
		if minCommit == 0 {
			return true
		}
		logs := make([][]pb.Entry, 0)
		for _, storage := range re.storages {
			l, err := storage.Entries(1, uint64(minCommit)+1, 100)
			if err != nil {
				return false
			}
			logs = append(logs, l)
		}

		for i := 0; i < minCommit; i++ {
			l := logs[0][i]
			for j := 1; j < len(logs); j++ {
				cur := logs[j][i]
				if cur.Term != l.Term || cur.Index != l.Index || !bytes.Equal(cur.Data, l.Data) {
					return false
				}
			}
		}
		return true
	}
}

func SingleLeader() func(*RaftEnvironment) bool {
	return func(re *RaftEnvironment) bool {
		leaders := 0
		for _, s := range re.curStates {
			if s.RaftState == raft.StateLeader {
				leaders += 1
			}
		}
		return leaders <= 1
	}
}
