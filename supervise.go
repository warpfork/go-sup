package sup

import (
	"context"
)

// Supervisor is a marker interface for supervisor implementations.
//
// It has no real functional purpose -- it's mostly to make godoc show
// you the supervisor creation methods in one group :)
type Supervisor interface {
	NamedTask
	_Supervisor()
}

// SuperviseRoot takes a supervisor and runs it in the current goroutine.
//
// (You can call `Run()` on a Supervisor yourself; however, you should almost
// certainly prefer to use this method instead, because you will get panic
// recovery, task name and path annotations, and all the usual features of
// go-sup.)
func SuperviseRoot(
	ctx context.Context,
	root Supervisor,
) error {
	return superviseRoot{}.init(root).Run(ctx)
}

// SupervisorForkJoin creates a Supervisor which will launch and handle
// a goroutine for each of the given set of tasks.
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
