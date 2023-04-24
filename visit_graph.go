package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path"
)

type VisitGraph struct {
	Nodes map[int64]*VisitGraphNode
}

func NewVisitGraph() *VisitGraph {
	return &VisitGraph{
		Nodes: make(map[int64]*VisitGraphNode),
	}
}

func (v *VisitGraph) IsEmpty() bool {
	return len(v.Nodes) == 0
}

func (v *VisitGraph) Update(trace []State) {
	for i := 0; i < len(trace)-1; i++ {
		cur := trace[i]
		next := trace[i+1]
		if _, ok := v.Nodes[cur.Key]; !ok {
			v.Nodes[cur.Key] = &VisitGraphNode{
				Key:    cur.Key,
				State:  cur.Repr,
				Visits: 0,
				Next:   make(map[int64]bool),
				Prev:   make(map[int64]bool),
			}
		}
		if _, ok := v.Nodes[next.Key]; !ok {
			v.Nodes[next.Key] = &VisitGraphNode{
				Key:    next.Key,
				State:  next.Repr,
				Visits: 0,
				Next:   make(map[int64]bool),
				Prev:   make(map[int64]bool),
			}
		}

		v.Nodes[cur.Key].Visits += 1
		v.Nodes[cur.Key].AddNext(next.Key)
		v.Nodes[next.Key].AddPrev(cur.Key)
	}
	last := trace[len(trace)-1]
	v.Nodes[last.Key].Visits += 1
}

func (v *VisitGraph) record(recordPath string, key string) {
	filePath := path.Join(recordPath, "visit_graph_"+key+".json")
	bs, err := json.Marshal(v)
	if err != nil {
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(bs)
	writer.Flush()
}

type VisitGraphNode struct {
	Key    int64
	State  string
	Visits int
	Next   map[int64]bool `json:",omitempty"`
	Prev   map[int64]bool `json:",omitempty"`
}

func (n *VisitGraphNode) AddNext(next int64) {
	if next == n.Key {
		return
	}
	if _, ok := n.Next[next]; !ok {
		n.Next[next] = true
	}
}

func (n *VisitGraphNode) AddPrev(prev int64) {
	if prev == n.Key {
		return
	}
	if _, ok := n.Prev[prev]; !ok {
		n.Prev[prev] = true
	}
}
