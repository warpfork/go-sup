package sup

func SuperviseForkJoin(
	taskGroupName string,
	tasks []Task,
	opts ...SupervisionOptions,
) NamedTask {
	return superviseFJ{name: taskGroupName}.init(tasks)
}

// Placeholder.
//
// ex:
//   - goroutineBucketSize(10)
//   - convertPanics(false)
//   - logRunaways(os.Stderr, 2*time.Second)
type SupervisionOptions func()
