package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
)

type CoverageStats struct {
	UniqueStates      int
	UniqueTraces      int
	UniqueStateTraces int
}

type Guider interface {
	Check(*List[*SchedulingChoice], *List[*Event]) bool
	Coverage() CoverageStats
	Reset()
}

type TLCStateGuider struct {
	TLCAddr        string
	statesMap      map[int64]bool
	tracesMap      map[string]bool
	stateTracesMap map[string]bool
	tlcClient      *TLCClient
	recordPath     string
	count          int
}

var _ Guider = &TLCStateGuider{}

func NewTLCStateGuider(tlcAddr, recordPath string) *TLCStateGuider {
	if recordPath != "" {
		if _, err := os.Stat(recordPath); err == nil {
			os.RemoveAll(recordPath)
		}
		os.Mkdir(recordPath, 0777)
	}
	return &TLCStateGuider{
		TLCAddr:        tlcAddr,
		statesMap:      make(map[int64]bool),
		tracesMap:      make(map[string]bool),
		stateTracesMap: make(map[string]bool),
		tlcClient:      NewTLCClient(tlcAddr),
		recordPath:     recordPath,
		count:          0,
	}
}

func (t *TLCStateGuider) Reset() {
	t.statesMap = make(map[int64]bool)
	t.tracesMap = make(map[string]bool)
	t.stateTracesMap = make(map[string]bool)
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
		t.recordTrace(trace, eventTrace, tlcStates)
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
	} else {
		panic(fmt.Sprintf("error connecting to tlc: %s", err))
	}
	return haveNewState
}

func (t *TLCStateGuider) recordTrace(trace *List[*SchedulingChoice], eventTrace *List[*Event], states []State) {
	if t.recordPath == "" {
		return
	}
	filePath := path.Join(t.recordPath, strconv.Itoa(t.count)+".json")
	t.count += 1
	data := map[string]interface{}{
		"trace":       trace,
		"event_trace": eventTrace,
		"state_trace": states,
	}
	dataB, err := json.Marshal(data)
	if err != nil {
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(dataB)
	writer.Flush()
}
