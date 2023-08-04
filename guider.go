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
	"strings"
)

type CoverageStats struct {
	UniqueStates      int
	UniqueTraces      int
	UniqueStateTraces int
}

type Guider interface {
	Check(*List[*SchedulingChoice], *List[*Event]) (int, float64)
	Coverage() CoverageStats
	Reset(string)
}

type TLCStateGuider struct {
	TLCAddr        string
	statesMap      map[int64]bool
	tracesMap      map[string]bool
	stateTracesMap map[string]bool
	tlcClient      *TLCClient
	recordPath     string
	recordTraces   bool
	count          int
}

var _ Guider = &TLCStateGuider{}

func NewTLCStateGuider(tlcAddr, recordPath string, recordTraces bool) *TLCStateGuider {
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
		recordTraces:   recordTraces,
		count:          0,
	}
}

func (t *TLCStateGuider) Reset(key string) {
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

func (t *TLCStateGuider) Check(trace *List[*SchedulingChoice], eventTrace *List[*Event]) (int, float64) {
	bs, _ := json.Marshal(trace)
	sum := sha256.Sum256(bs)
	hash := hex.EncodeToString(sum[:])
	if _, ok := t.tracesMap[hash]; !ok {
		// fmt.Printf("New trace: %s\n", hash)
		t.tracesMap[hash] = true
	}

	curStates := len(t.statesMap)
	numNewStates := 0
	if tlcStates, err := t.tlcClient.SendTrace(eventTrace); err == nil {
		t.recordTrace(trace, eventTrace, tlcStates)
		for _, s := range tlcStates {
			_, ok := t.statesMap[s.Key]
			if !ok {
				numNewStates += 1
				t.statesMap[s.Key] = true
			}
		}
		bs, _ := json.Marshal(tlcStates)
		sum := sha256.Sum256(bs)
		stateTraceHash := hex.EncodeToString(sum[:])
		if _, ok := t.stateTracesMap[stateTraceHash]; !ok {
			// fmt.Printf("New state trace: %s\n", stateTraceHash)
			t.stateTracesMap[stateTraceHash] = true
		}
	} else {
		panic(fmt.Sprintf("error connecting to tlc: %s", err))
	}
	return numNewStates, float64(numNewStates) / float64(max(curStates, 1))
}

func (t *TLCStateGuider) recordTrace(trace *List[*SchedulingChoice], eventTrace *List[*Event], states []State) {
	if !t.recordTraces {
		return
	}
	filePath := path.Join(t.recordPath, strconv.Itoa(t.count)+".json")
	t.count += 1
	data := map[string]interface{}{
		"trace":       trace,
		"event_trace": eventTrace,
		"state_trace": parseTLCStateTrace(states),
	}
	dataB, err := json.MarshalIndent(data, "", "\t")
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

func parseTLCStateTrace(states []State) []State {
	newStates := make([]State, len(states))
	for i, s := range states {
		repr := strings.ReplaceAll(s.Repr, "\n", ",")
		repr = strings.ReplaceAll(repr, "/\\", "")
		repr = strings.ReplaceAll(repr, "\u003e\u003e", "]")
		repr = strings.ReplaceAll(repr, "\u003c\u003c", "[")
		repr = strings.ReplaceAll(repr, "\u003e", ">")
		newStates[i] = State{
			Repr: repr,
			Key:  s.Key,
		}
	}
	return newStates
}
