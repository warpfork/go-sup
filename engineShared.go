package sup

import (
	"context"
	"path/filepath"
)

type phase uint32

const (
	phase_uninitalized = phase(0) // panic if you see this.
	phase_init         = phase(1) // when the mgr is properly constructed.
	phase_running      = phase(2) // immediately after the manager task has been Run(), and new tasks can still be submitted.
	phase_collecting   = phase(3) // when the manager is running, but no new tasks can be submitted (n.b. this replaces phase_running completely for forkjoin).
	phase_halting      = phase(4) // when waiting for all children to return (we've either been cancelled by parent or child has errored).
	phase_halt         = phase(5) // all tasks have returned, we're done here and you can have the final result.
)

type reportMsg struct {
	task   *boundTask
	result error
}

// childLaunch is the first function on a child goroutine's stack.
// It handles context tree extension, defer capturing, etc.
func childLaunch(groupCtx context.Context, report chan<- reportMsg, task *boundTask) {
	var childErr error
	defer func() {
		report <- reportMsg{task, childErr}
		// TODO panic recovery
	}()
	taskPath := filepath.Join(CtxTaskPath(groupCtx), task.name)
	ctx := appendCtxInfo(groupCtx, ctxInfo{task, taskPath})
	childErr = task.original.Run(ctx)
}
