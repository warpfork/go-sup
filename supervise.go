package sup

// Supervisor is a marker interface for supervisor implementations.
//
// It has no real functional purpose -- it's mostly to make godoc show
// you the supervisor creation methods in one group :)
type Supervisor interface {
	NamedTask
	_Supervisor()
}

func SuperviseForkJoin(
	taskGroupName string,
	tasks []Task,
	opts ...SupervisionOptions,
) Supervisor {
	return superviseFJ{name: taskGroupName}.init(tasks)
}

// Placeholder.
//
// ex:
//   - goroutineBucketSize(10)
//   - convertPanics(false)
//   - logRunaways(os.Stderr, 2*time.Second)
type SupervisionOptions func()
