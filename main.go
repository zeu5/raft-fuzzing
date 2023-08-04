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
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 50, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&savePath, "save", "s", "results", "Save the results to the specified path")
	rootCommand.PersistentFlags().IntVarP(&replicas, "replicas", "r", 3, "Num of replicas to run in environment")
	rootCommand.PersistentFlags().IntVar(&requests, "requests", 1, "Num of initial requests to serve")
	rootCommand.PersistentFlags().IntVar(&numRuns, "runs", 5, "Number of runs to average over")
	rootCommand.PersistentFlags().BoolVar(&recordTraces, "record-traces", false, "Record the traces explored")
	rootCommand.AddCommand(FuzzCommand())
	rootCommand.AddCommand(OneCommand())

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
	}
}

func FuzzCommand() *cobra.Command {
	return &cobra.Command{
		Use: "fuzz",
		RunE: func(cmd *cobra.Command, args []string) error {
			fuzzer := NewFuzzer(&FuzzerConfig{
				Iterations: episodes,
				Steps:      horizon,
				Strategy:   NewRandomStrategy(),
				Guider:     NewTLCStateGuider("127.0.0.1:2023", "traces", recordTraces),
				Mutator:    &EmptyMutator{},
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas:      replicas,
					ElectionTick:  20,
					HeartbeatTick: 2,
					TicksPerStep:  2,
				},
				MutPerTrace:       5,
				NumberRequests:    requests,
				CrashQuota:        2,
				MaxMessages:       10,
				InitialPopulation: 10,
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
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas:      replicas,
					ElectionTick:  12,
					HeartbeatTick: 2,
					TicksPerStep:  3,
				},
				MutPerTrace:       6,
				NumberRequests:    requests,
				CrashQuota:        5,
				MaxMessages:       5,
				InitialPopulation: 10,
			}, numRuns)
			c.AddGuider("tlcstate", NewTLCStateGuider("127.0.0.1:2023", "traces", recordTraces))
			c.AddMutator("random", &EmptyMutator{})
			// c.AddMutator("swapCrashNodes", NewSwapCrashNodeMutator(1))
			// c.AddMutator("swapMaxMessages", NewSwapMaxMessagesMutator(10))
			c.AddMutator("all_mutators", CombineMutators(NewSwapCrashNodeMutator(2), NewSwapNodeMutator(20), NewSwapMaxMessagesMutator(20)))

			c.Run()
		},
	}
}
