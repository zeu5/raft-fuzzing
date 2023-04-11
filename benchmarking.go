package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type Comparision struct {
	config   *FuzzerConfig
	mutators map[string]Mutator
	plotFile string

	coverages map[string][]CoverageStats
}

func NewComparision(plotFile string, config *FuzzerConfig) *Comparision {
	return &Comparision{
		plotFile:  plotFile,
		config:    config,
		coverages: make(map[string][]CoverageStats),
		mutators:  make(map[string]Mutator),
	}
}

func (c *Comparision) AddMutator(name string, mutator Mutator) {
	c.mutators[name] = mutator
	c.coverages[name] = make([]CoverageStats, 0)
}

func (c *Comparision) Run() {
	for name, m := range c.mutators {
		c.config.Mutator = m
		fuzzer := NewFuzzer(c.config)
		for i := 0; i < c.config.Iterations; i++ {
			fmt.Printf("\rRunning for mutator: %s, Episode: %d/%d", name, i+1, c.config.Iterations)
			fuzzer.RunIteration(i)
			c.coverages[name] = append(c.coverages[name], fuzzer.Coverage())
		}
		fmt.Println("")
	}
	c.record()
}

func (c *Comparision) record() {
	p := plot.New()
	p.Title.Text = "Comparison"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "States covered"

	i := 0
	for name, points := range c.coverages {
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
		line.Color = plotutil.Color(i)
		p.Add(line)
		p.Legend.Add(name, line)

		fmt.Printf("Coverage for mutator %s is %d\n", name, points[len(points)-1])
		i++
	}
	p.Save(4*vg.Inch, 4*vg.Inch, c.plotFile)

	finalCoverage := make(map[string]CoverageStats)
	for name, points := range c.coverages {
		finalCoverage[name] = points[len(points)-1]
	}
	cov, _ := json.Marshal(finalCoverage)
	os.WriteFile("cov.json", cov, 0644)
}