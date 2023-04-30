package main

func CompareMutations(episodes, horizon int, saveFile string, replicas, requests int) {
	c := NewComparision(saveFile, &FuzzerConfig{
		Iterations: episodes,
		Steps:      horizon,
		Strategy:   NewRoundRobinStrategy(replicas),
		Mutator:    &EmptyMutator{},
		RaftEnvironmentConfig: RaftEnvironmentConfig{
			Replicas:      replicas,
			ElectionTick:  10,
			HeartbeatTick: 2,
			NumRequests:   requests,
		},
		MutPerTrace: 3,
	})
	c.AddGuider("tlcstate", NewTLCStateGuider("127.0.0.1:2023", "traces", false))
	c.AddMutator("random", &EmptyMutator{})
	c.AddMutator("scaleUpInt", NewScaleUpIntChoiceMutator(5, 10))
	// c.AddMutator("swapNodes", NewSwapNodeMutator(40))
	c.AddMutator("swapNodes_scaleUpInt", CombineMutators(NewScaleUpIntChoiceMutator(5, 10), NewSwapNodeMutator(30)))

	c.Run()
}
