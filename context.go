package sup

import (
	. "context"
)

type ctxKey struct{}

type ctxInfo struct {
	task *boundTask
	path string
}

func appendCtxInfo(ctx Context, x ctxInfo) Context {
	return WithValue(ctx, ctxKey{}, x)
}

// CtxTaskName returns the name of the current task
// (or if there is no task annotated as owner of this context,
// returns the empty string).
//
// Task name and path info is annotated when tasks are launched by supervisors,
// and may be missing if you call a task's Run method manually.
func CtxTaskName(ctx Context) string {
	ctxInfo, ok := ctx.Value(ctxKey{}).(ctxInfo)
	if !ok {
		return ""
	}
	return ctxInfo.task.name
}

// CtxTaskPath returns the full path of names for each task in the supervision
// tree above this one
// (or if there is no task annotated as owner of this context,
// returns the empty string).
//
// The path is separated by slashes, as per file paths.
//
// Task name and path info is annotated when tasks are launched by supervisors,
// and may be missing if you call a task's Run method manually.
func CtxTaskPath(ctx Context) string {
	ctxInfo, ok := ctx.Value(ctxKey{}).(ctxInfo)
	if !ok {
		return ""
	}
	return ctxInfo.path
}
