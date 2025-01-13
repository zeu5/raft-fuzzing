package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	episodes     int
	horizon      int
	savePath     string
	replicas     int
	requests     int
	numRuns      int
	recordTraces bool
)

func main() {
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 100, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&savePath, "save", "s", "results", "Save the results to the specified path")
	rootCommand.PersistentFlags().IntVarP(&replicas, "replicas", "r", 3, "Num of replicas to run in environment")
	rootCommand.PersistentFlags().IntVar(&requests, "requests", 1, "Num of initial requests to serve")
	rootCommand.PersistentFlags().IntVar(&numRuns, "runs", 5, "Number of runs to average over")
	rootCommand.PersistentFlags().BoolVar(&recordTraces, "record-traces", false, "Record the traces explored")
	rootCommand.AddCommand(FuzzCommand())
	rootCommand.AddCommand(OneCommand())
	rootCommand.AddCommand(MeasureCommand())

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
	}
}

func MeasureCommand() *cobra.Command {
	var tracesPath string
	var tlcAddr string
	var outPath string

	cmd := &cobra.Command{
		Use: "measure",
		Run: func(cmd *cobra.Command, args []string) {
			m := NewTLCCoverageMeasurer(tracesPath, outPath, tlcAddr)
			if err := m.Measure(); err != nil {
				fmt.Println(err)
			}
		},
	}
	cmd.Flags().StringVar(&tracesPath, "traces", "traces", "Path to traces")
	cmd.Flags().StringVar(&tlcAddr, "tlc", "tlc", "TLC Server address")
	cmd.Flags().StringVar(&outPath, "out", "out", "Output path")

	return cmd
}

func FuzzCommand() *cobra.Command {
	return &cobra.Command{
		Use: "fuzz",
		RunE: func(cmd *cobra.Command, args []string) error {
			fuzzer := NewFuzzer(&FuzzerConfig{
				Iterations: episodes,
				Steps:      horizon,
				Strategy:   NewRandomStrategy(),
				Guider:     NewLineCoverageGuider("127.0.0.1:2023", "traces", recordTraces),
				Mutator:    &EmptyMutator{},
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas:      replicas,
					ElectionTick:  20,
					HeartbeatTick: 2,
					TicksPerStep:  2,
				},
				MutPerTrace:        5,
				NumberRequests:     requests,
				CrashQuota:         2,
				MaxMessages:        10,
				SeedPopulationSize: 10,
			})
			fuzzer.Run()
			return nil
		},
	}
}

func OneCommand() *cobra.Command {
	return &cobra.Command{
		Use: "compare",
		Run: func(cmd *cobra.Command, args []string) {

			c := NewComparision(savePath, &FuzzerConfig{
				Iterations: episodes,
				Steps:      horizon,
				Strategy:   NewRandomStrategy(),
				Mutator:    &EmptyMutator{},
				Checker:    SerializabilityChecker(),
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas: replicas,
					// Lower election tick gives random better chances. (more timeouts)
					ElectionTick:  20,
					HeartbeatTick: 4,
					// Should not be more than ElectionTick/(replica+1) otherwise you are more likely to starve processes
					TicksPerStep: 3,
				},
				// Too much is bad, can lead to very local search
				MutPerTrace:    5,
				NumberRequests: requests,
				// More makes random worse
				CrashQuota: 10,
				// Too few messages are better for random
				MaxMessages:        5,
				SeedPopulationSize: 10,
				ReseedFrequency:    2000,
			}, numRuns)
			combinedMutator := CombineMutators(NewSwapCrashNodeMutator(2), NewSwapNodeMutator(20), NewSwapMaxMessagesMutator(20))
			c.Add("traceCov", combinedMutator, NewTraceCoverageGuider("127.0.0.1:2023", "traces", recordTraces))
			c.Add("lineCov", combinedMutator, NewLineCoverageGuider("127.0.0.1:2023", "traces", recordTraces))
			c.Add("tlcstate", combinedMutator, NewTLCStateGuider("127.0.0.1:2023", "traces", recordTraces))
			c.Add("random", &EmptyMutator{}, NewTLCStateGuider("127.0.0.1:2023", "traces", recordTraces))

			c.Run()
		},
	}
}
