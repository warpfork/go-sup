package sup

import (
	"context"
	"path/filepath"
)

type superviseRoot struct {
	// no need for the whole phase machine on this one; we never return a
	//  public handle to any part of this implementation.

	task *boundTask
}

func (superviseRoot) _Supervisor() {}

func (mgr superviseRoot) init(task Supervisor) Supervisor {
	mgr.task = bindTask(task)
	return &mgr
}

func (mgr superviseRoot) Name() string {
	return "-"
}

func (mgr *superviseRoot) Run(parentCtx context.Context) error {
	return mgr.childLaunch(parentCtx, mgr.task)
}

func (mgr superviseRoot) childLaunch(groupCtx context.Context, task *boundTask) (report error) {
	var childErr error
	defer func() {
		report = childErr
		// TODO panic recovery
		// also TODO this child launcher isn't *exactly* duped yet but it's close, refactor
	}()
	taskPath := filepath.Join(CtxTaskPath(groupCtx), task.name)
	ctx := appendCtxInfo(groupCtx, ctxInfo{task, taskPath})
	childErr = task.original.Run(ctx)
	return
}
