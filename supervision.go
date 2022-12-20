package sup

func NewSupervisor() Supervisor {
	panic("todo")
	//return &supervisor{}
}

type Supervisor interface {
	// Supervisor is itself a Task -- it has a Run(Context) method.
	//
	// The Supervisor's Run will return only when all SupervisedTask submitted to it have returned,
	// or when a rapid exit is demanded (via a warning being upgraded to an error by the WarningHandler,
	// or via QuitAggressively).
	//
	// None of the SupervisedTask submitted to this Supervisor will run until this is run.
	Task

	// Submit enrolls a Task with this Supervisor and produces a SupervisedTask.
	// SupervisedTask is then ready to be run, and its Run method can be called from any goroutine.
	//
	// (Note that SupervisedTask.Run requires no Context parameter:
	// this is because the Context will be supplied by this Supervisor.)
	Submit(Task) SupervisedTask

	// QuitAggressively tells the Supervisor to cancel all children, refuse new submissions,
	// and return from its Run method as rapidly as possible, even if that involves ignoring
	// and abandoning children.
	//
	// Use of this is not recommended unless the program is on its way to exiting entirely,
	// as it may leave goroutines unsupervised, and this may lead to resource leaks
	// if the unsupervised goroutines do not honor the cancel signals.
	QuitAggressively()

	// SetReturnOnEmpty configures if the Supervisor's Run method will return as soon as it
	// has no SupervisedTask children which have not yet returned; by default, this is true.
	//
	// Setting return-on-empty to false lets you create a "task pool" pattern
	// which accepts submissions of more tasks at any time.
	// (Do this before calling Run, or a logical race condition is present!)
	// However, you must then remember to clean up:
	// either set return-on-empty back to true as part of shutting the program down,
	// or use cancellation on the Context that the Supervisor was Run with,
	// or your program will never exit!
	SetReturnOnEmpty(bool)

	// SetErrorReactor allows setting a callback that will define the supervisor's response
	// to an error being returned by one of the SupervisedTask that it oversees.
	//
	// By default, any errors from a supervised child are treated as toxic,
	// and cause the supervisor to send cancellations to all other children,
	// and return this error from its own Run method (as soon as all other children have returned).
	//
	// By customizing this behavior, you can ignore some errors,
	// or decide to abort rapidly (not waiting for other children to return in response to cancels!)
	// or even take other custom actions in response before letting the supervisor continue in its reaction.
	SetErrorReactor(func(error) SupervisionReaction)

	// "Warnings" encompases things like "a child has gone un-launched for more than a second"
	// or "we can't shut down and return because a child still hasn't been launched",
	// or similar situations that almost certainly indicate a programming error.
	// By default these are logged somewhat noisily.
	// By setting your own handler, you can decide how much information is logged (and to where),
	// and also have the option to return an error which will cause the supervisor to abort rapidly.
	SetWarningHandler(func(SupervisionWarning) error)

	// Parent returns the next Supervisor up from this one, if any.
	//
	// Parent is only valid to ask after the supervisor has been Run; it will return nil before that.
	// (Note it is generally valid to ask `ContextSupervisor(ctx).Parent()` at any time,
	// because implicitly, any supervisor, and thus any of its parents, must have been running
	// already for that Context to have been produced.)
	Parent() Supervisor
}

type SupervisedTask interface {
	Name() string // The name of a SupervisedTask is assigned when it's bound to a supervisor.  It includes the task's supervision parantage names, separated by dots.

	// Run launches the task, and blocks until the task is complete.
	// It performs all monitoring setup,
	// waits to make sure the task's Supervisor is also ready and launched,
	// and then proceeds into the Task.Run method of the actual task.
	// When the Task.Run feturns, SupervisedTask.Run also ensures all completion notifications are distributed.
	//
	// Typically, Run is called in a new goroutine as soon as the SupervisedTask is produced by Supervisor.Submit.
	// However, it can be called at any later time if you wish to perform some interesting scheduling control.
	Run() error

	// State peeks at the current state of the task.
	// This is an atomically loaded view, but may instantly be out of date, and so is really only useful for inspection and monitoring purposes.
	//
	// Use Promise if you want to wait for the state to become TaskState_Done.
	State() TaskState

	// Task returns a pointer to the raw Task that this SupervisedTask wraps.
	//
	// Note that undefined behavior will result if calling Task.Run directly;
	// the task is only meant to be run via SupervisedTask.Run.
	Task() Task

	// Parent returns the Supervisor that this task was submitted to.
	Parent() Supervisor

	// Promise can be used to await the completion of a SupervisedTask.
	// The returned Promise value will be resolved when this task becomes done.
	Promise() Promise[SupervisedTask]
}

type TaskState uint8

const (
	TaskState_Initial                TaskState = iota // unpowered and unsupervised
	TaskState_SupervisedButUnpowered                  // We've been enrolled in a supervisor, but no thread provided.
	TaskState_BlockedUntilSupervised                  // `Do` has been called, so we have a thread ready to go to work, but we've parked that thread until we get a supervisor.
	TaskState_Running                                 // `Do` has been called; we are supervised; work is in progress; it hasn't halted or been cancelled yet.
	TaskState_Cancelling                              // The task state was previously Running, but we've now been cancelled, and we're waiting on the task to wrap up before transitioning to Done.
	TaskState_Done                                    // Running is done.
)

type SupervisionReaction uint8

const (
	SupervisionReaction_Error  = iota // The natural thing to do.  If a child task errored, and we don't know what to do about it, the supervisor as a whole should start shutting down the other children and getting ready to error up.
	SupervisionReaction_Ignore        // Ignore the error.  The supervisor will continue running, not cancel any children, and not return.
	// ... we don't really have a "restart" operation available.  Maybe with more optional interfaces for tasks (e.g. "Init" or "Reinit" as well as just "Do"), we might.
	SupervisionReaction_AbortRapidly // Send cancels to other children (same as when SupervisionReaction_Error), then return _immediately_ (don't wait for other children to wrap up).
)

type SupervisionWarning struct {
	// TBD.
}

// supervisor is the only real implementation of the Supervisor interface.
// We used an interface for the public API as a "just in case",
// but for now, any variations we're aware of are doable with configuration rather than whole polymorphism.
type supervisor struct {
}
