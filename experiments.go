package main

func CompareMutations(episodes, horizon int, saveFile string, replicas, requests int) {
	c := NewComparision(saveFile, &FuzzerConfig{
		Iterations: episodes,
		Steps:      horizon,
		TLCAddr:    "127.0.0.1:2023",
		Mutator:    &EmptyMutator{},
		RaftEnvironmentConfig: RaftEnvironmentConfig{
			Replicas:      replicas,
			ElectionTick:  10,
			HeartbeatTick: 2,
			NumRequests:   requests,
		},
		MutPerTrace: 5,
	})
	c.AddMutator("random", &EmptyMutator{})
	c.AddMutator("flipChoices", NewChoiceMutator(5))
	c.AddMutator("skipNodes", NewSkipNodeMutator(5))
	c.AddMutator("swapNodes", NewSwapNodeMutator(5))

	c.Run()
}

// Goals
// 1. Play with timeouts
// 2.
