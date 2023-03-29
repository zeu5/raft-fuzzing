package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	episodes int
	horizon  int
	saveFile string
)

func main() {
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 50, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&saveFile, "save", "s", "save.png", "Save the plot to the specified file")
	rootCommand.AddCommand(FuzzCommand())

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
				TLCAddr:    "127.0.0.1:2023",
				Mutator:    &EmptyMutator{},
				RaftEnvironmentConfig: RaftEnvironmentConfig{
					Replicas:      3,
					ElectionTick:  10,
					HeartbeatTick: 2,
				},
			})
			return fuzzer.Run()
		},
	}
}
