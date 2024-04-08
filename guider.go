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

	"github.com/zeu5/gocov"
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

type TraceCoverageGuider struct {
	traces map[string]bool
	*TLCStateGuider
}

var _ Guider = &TraceCoverageGuider{}

func NewTraceCoverageGuider(tlcAddr, recordPath string, recordTraces bool) *TraceCoverageGuider {
	return &TraceCoverageGuider{
		traces:         make(map[string]bool),
		TLCStateGuider: NewTLCStateGuider(tlcAddr, recordPath, recordTraces),
	}
}

func (t *TraceCoverageGuider) Check(trace *List[*SchedulingChoice], events *List[*Event]) (int, float64) {
	t.TLCStateGuider.Check(trace, events)

	eTrace := newEventTrace(events)
	key := eTrace.Hash()

	new := 0
	if _, ok := t.traces[key]; !ok {
		t.traces[key] = true
		new = 1
	}

	return new, float64(new) / float64(len(t.traces))
}

func (t *TraceCoverageGuider) Coverage() CoverageStats {
	c := t.TLCStateGuider.Coverage()
	c.UniqueTraces = len(t.traces)
	return c
}

func (t *TraceCoverageGuider) Reset(key string) {
	t.traces = make(map[string]bool)
	t.TLCStateGuider.Reset(key)
}

type eventTrace struct {
	Nodes map[string]*eventNode
}

func (e *eventTrace) Hash() string {
	bs, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

type eventNode struct {
	*Event
	Node uint64
	Prev string
	ID   string `json:"-"`
}

func (e *eventNode) Hash() string {
	bs, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func newEventTrace(events *List[*Event]) *eventTrace {
	eTrace := &eventTrace{
		Nodes: make(map[string]*eventNode),
	}
	curEvent := make(map[uint64]*eventNode)

	for _, e := range events.Iter() {
		node := &eventNode{
			Event: e,
			Node:  e.Node,
			Prev:  "",
		}
		prev, ok := curEvent[e.Node]
		if ok {
			node.Prev = prev.ID
		}
		node.ID = node.Hash()
		curEvent[e.Node] = node
		eTrace.Nodes[node.ID] = node
	}
	return eTrace
}

type LineCoverageGuider struct {
	covData *gocov.Coverage
	*TLCStateGuider
}

func NewLineCoverageGuider(tlcAddr, recordPath string, recordTraces bool) *LineCoverageGuider {
	return &LineCoverageGuider{
		covData:        nil,
		TLCStateGuider: NewTLCStateGuider(tlcAddr, recordPath, recordTraces),
	}
}

var _ Guider = &LineCoverageGuider{}

func (l *LineCoverageGuider) Check(trace *List[*SchedulingChoice], events *List[*Event]) (int, float64) {
	l.TLCStateGuider.Check(trace, events)
	cov, err := gocov.GetCoverage(gocov.CoverageConfig{
		MatchPkgs: []string{"github.com/zeu5/raft-fuzzing/raft"},
	})
	if err != nil {
		return 0, 0
	}
	if l.covData == nil {
		l.covData = cov
		return cov.GetCoveredLines(), 1
	}
	curLines := l.covData.GetCoveredLines()
	l.covData.Data.Merge(cov.Data)
	updatedLines := l.covData.GetCoveredLines()
	newLines := updatedLines - curLines
	return newLines, float64(newLines) / float64(max(curLines, 1))
}

func (l *LineCoverageGuider) Reset(key string) {
	fmt.Printf("Percentage of lines covered: %f\n", l.covData.GetPercent())
	l.covData.Reset()
	l.covData = nil
	l.TLCStateGuider.Reset(key)
}
