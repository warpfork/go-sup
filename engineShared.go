package sup

import (
	"context"
	"fmt"
	"path/filepath"
)

type Phase uint32

const (
	Phase_uninitalized = Phase(0) // panic if you see this.
	Phase_init         = Phase(1) // when the mgr is properly constructed.
	Phase_running      = Phase(2) // immediately after the manager task has been Run(), and new tasks can still be submitted.
	Phase_collecting   = Phase(3) // when the manager is running, but no new tasks can be submitted (n.b. this replaces Phase_running completely for forkjoin).
	Phase_halting      = Phase(4) // when waiting for all children to return (we've either been cancelled by parent or child has errored).
	Phase_halt         = Phase(5) // all tasks have returned, we're done here and you can have the final result.
)

type phaseFn func(parentCtx context.Context) phaseFn

type reportMsg struct {
	task   *boundTask
	result *ErrChild
}

// ErrChild wraps any errors returned or panicked from a Task when they're
// yielded up a supervision tree.
//
// The original error can be extracted from the `Err` field.
//
// Some additional metadata is available from the other fields.
type ErrChild struct {
	Err      error
	WasPanic bool
}

func (e ErrChild) Error() string {
	return e.Err.Error()
}

// childLaunch is the first function on a child goroutine's stack.
// It handles context tree extension, defer capturing, etc.
func childLaunch(groupCtx context.Context, report chan<- reportMsg, task *boundTask) {
	var childErr error // The child's *returned* error is stored here.
	defer func() {
		report <- reportMsg{task, siftError(childErr, recover())}
	}()
	taskPath := filepath.Join(CtxTaskPath(groupCtx), task.name)
	ctx := appendCtxInfo(groupCtx, ctxInfo{task, taskPath})
	childErr = task.original.Run(ctx)
}

func siftError(retErr error, rcvr interface{}) *ErrChild {
	if rcvr != nil {
		if err, ok := rcvr.(error); ok {
			return &ErrChild{err, true}
		}
		return &ErrChild{fmt.Errorf("%v", rcvr), true}
	}
	if retErr == nil {
		return nil
	}
	if e2, ok := retErr.(*ErrChild); ok {
		return e2
	}
	return &ErrChild{retErr, false}
}
