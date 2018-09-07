package sup

import (
	"context"
)

// Supervisor is a marker interface for supervisor implementations.
// All Supervisors are themselves a perfectly normal Task, plus some additional
// methods which allow monitoring their status.
//
// Since a Supervisor is a Task, any supervisor may be submitted to another
// supervisor!  Composing trees of supervision like this is a great way to
// architect reliable programs.
//
// Like most other Task implementations, most of the work a supervisor should
// be doing is bound at construction time.  For supervisors, usually means
// either a slice []Task or TaskGen channel is a parameter when creating the
// supervisor.
//
// Supervisors can be cancelled just like any other Task -- through Context.
// Cancellation of one supervisor will automatically fan out to all children
// (including, of course, recursively through other supervisors).
type Supervisor interface {
	NamedTask     // All supervisors are themselves tasks that can be submitted to another supervisor.
	Phase() Phase // Return the current phase the supervisor is in (advisory/monitoring only).
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

// SuperviseStream creates a Supervisor which will launch and handle
// a goroutine for each of the tasks supplied by the given TaskGen channel.
// When run, the supervisor will not return until the TaskGen channel is closed
// or the Run context is cancelled.
func SuperviseStream(
	taskGroupName string,
	taskSrc TaskGen,
	opts ...SupervisionOptions,
) Supervisor {
	return superviseStream{name: taskGroupName}.init(taskSrc)
}

// Placeholder.
//
// ex:
//   - goroutineBucketSize(10)
//   - convertPanics(false)
//   - logRunaways(os.Stderr, 2*time.Second)
type SupervisionOptions func()
