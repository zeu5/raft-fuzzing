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
	guiders  map[string]Guider
	plotFile string

	coverages map[string][]CoverageStats
}

func NewComparision(plotFile string, config *FuzzerConfig) *Comparision {
	return &Comparision{
		plotFile:  plotFile,
		config:    config,
		coverages: make(map[string][]CoverageStats),
		mutators:  make(map[string]Mutator),
		guiders:   make(map[string]Guider),
	}
}

func (c *Comparision) AddMutator(name string, mutator Mutator) {
	c.mutators[name] = mutator
}

func (c *Comparision) AddGuider(name string, guider Guider) {
	c.guiders[name] = guider
}

func (c *Comparision) Run() {
	fmt.Println("Starting comparision...")
	for guiderName, guider := range c.guiders {
		for mutatorName, mutator := range c.mutators {
			key := mutatorName + "_" + guiderName
			c.config.Guider = guider
			c.config.Mutator = mutator
			c.coverages[key] = make([]CoverageStats, 0)
			fuzzer := NewFuzzer(c.config)
			for i := 0; i < c.config.Iterations; i++ {
				fmt.Printf("\rRunning for mutator: %s, guider: %s, Episode: %d/%d", mutatorName, guiderName, i+1, c.config.Iterations)
				fuzzer.RunIteration(i)
				c.coverages[key] = append(c.coverages[key], guider.Coverage())
			}
			fmt.Println("")
			// Reset guider
			guider.Reset(mutatorName)
		}
	}
	fmt.Printf("Completed running.\nStarting analysis...\n")
	c.record()
	fmt.Println("Completed analysis.")
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
	if cov, err := json.Marshal(finalCoverage); err == nil {
		os.WriteFile("cov.json", cov, 0644)
	}
}
