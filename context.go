package sup

import "context"

// Context is an alias permitting you to refer to sup.Context if you so desire.
type Context = context.Context

// ctxKey is a magic type used as a unique key for ctx.Value attachments.
//
// We have exactly one such key and store all further information in a struct underneath it.
// This approach is for sympathy to the internals of the context value attachment system --
// it performs an allocation and forms roughly a sort of long linked list for each
// additional attachment, so we reduce overhead by putting all value attachments
// we know about into a single attachment.
type ctxKey = struct{}

func ReadContext(ctx Context) CtxAttachments {
	v := ctx.Value(ctxKey{})
	if v == nil {
		return CtxAttachments{
			TaskNameShort: "[unmanaged]",
			TaskNameFull:  "[unmanaged]",
		}
	}
	return v.(CtxAttachments)
}

type CtxAttachments struct {
	Supervisor    Supervisor
	Task          SupervisedTask
	TaskNameShort string
	TaskNameFull  string
}

// ContextSupervisor returns a pointer to the Supervisor
// that's the nearest parent in this context's tree,
// or nil if there is no supervisor.
func ContextSupervisor(ctx Context) Supervisor {
	return ReadContext(ctx).Supervisor
}

// ContextTask returns a pointer to the SupervisedTask
// that's the nearest parent in this context's tree.
func ContextTask(ctx Context) SupervisedTask {
	return ReadContext(ctx).Task
}

// ContextName is a shortcut for `ContextTask(ctx).Name()`,
// or returns a placeholder string if there is no Task in this context.
func ContextName(ctx Context) string {
	return ReadContext(ctx).TaskNameFull
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
