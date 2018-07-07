package sup

import (
	"context"
)

// Task is an interface with one function: Run.  It's for running things.
//
// All concurrent functions take a context.Context -- this is is Go standard,
// and necessary for graceful concurrent halting, as well as a carrier for
// metadata like task tree name -- and may return an error.
//
// See the other "*Task" interfaces for more expressive functions you can
// add to the same object in order to enable more go-sup features.
type Task interface {
	Run(context.Context) error
}

// NamedTask implementers can specify a custom name string that go-sup will
// attach to the context when launching the task and use in any go-sup logging.
//
// If this interface is not implemented by a Task, the default behavior is to
// generate a name based on an interal memory address of the Task (based on the
// assumption that this should be reasonably uniqueish in practice).
type NamedTask interface {
	Task
	Name() string
}
