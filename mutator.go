package main

type EmptyMutator struct {
}

var _ Mutator = &EmptyMutator{}

func (e *EmptyMutator) Mutate(schedulerTrace *List[*SchedulingChoice], trace *List[*Event]) (*List[*SchedulingChoice], bool) {
	return nil, false
}
