package main

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type Comparision struct {
	config   *FuzzerConfig
	mutators map[string]Mutator
	plotFile string

	coverages map[string][]int
}

func NewComparision(plotFile string, config *FuzzerConfig) *Comparision {
	return &Comparision{
		plotFile:  plotFile,
		config:    config,
		coverages: make(map[string][]int),
	}
}

func (c *Comparision) AddMutator(name string, mutator Mutator) {
	c.mutators[name] = mutator
	c.coverages[name] = make([]int, 0)
}

func (c *Comparision) Run() {
	for name, m := range c.mutators {
		c.config.Mutator = m
		fuzzer := NewFuzzer(c.config)
		for i := 0; i < c.config.Iterations; i++ {
			fuzzer.RunIteration(i)
			c.coverages[name] = append(c.coverages[name], fuzzer.Coverage())
		}
	}
	c.record()
}

func (c *Comparision) record() {
	p := plot.New()
	p.Title.Text = "Comparison"
	p.X.Label.Text = "Iteration"
	p.Y.Label.Text = "States covered"

	datasetPlotter := func(name string) plotter.XYs {
		points := make(plotter.XYs, len(c.coverages[name]))
		for i, v := range c.coverages[name] {
			points[i] = plotter.XY{
				X: float64(i),
				Y: float64(v),
			}
		}
		return points
	}
	i := 0
	for name := range c.coverages {
		line, err := plotter.NewLine(datasetPlotter(name))
		if err != nil {
			continue
		}
		line.Color = plotutil.Color(i)
		p.Add(line)
		p.Legend.Add(name, line)
		i++
	}
	p.Save(4*vg.Inch, 4*vg.Inch, c.plotFile)
}
