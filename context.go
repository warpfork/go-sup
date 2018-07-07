package sup

import (
	. "context"
)

type ctxKey struct{}

type ctxInfo struct {
	task *boundTask
}

func appendCtxInfo(ctx Context, x ctxInfo) Context {
	return WithValue(ctx, ctxKey{}, x)
}

// CtxTaskName returns the name of the current task
// (or if there is no task annotated as owner of this context,
// returns the empty string).
func CtxTaskName(ctx Context) string {
	ctxInfo, ok := ctx.Value(ctxKey{}).(ctxInfo)
	if !ok {
		return ""
	}
	return ctxInfo.task.name
}
