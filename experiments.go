package main

func CompareMutations(episodes, horizon int, saveFile string, replicas, requests int) {
	c := NewComparision(saveFile, &FuzzerConfig{
		Iterations: episodes,
		Steps:      horizon,
		Strategy:   NewRoundRobinStrategy(replicas),
		Mutator:    &EmptyMutator{},
		RaftEnvironmentConfig: RaftEnvironmentConfig{
			Replicas:      replicas,
			ElectionTick:  16,
			HeartbeatTick: 2,
			NumRequests:   requests,
		},
		MutPerTrace: 5,
	})
	c.AddGuider("tlcstate", NewTLCStateGuider("127.0.0.1:2023", "traces", false))
	c.AddMutator("random", &EmptyMutator{})
	c.AddMutator("swapInt", NewSwapIntegerChoiceMutator(5))
	c.AddMutator("flipChoices", NewChoiceMutator(5))
	c.AddMutator("skipNodes", NewSkipNodeMutator(5))
	c.AddMutator("swapNodes", NewSwapNodeMutator(5))
	c.AddMutator("scaleDownInt", NewScaleDownIntChoiceMutator(5))

	c.Run()
}
