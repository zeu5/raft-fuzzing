package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	episodes int
	horizon  int
	saveFile string
	replicas int
	requests int
)

func main() {
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 50, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&saveFile, "save", "s", "save.png", "Save the plot to the specified file")
	rootCommand.PersistentFlags().IntVarP(&replicas, "replicas", "r", 3, "Num of replicas to run in environment")
	rootCommand.PersistentFlags().IntVar(&requests, "requests", 1, "Num of initial requests to serve")
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
				Guider:     NewTLCStateGuider("127.0.0.1:2023", "traces", true),
				Mutator:    NewScaleUpIntChoiceMutator(5, 10),
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas:      replicas,
					ElectionTick:  10,
					HeartbeatTick: 2,
					NumRequests:   requests,
				},
				MutPerTrace: 5,
			})
			return fuzzer.Run()
		},
	}
}

func OneCommand() *cobra.Command {
	return &cobra.Command{
		Use: "compare",
		Run: func(cmd *cobra.Command, args []string) {
			c := NewComparision(saveFile, &FuzzerConfig{
				Iterations: episodes,
				Steps:      horizon,
				Strategy:   NewRandomStrategy(),
				Mutator:    &EmptyMutator{},
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas:      replicas,
					ElectionTick:  20,
					HeartbeatTick: 2,
					NumRequests:   requests,
				},
				MutPerTrace: 3,
			})
			c.AddGuider("tlcstate", NewTLCStateGuider("127.0.0.1:2023", "traces", false))
			c.AddMutator("random", &EmptyMutator{})
			c.AddMutator("scaleUpInt", NewScaleUpIntChoiceMutator(5, 20))
			c.AddMutator("swapNodes", NewSwapNodeMutator(30))
			c.AddMutator("swapNodes_scaleUpInt", CombineMutators(NewScaleUpIntChoiceMutator(5, 20), NewSwapNodeMutator(30)))

			c.Run()
		},
	}
}
