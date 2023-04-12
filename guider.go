package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type CoverageStats struct {
	UniqueStates      int
	UniqueTraces      int
	UniqueStateTraces int
}

type Guider interface {
	Check(*List[*SchedulingChoice], *List[*Event]) bool
	Coverage() CoverageStats
}

type TLCStateGuider struct {
	TLCAddr        string
	statesMap      map[int64]bool
	tracesMap      map[string]bool
	stateTracesMap map[string]bool
	tlcClient      *TLCClient
}

var _ Guider = &TLCStateGuider{}

func NewTLCStateGuider(tlcAddr string) *TLCStateGuider {
	return &TLCStateGuider{
		TLCAddr:        tlcAddr,
		statesMap:      make(map[int64]bool),
		tracesMap:      make(map[string]bool),
		stateTracesMap: make(map[string]bool),
		tlcClient:      NewTLCClient(tlcAddr),
	}
}

func (t *TLCStateGuider) Coverage() CoverageStats {
	return CoverageStats{
		UniqueStates:      len(t.statesMap),
		UniqueTraces:      len(t.tracesMap),
		UniqueStateTraces: len(t.stateTracesMap),
	}
}

func (t *TLCStateGuider) Check(trace *List[*SchedulingChoice], eventTrace *List[*Event]) bool {
	bs, _ := json.Marshal(trace)
	sum := sha256.Sum256(bs)
	hash := hex.EncodeToString(sum[:])
	if _, ok := t.tracesMap[hash]; !ok {
		t.tracesMap[hash] = true
	}

	haveNewState := false
	if tlcStates, err := t.tlcClient.SendTrace(eventTrace); err == nil {
		for _, s := range tlcStates {
			_, ok := t.statesMap[s.Key]
			if !ok {
				haveNewState = true
				t.statesMap[s.Key] = true
			}
		}
		bs, _ := json.Marshal(tlcStates)
		sum := sha256.Sum256(bs)
		stateTraceHash := hex.EncodeToString(sum[:])
		if _, ok := t.stateTracesMap[stateTraceHash]; !ok {
			t.stateTracesMap[stateTraceHash] = true
		}
	}
	return haveNewState
}
