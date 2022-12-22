package sup

// Task is an interface you implement in order to make some function supervisable.
// Alternatively, can also make any `func(Context) error` into a Task with the TaskOfFunc constructor,
// or make a task that does some features in a loop by implementing SteppedTask.
//
// The Run method of every task only speaks in terms of two types (and is not generic):
// Context as an input (for cancellability, and also to access contextual information);
// and error as an output (for obvious purposes).
// If a task produces other results, this can be handled by exposing it as a field
// or accessor method, and this is not described by the Task interface.
// (Use the Await method on the SupervisedTask obtained by submitting a Task to a Supervisor
// to control access to such a result value: Await, and access the value afterwards.)
//
// Task is not meant to be run directly; its primary use is to be given to the
// Supervisor.Submit method, which returns a SupervisedTask, which is then more appropriate to run.
// The process of submitting the Task to a Supervisor and calling it via SupervisedTask
// is what allows the go-sup library to provide most of its features around monitoring and safety.
type Task interface {
	Run(Context) error
}

// SteppedTask is a convenient alternative to Task which calls the RunStep method in a loop
// as long as the Context has not been cancelled.
// It's just here to save you about a dozen lines of very common boilerplate.
type SteppedTask interface {
	RunStep(Context) error
}

// FUTURE: We may add some features like `NewActor` (name tbd) that takes your runnable and wraps it in RunOnce sanity checkers.
// Some runnables are written to be reusable; others are not; and we can't really usefully make those differentiable that in the API.
// The next best thing we can do is offer a standard way to declare that, and deploy cost-effective runtime sanity checks to ensure incorrect usage can't go unnoticed.
// The value of this _will_ remain best-effort: if someone retains a pointer to the thing _before_ it was wrapped, we can't do anything to stop that.

// type RebootableTask interface { Task; Reset() error; } // ?

func TaskOfFunc(fn func(Context) error) Task {
	return simpleTask{fn}
}
func TaskOfSteppedTask(t SteppedTask) Task {
	return steppedTask{t}
}

type simpleTask struct {
	fn func(Context) error
}

func (t simpleTask) Run(ctx Context) error { return t.fn(ctx) }

// REVIEW: this is pretty indirection-heavy and also TaskOfSteppedTask forces syntax burden on the end-user.
// Maybe this should be accomplished by offering a helper function that one uses inside one's Run func, instead.
// We don't gain much that's special by implementing this control loop out here in a known place.
//
// We could have the body be recommended to be something like:
//   `func (t *myTask) Run(Context) error { return sup.RunSteps(t.RunStep) }`.
//  That does basically the right things and is very non-magical,
//  and while it shifts *some* syntax burden onto the user, it's in the definition area, not in the registration area, which seems better.
//  Grabbing a reference to another method of the same value like that also does not provoke enclosure allocations, so is roughly free in performance terms.
//
// Or on the gripping hand: we could have more methods as markers, like `func (myTask) SteppedTask()` which have no purpose except to change how it's handled.
//  I'm not sure this is maximally golang-idiomatic.  And it has the unfortunate effect that it can purpose relatively spooky-action-at-a-distance, while the main run method name is always the same, which is a bit confusing if you're not expecting it.
//   Yeah, let's not do this.  I'd hate this when reading code review.

type steppedTask struct {
	t SteppedTask
}

func (t steppedTask) Run(ctx Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := t.t.RunStep(ctx); err != nil {
				return err
			}
		}
	}
}
