package sup

import "context"

type Context = context.Context

type (
	ctxKey_supervisor = struct{}
	ctxKey_task       = struct{}
)

// ContextSupervisor returns a pointer to the Supervisor
// that's the nearest parent in this context's tree.
func ContextSupervisor(ctx Context) Supervisor {
	return ctx.Value(ctxKey_supervisor{}).(Supervisor)
}

// ContextTask returns a pointer to the SupervisedTask
// that's the nearest parent in this context's tree.
func ContextTask(ctx Context) SupervisedTask {
	return ctx.Value(ctxKey_task{}).(SupervisedTask)
}

// ContextName is a shortcut for `ContextTask(ctx).Name()`,
// or returns a placeholder string if there is no Task in this context.
func ContextName(ctx Context) string {
	task := ContextTask(ctx)
	if task == nil {
		return "#noTask#"
	}
	return task.Name()
}

// Considered introducing a second quit/cancel channel that would mean "quit harder"
// and be honored even when a cancel wasn't, for whatever reason.
// E.g., the first cancel should cause cleanup and graceful shutdown; the second should cause rapid abort.
//
// This is a nice idea but might be impractical to implement.
// It would be difficult to make it widely honored.
// (The usual cancel would almost certainly have to be treated as the aggressive one.)
// And if one *really, really* means it about aggressive rapid abort,
// then the most brutal challenge probably isn't communicating it:
// it's is going to be finding some way to not get stuck in IO operations that lack good interruptability...
// for which we simply have no way to wage that war other than full OS-level-process shutdown;
// and that in turn is served reasonably well by simply giving up waiting for children in all supervision
// and hastening for the exit.  (Or, you know, going straight and aggressively to `os.Exit`.)
//
// So.  Nice idea.  But seems low-practicality, and thus low-utility to chase,
// when acknowledging the practical constraints of kernels and the real inability to interrupt any tasks that are uncooperative.
