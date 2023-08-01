package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type Comparision struct {
	config   *FuzzerConfig
	mutators map[string]Mutator
	guiders  map[string]Guider
	plotPath string
	runs     int
	runInfos []runInfo
}

type runInfo struct {
	runTimes  map[string]time.Duration
	coverages map[string][]CoverageStats
}

func NewComparision(plotPath string, config *FuzzerConfig, runs int) *Comparision {
	if plotPath != "" {
		if _, err := os.Stat(plotPath); err == nil {
			os.RemoveAll(plotPath)
		}
		os.Mkdir(plotPath, 0777)
	}

	return &Comparision{
		plotPath: plotPath,
		config:   config,
		runInfos: make([]runInfo, runs),
		runs:     runs,
		mutators: make(map[string]Mutator),
		guiders:  make(map[string]Guider),
	}
}

func (c *Comparision) AddMutator(name string, mutator Mutator) {
	c.mutators[name] = mutator
}

func (c *Comparision) AddGuider(name string, guider Guider) {
	c.guiders[name] = guider
}

func (c *Comparision) doRun(run int) runInfo {
	fmt.Printf("Starting run %d...\n", run+1)
	rI := runInfo{
		runTimes:  make(map[string]time.Duration),
		coverages: make(map[string][]CoverageStats),
	}
	for guiderName, guider := range c.guiders {
		for mutatorName, mutator := range c.mutators {
			key := mutatorName + "_" + guiderName
			c.config.Guider = guider
			c.config.Mutator = mutator
			rI.coverages[key] = make([]CoverageStats, 0)
			fuzzer := NewFuzzer(c.config)
			start := time.Now()
			for i := 0; i < c.config.Iterations; i++ {
				fmt.Printf("\rRunning for mutator: %s, guider: %s, Episode: %d/%d", mutatorName, guiderName, i+1, c.config.Iterations)
				fuzzer.RunIteration(i)
				rI.coverages[key] = append(rI.coverages[key], guider.Coverage())
			}
			end := time.Since(start)
			rI.runTimes[key] = end
			fmt.Printf("\nRun time: %s\n", end.String())
			// Reset guider
			guider.Reset(mutatorName)
		}
	}
	return rI
}

func (c *Comparision) Run() {
	for i := 0; i < c.runs; i++ {
		c.runInfos[i] = c.doRun(i)
	}
	fmt.Printf("Completed running.\nStarting analysis...\n")
	c.record()
	fmt.Println("Completed analysis.")
}

func (c *Comparision) record() {

	runTimes := make(map[string][]time.Duration)
	finalCoverages := make(map[string][]CoverageStats)

	for i := 0; i < c.runs; i++ {
		plotFile := path.Join(c.plotPath, fmt.Sprintf("%d.png", i))
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		k := 0
		for name, points := range c.runInfos[i].coverages {
			plotPoints := make([]plotter.XY, len(points))
			for j, point := range points {
				plotPoints[j] = plotter.XY{
					X: float64(j),
					Y: float64(point.UniqueStates),
				}
			}
			line, err := plotter.NewLine(plotter.XYs(plotPoints))
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(k)
			p.Add(line)
			p.Legend.Add(name, line)

			k++
		}
		p.Save(4*vg.Inch, 4*vg.Inch, plotFile)

		for name, duration := range c.runInfos[i].runTimes {
			if _, ok := runTimes[name]; !ok {
				runTimes[name] = make([]time.Duration, 0)
			}
			runTimes[name] = append(runTimes[name], duration)
		}

		for name, points := range c.runInfos[i].coverages {
			if _, ok := finalCoverages[name]; !ok {
				finalCoverages[name] = make([]CoverageStats, 0)
			}
			finalCoverages[name] = append(finalCoverages[name], points[len(points)-1])
		}
	}

	recordData := make(map[string]map[string]interface{})

	for name, rts := range runTimes {
		if _, ok := recordData[name]; !ok {
			recordData[name] = make(map[string]interface{})
		}
		sum := time.Duration(0)
		for _, rt := range rts {
			sum += rt
		}
		avg := int64(sum) / int64(len(rts))
		recordData[name]["average_runtime"] = time.Duration(avg).String()
		recordData[name]["runtimes"] = rts
	}

	for name, coverages := range finalCoverages {
		sum := CoverageStats{}
		for _, cov := range coverages {
			sum.UniqueStates += cov.UniqueStates
			sum.UniqueStateTraces += cov.UniqueStateTraces
			sum.UniqueTraces += cov.UniqueTraces
		}
		avg := CoverageStats{
			UniqueStates:      sum.UniqueStates / len(coverages),
			UniqueStateTraces: sum.UniqueStateTraces / len(coverages),
			UniqueTraces:      sum.UniqueTraces / len(coverages),
		}
		recordData[name]["average_coverage"] = avg
		recordData[name]["coverages"] = coverages
	}

	recordPath := path.Join(c.plotPath, "data.json")

	if cov, err := json.Marshal(recordData); err == nil {
		os.WriteFile(recordPath, cov, 0644)
	}
}
