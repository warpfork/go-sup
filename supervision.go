package sup

import (
	"sync"
	"sync/atomic"
)

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
	//
	// The name string provided should be unique, but if it is not,
	// some other similar name will be determined and used instead.
	// The resolution mechanism can be controlled by SetNameSelectionStrategy,
	// but the default strategy has two behaviors:
	// if the name has "%" characters, it replaces each of them with a random digit [0-9];
	// or in other collisions, it suffixes a "+1" to the name until it ceases to collide.
	//
	// Submit can be called at any point in time, and will never block.
	// However, calling it very early in the Supervisor's lifecycle,
	// and calling it during or after the Supervisor's shutdown, has some different results.
	// If called before the Supervisor is started, note that running the SupervisedTask
	// returned *will* block until the Supervisor is started (it will refuse to run
	// until there's someone to report errors to!).
	// If Submit is called after the Supervisor is beginning to wind down,
	// a SupervisedTask is still returned, but it is considered rejected and lame:
	// the Context it receives when it is run will already be cancelled
	// (meaning the correct behavior for such a lame task would be to return
	// immediately without attempting any action);
	// and an errors returned from a such a lame task are not guaranteed to
	// be reported by the Supervisor's own final error return.
	Submit(name string, t Task) SupervisedTask

	// REVIEW: instead of documenting "correct" behavior for lame tasks, consider returning a dummy SupervisedTask that refuses to invoke the real logic at all.  Much safer.

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

	// SetNameSelectionStrategy allows configuring how names for SupervisedTask
	// are chosen when they are submitted.
	// It is not usually necessary to set a behavior other than the default.
	//
	// The default strategy is to respect the requested name,
	// except where it contains "%" characters, replace each of them with a random digit [0-9]
	// (this is meant to be a convenience system for task pools),
	// and in case of collisions, either try again (if "%" characters were involved),
	// or suffix a "+1" string literal (if "%" characters were not involved).
	//
	// The "requested" parameter sent to the callback is the name as requested
	// by the call to Submit.  The "attempted" parameter is often the empty string,
	// but in the case that a collision has occured, contains the previously attempted string
	// (which may already differ from the originally requested string, depending
	// on the behavior of your callback).  The "attempts" parameter increments
	// each time the supervisor has to ask again to find a non-colliding name
	// (it is zero the first time, one after a collision, and so on),
	// which can be used to notice excessive collisions and change tactics.
	SetNameSelectionStrategy(func(requested, attempted string, attempts int) (proposed string))

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
	// Name returns the fully-qualifed name of the SupervisedTask.
	// This name is created by the Supervisor when a Task is submitted to it,
	// and contains the name of the Supervisor (and implicitly, all of the name
	// of its parents as well), as well as the name suggested when the Task was
	// submitted (though this may have been modified by the Supervisor's task
	// name selection strategy).
	//
	// FIXME big whoops.  can't do the FQ part until the supervisor got ran.  what do?  move Context param to supervisor constructor after all?  but can't remove it from Run so uh??  i guess we just document that the first one is for logging and the second one is for running and we panic if they aren't the same one.
	Name() string

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
	TaskState_Initial                TaskState = iota // Unpowered itself and its supervisor is also unpowered.  Both must change before work will happen.
	TaskState_SupervisedButUnpowered                  // The task's Supervisor is running, but Run on this SupervisedTask has not be called.
	TaskState_BlockedUntilSupervised                  // Run on this SupervisedTask has been called, but the Supervisor hasn't been Run yet, so we have a thread ready to go to work, but we've parked it until the Supervisor comes up.
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

func NewSupervisor(ctx Context) Supervisor {
	return &supervisor{
		ctx:                   ctx,
		parent:                ContextSupervisor(ctx),
		nameSelectionStrategy: NameSelectionStrategy.Default,
		returnOnEmpty:         true,

		phase:         SupervisorPhase_NotStarted,
		knownTasks:    make(map[string]SupervisedTask),
		reservedNames: make(map[string]struct{}),
	}
}

// supervisor is the only real implementation of the Supervisor interface.
// We used an interface for the public API as a "just in case",
// but for now, any variations we're aware of are doable with configuration rather than whole polymorphism.
type supervisor struct {
	mu sync.Mutex // way too much going on here to keep it straight any other way.

	// config:
	ctx                   Context
	parent                Supervisor
	nameSelectionStrategy func(requested, attempted string, attempts int) (proposed string)
	returnOnEmpty         bool

	// state:
	phase         SupervisorPhase
	knownTasks    map[string]SupervisedTask
	reservedNames map[string]struct{}
}

type SupervisorPhase = uint32

// Implementation note: it may look somewhat odd that `SupervisorPhase` is an alias rather than a normal type declaration,
// and it is, but the reason for this is otherwise `atomic.CompareAndSwapUint32` won't let you touch it,
// and it produces an unreasonable amount of syntactic hullaballoo.
// It would be nice if perhaps `atomic.CompareAndSwapUint32` used a constraint like `~uint32`, now that golang supports that.

const (
	_ SupervisorPhase = iota
	SupervisorPhase_NotStarted
	SupervisorPhase_Running
	SupervisorPhase_WindingDown
	SupervisorPhase_Aborting
	SupervisorPhase_Halted
)

func (s *supervisor) Submit(name string, t Task) SupervisedTask {
	panic("todo")

	// TODO wrap it in panic gathering?
	//  I guess by default, yes, and opting out of that is yet another configurable property of the supervisor.

	// TODO also peek for if the task is another supervisor.  save a tree of these.
	//  It's not strictly necessary for the supervision/waiting/error-gathering jobs,
	//  but it enables some neat stuff like being able to ask for a report about the whole tree of tasks and their statuses.
}

func (s *supervisor) Run(Context) error {
	// Enforce single-run.
	ok := atomic.CompareAndSwapUint32(&s.phase, SupervisorPhase_NotStarted, SupervisorPhase_Running)
	if !ok {
		panic("supervisor can only be Run() once!")
	}
	panic("todo")
}

func (s *supervisor) Parent() Supervisor {
	panic("todo")
}

func (s *supervisor) QuitAggressively() {
	panic("todo")
}

func (s *supervisor) SetReturnOnEmpty(b bool) {
	s.mu.Lock()
	s.returnOnEmpty = b
	// todo signal a core loop tick if this was a transition to true.
	s.mu.Unlock()
}

func (s *supervisor) SetNameSelectionStrategy(nss func(string, string, int) string) {
	s.mu.Lock()
	s.nameSelectionStrategy = nss
	s.mu.Unlock()
}

func (s *supervisor) SetErrorReactor(func(error) SupervisionReaction) {
	panic("todo")
}

func (s *supervisor) SetWarningHandler(func(SupervisionWarning) error) {
	panic("todo")
}

type supervisedTask struct {
	task   Task
	parent Supervisor
	state  TaskState
}
