package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TLCCoverageMeasurer struct {
	tracesPath string
	tlcAddr    string
	outPath    string

	tlcClient *TLCClient
	cov       map[int64]int
}

func NewTLCCoverageMeasurer(tracesPath, outPath, tlcAddr string) *TLCCoverageMeasurer {
	return &TLCCoverageMeasurer{
		tracesPath: tracesPath,
		tlcAddr:    tlcAddr,

		tlcClient: NewTLCClient(tlcAddr),
	}
}

func (p *TLCCoverageMeasurer) Measure() error {
	traces, err := p.loadTraces()
	if err != nil {
		return fmt.Errorf("error loading traces: %s", err)
	}
	coverages := make([]int, 0)
	coverages = append(coverages, 0)
	for _, trace := range traces {
		states, err := p.tlcClient.SendTrace(trace)
		if err != nil {
			return fmt.Errorf("error sending trace to tlc: %s", err)
		}
		for _, state := range states {
			p.cov[state.Key]++
		}
		coverages = append(coverages, len(states))
	}
	jsonData, err := json.Marshal(map[string]interface{}{
		"coverages": coverages,
	})
	if err != nil {
		return fmt.Errorf("error marshalling json: %s", err)
	}
	if err = os.WriteFile(filepath.Join(p.outPath, "coverage.json"), jsonData, 0644); err != nil {
		return fmt.Errorf("error writing coverage file: %s", err)
	}
	return nil
}

func (p *TLCCoverageMeasurer) loadTraces() ([]*List[*Event], error) {
	traces := make([]*List[*Event], 0)
	files, err := os.ReadDir(p.tracesPath)
	if err != nil {
		return traces, fmt.Errorf("error reading traces directory: %s", err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.tracesPath, file.Name()))
		if err != nil {
			return traces, fmt.Errorf("error reading trace file: %s", err)
		}
		trace := &List[*Event]{}
		if err = json.Unmarshal(data, trace); err != nil {
			return traces, fmt.Errorf("error parsing trace file: %s", err)
		}
		traces = append(traces, trace)
	}
	return traces, nil
}
