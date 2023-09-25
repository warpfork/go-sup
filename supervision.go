package sup

import (
	"context"
	"fmt"
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

	// Phase peeks at the current phase of the task.
	// This is an atomically loaded view, but may instantly be out of date,
	// and so is really only useful for inspection and monitoring purposes.
	//
	// Use Promise if you want to wait for the task's phase to become TaskPhase_Done.
	Phase() TaskPhase

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

type TaskPhase = uint32

// Implementation note: it may look somewhat odd that `TaskPhase` is an alias rather than a normal type declaration,
// and it is, but ... see similar note on SupervisorPhase.

const (
	TaskPhase_Initial                TaskPhase = iota // Unpowered itself and its supervisor is also unpowered.  Both must change before work will happen.
	TaskPhase_SupervisedButUnpowered                  // The task's Supervisor is running, but Run on this SupervisedTask has not be called.
	TaskPhase_BlockedUntilSupervised                  // Run on this SupervisedTask has been called, but the Supervisor hasn't been Run yet, so we have a thread ready to go to work, but we've parked it until the Supervisor comes up.
	TaskPhase_Running                                 // `Do` has been called; we are supervised; work is in progress; it hasn't halted or been cancelled yet.
	TaskPhase_Cancelling                              // The task state was previously Running, but we've now been cancelled, and we're waiting on the task to wrap up before transitioning to Done.
	TaskPhase_Done                                    // Running is done.
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

// NewRootSupervisor creates a new Supervisor with no parent
// and the name "root".
//
// Use this the first time you create a supervisor in your program.
//
// (Currently, there is no enforcement that you do this only one time,
// but task names may become confusing and collide if this is used more than once.)
func NewRootSupervisor(ctx Context) Supervisor {
	ctx2, cancelFn := context.WithCancel(ctx)
	return &supervisor{
		name:                  "root", // TODO make this a parameter, and use a package-global map to force uniqueness.
		nameFQ:                "root",
		ctxSelf:               ctx,
		ctxChildren:           ctx2,
		cancelChildren:        cancelFn,
		parent:                nil,
		nameSelectionStrategy: NameSelectionStrategy.Default,
		returnOnEmpty:         true,
		errReactor:            func(error) SupervisionReaction { return SupervisionReaction_Error },

		phase:      SupervisorPhase_NotStarted,
		knownTasks: make(map[string]*supervisedTask),

		childCompletion: make(chan *supervisedTask, 1),
	}
}

// NewSupervisor creates a new, not-yet-launched Supervisor,
// which inherents its name from the current context's task
// (e.g. `ContextName(ctx)`), and is connected to the parent Supervisor of that task.
//
// Use this in preference to NewRootSupervisor unless it's the first one in the program.
func NewSupervisor(ctx Context) Supervisor {
	ctxInfo := ReadContext(ctx)
	ctx2, cancelFn := context.WithCancel(ctx)
	return &supervisor{
		name:                  ctxInfo.TaskNameShort,
		nameFQ:                ctxInfo.TaskNameFull,
		ctxSelf:               ctx,
		ctxChildren:           ctx2,
		cancelChildren:        cancelFn,
		parent:                ctxInfo.Supervisor,
		nameSelectionStrategy: NameSelectionStrategy.Default,
		returnOnEmpty:         true,
		errReactor:            func(error) SupervisionReaction { return SupervisionReaction_Error },

		phase:      SupervisorPhase_NotStarted,
		knownTasks: make(map[string]*supervisedTask),

		childCompletion: make(chan *supervisedTask, 1),
	}
}

// supervisor is the only real implementation of the Supervisor interface.
// We used an interface for the public API as a "just in case",
// but for now, any variations we're aware of are doable with configuration rather than whole polymorphism.
type supervisor struct {
	// One mutex guards most operations relating to knownTasks.
	// The Submit function grabs it when used, and the Run function grabs it cyclically while the supervisor is running.
	// Because Submit can be called before Run, we need a mutex, rather than a submission channel handled actor-style, which might've otherwise been preferable.
	// Since we've got it, then, we alsouse it to guard changes to and reads from all the other config fields in a supervisor.
	// It's not typical to change most of the config fields during the run, but we face no significant additional cost by supporting it, so we might as well.
	mu sync.Mutex

	// config:
	name                  string
	nameFQ                string
	ctxSelf               Context
	ctxChildren           Context // all child contexts fork from this (thus share cancellation)
	cancelChildren        func()
	parent                Supervisor
	nameSelectionStrategy func(requested, attempted string, attempts int) (proposed string)
	returnOnEmpty         bool
	errReactor            func(error) SupervisionReaction

	// state:
	phase      SupervisorPhase
	knownTasks map[string]*supervisedTask

	// wiring:
	childCompletion chan *supervisedTask // children send themselves here when done.
}

type SupervisorPhase = uint32

// Implementation note: it may look somewhat odd that `SupervisorPhase` is an alias rather than a normal type declaration,
// and it is, but the reason for this is otherwise `atomic.CompareAndSwapUint32` won't let you touch it,
// and it produces an unreasonable amount of syntactic hullaballoo.
// It would be nice if perhaps `atomic.CompareAndSwapUint32` used a constraint like `~uint32`, now that golang supports that.

const (
	_                           SupervisorPhase = iota
	SupervisorPhase_NotStarted                  // The supervisor hasn't started yet.  Task submission is acceptable.
	SupervisorPhase_Running                     // The supervisor is running, watching existing tasks, and accepting new submissions.
	SupervisorPhase_WindingDown                 // This supervisor is waiting for existing tasks, but not accepting new submissions.
	SupervisorPhase_Aborted                     // The supervisor did a hard abort; children have been cancelled and their results not collected.  New submissions are not acceptable.
	SupervisorPhase_Halted                      // The supervisor completed a graceful winding down: all child tasks were gathered.  New submissions are acceptable.  Children may have errored.
)

func (s *supervisor) Submit(name string, t Task) SupervisedTask {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO if we're already WindingDown, Aborted, or Halted: return a dummy task.
	//  ...probably do still have to name it?  Or can we give it a single reused dummy name?  Tbd.
	// Perhaps also have configurable rejection strategy.  Some might prefer a panic if they submit to a closed supervisor.

	// Pick a locally unique name.
	name = s._submit_selectName(name)

	// Create the supervisedTask struct, and record it.
	st := &supervisedTask{
		task:         t,
		name:         name,
		nameFQ:       s.nameFQ + "." + name,
		parent:       s,
		phase:        TaskPhase_Initial,
		clearToStart: make(chan struct{}), // TODO I think we can avoid this alloc in the case the supervisor is already running.
	}
	st.promise, st.resolveFn = NewPromise[SupervisedTask]()
	s.knownTasks[name] = st // TODO don't do this if the supervisor is rejecting; it just adds more mutex needs and garbage collection problems.

	// Create the Context for this soon-to-be child.
	// Each supervised task gets a new context value, with attachments describing it,
	// and decended from the context this superviser users to cancel all children.
	st.ctx = context.WithValue(s.ctxChildren, ctxKey{}, CtxAttachments{
		Supervisor:    s,
		Task:          st,
		TaskNameShort: st.name,
		TaskNameFull:  st.nameFQ,
	})

	// If this supervisor is already running: we can let it launch right away.
	switch s.phase {
	case SupervisorPhase_Running:
		close(st.clearToStart)
		// No atomics needed here; no one else can see this memory yet.
		st.phase = TaskPhase_SupervisedButUnpowered
	}

	// TODO wrap it in panic gathering?
	//  I guess by default, yes, and opting out of that is yet another configurable property of the supervisor.

	// TODO also peek for if the task is another supervisor.  save a tree of these.
	//  It's not strictly necessary for the supervision/waiting/error-gathering jobs,
	//  but it enables some neat stuff like being able to ask for a report about the whole tree of tasks and their statuses.

	return st
}

func (s *supervisor) _submit_selectName(requested string) string {
	nameAttempts := 0
	actualName := requested
	for {
		actualName = s.nameSelectionStrategy(requested, actualName, nameAttempts)
		if _, exists := s.knownTasks[actualName]; !exists {
			return actualName
		}
		nameAttempts++
		if nameAttempts > 100 {
			panic("your name selection strategy is broken") // TODO do some other fallback.  Maybe just add more numbers than the nss asked for.
		}
	}
}

// Run matches the standard `Task` interface.
// The running of supervision logic is just another task, after all!
//
// Note that the Context argument must exactly match the one given
// to NewSupervisor at construction time.
// (We need a Context at NewSupervisor time because we use it to derive task names,
// and we do those promptly during submission... but we also need a Context object
// here, purely to satisfy the Task interface.  This is unfortunate.)
func (s *supervisor) Run(ctx Context) (err error) {
	if s.ctxSelf != ctx {
		panic("supervisor.Run must be given the same Context used to construct it!")
	}

	phase := s._run_start()

	// Loop, servicing the childCompletion channel, until either:
	//  - knownTasks is empty, and returnOnEmpty is true at the same time;
	//  - one of those child completions carries an error that the error reactor didn't swallow;
	//  - or quitAggressively is called.
	for phase == SupervisorPhase_Running {
		select {
		// TODO case for quitAggressively
		// TODO case for transitioned to returnOnEmpty==true
		case child := <-s.childCompletion:
			phase, err = s._run_recvChild(child)
		}
	}

	// Fan out cancellations.
	//  (This may be functionally a no-op if we're shutting down gracefully from a lack of tasks,
	//   but the context system obscures that from us to a high degree.)
	s.cancelChildren()

	// If we're in quitAggressively/abort mode: that's it.  Get outta here, without waiting.
	if phase == SupervisorPhase_Aborted {
		return
	}

	// Wait for all remaining children to roll up.  (This is a no-op if we're shutting down gracefully from a lack of tasks.)
	// Or, also still be ready to bug out hard and fast if quitAggressively is called.
	// Keep a different error value here because it's not dominant.
	var err2 error
	for phase == SupervisorPhase_WindingDown {
		select {
		// TODO case for quitAggressively
		case child := <-s.childCompletion:
			phase, err2 = s._winddown_recvChild(child)
		}
	}
	if err == nil {
		err = err2
	}
	return
}

func (s *supervisor) _run_start() SupervisorPhase {
	// Do the phase transition and the launch of tasks submitted earlier under one contiguous mutex hold,
	//  because we're changing the rule for whether knownTasks contents are expected to be launched.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce single-run.
	// (Atomics on the phase are somewhat overkill, since we hold the lock anyway, but keeps any peeking loads well-defined.)
	ok := atomic.CompareAndSwapUint32(&s.phase, SupervisorPhase_NotStarted, SupervisorPhase_Running)
	if !ok {
		panic("supervisor can only be Run() once!")
	}

	// Finish setting up and unblock all SupervisedTask that were registered before we launched.
	for _, child := range s.knownTasks {
		close(child.clearToStart)
	}

	// Corner case: if there were actually no tasks, and returnOnEmpty==true... we kinda never really need to do anything again.
	if s.returnOnEmpty && len(s.knownTasks) == 0 {
		atomic.StoreUint32(&s.phase, SupervisorPhase_Halted)
		return SupervisorPhase_Halted
	}
	return SupervisorPhase_Running
}

// Handle the child's exit.  Remove it from tracking;
// decide how to handle the error, if there is one;
// and if we change phase, both store that and return it.
func (s *supervisor) _run_recvChild(child *supervisedTask) (SupervisorPhase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove it from the set of things we continue to need to track.
	delete(s.knownTasks, child.name)

	// If error is nil, we might quietly continue, or be done.
	if child.err == nil {
		if s.returnOnEmpty && len(s.knownTasks) == 0 {
			atomic.StoreUint32(&s.phase, SupervisorPhase_Halted)
			return SupervisorPhase_Halted, nil
		}
		return SupervisorPhase_Running, nil
	}

	// If error was non-nil, use the reactor callback to decide what happens next.
	switch s.errReactor(child.err) {
	case SupervisionReaction_Error:
		if len(s.knownTasks) == 0 {
			atomic.StoreUint32(&s.phase, SupervisorPhase_Halted)
			return SupervisorPhase_Halted, child.err // TODO probably wrap
		}
		atomic.StoreUint32(&s.phase, SupervisorPhase_WindingDown)
		return SupervisorPhase_WindingDown, child.err // TODO probably wrap
	case SupervisionReaction_Ignore:
		if s.returnOnEmpty && len(s.knownTasks) == 0 {
			atomic.StoreUint32(&s.phase, SupervisorPhase_WindingDown)
			return SupervisorPhase_WindingDown, nil
		}
		return SupervisorPhase_Running, nil
	case SupervisionReaction_AbortRapidly:
		atomic.StoreUint32(&s.phase, SupervisorPhase_Aborted)
		return SupervisorPhase_Aborted, child.err // TODO probably wrap
	default:
		panic("invalid SupervisionReaction enum returned by error reactor func")
	}
}

// Very similar to _run_recvChild, but slightly different constants,
// and we gave it a different name in for stack trace legibility purposes.
func (s *supervisor) _winddown_recvChild(child *supervisedTask) (SupervisorPhase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove it from the set of things we continue to need to track.
	delete(s.knownTasks, child.name)

	// If error is nil, we might quietly continue, or be done.
	if child.err == nil {
		if len(s.knownTasks) == 0 {
			atomic.StoreUint32(&s.phase, SupervisorPhase_Halted)
			return SupervisorPhase_Halted, nil
		}
		return SupervisorPhase_WindingDown, nil
	}

	// If error was non-nil, use the reactor callback to decide what happens next.
	switch s.errReactor(child.err) {
	case SupervisionReaction_Error:
		if len(s.knownTasks) == 0 {
			atomic.StoreUint32(&s.phase, SupervisorPhase_Halted)
			return SupervisorPhase_Halted, child.err // TODO probably wrap
		}
		return SupervisorPhase_WindingDown, child.err // TODO probably wrap
	case SupervisionReaction_Ignore:
		if len(s.knownTasks) == 0 {
			atomic.StoreUint32(&s.phase, SupervisorPhase_Halted)
			return SupervisorPhase_Halted, nil
		}
		return SupervisorPhase_WindingDown, nil
	case SupervisionReaction_AbortRapidly:
		atomic.StoreUint32(&s.phase, SupervisorPhase_Aborted)
		return SupervisorPhase_Aborted, child.err // TODO probably wrap
	default:
		panic("invalid SupervisionReaction enum returned by error reactor func")
	}
}

func (s *supervisor) Parent() Supervisor {
	return s.parent
}

func (s *supervisor) QuitAggressively() {
	panic("todo")
}

func (s *supervisor) SetReturnOnEmpty(b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase > SupervisorPhase_Running {
		panic("nonsensical to change winddown triggers on a supervisor that's already past running")
	}
	s.returnOnEmpty = b
	// TODO signal a core loop tick if this was a transition to true.
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
	task         Task
	name         string      // set by parent during Submit
	nameFQ       string      // set by parent during Submit
	parent       *supervisor // set by parent during Submit
	ctx          Context     // set by parent during Submit
	phase        TaskPhase
	clearToStart chan struct{} // closed by the parent supervisor when it has started, signalling it's ready to receive any errors and that this task can thus start.
	err          error         // stored at end of Run; parent can pluck it back out.
	promise      Promise[SupervisedTask]
	resolveFn    func(SupervisedTask)
}

func (t *supervisedTask) Name() string {
	return t.nameFQ
}

func (t *supervisedTask) Task() Task {
	return t.task
}

func (t *supervisedTask) Parent() Supervisor {
	return t.parent
}

func (t *supervisedTask) Phase() TaskPhase {
	return atomic.LoadUint32(&t.phase)
}

func (t *supervisedTask) Promise() Promise[SupervisedTask] {
	return t.promise
}

func (t *supervisedTask) Run() error {
	// Each phase is factored out so they show up obviously on any stack traces.
	// Note that these don't correspond exactly to the TaskPhase codes that are exported.
	// The await supervision phase can cover several codes.
	// TaskPhase_Cancelling is somewhat ellusive; go-sup helpers (like the channel guards) can set it, but if it's the user's code that picks it up, well.
	// And we give notification a phase here again just for labelling purposes.  It "should" be instant.  But... just in case: let's have it be visible in the stack trace.
	t._phase_awaitSupervision()
	defer t._phase_notify()
	defer t._panicCollector()
	t._phase_run()
	return t.err
}

func (t *supervisedTask) _phase_awaitSupervision() {
	updated := false
	for !updated { // Improbable that this loops at all, and certainly not more than once, but this may be racing a transition from Initial to SupervisedButUnpowered state in another goroutine.
		prev := atomic.LoadUint32(&t.phase)
		switch prev {
		case TaskPhase_Initial, TaskPhase_SupervisedButUnpowered:
			select {
			case <-t.clearToStart:
				updated = atomic.CompareAndSwapUint32(&t.phase, prev, TaskPhase_Running)
				if updated {
					return
				}
			default:
				updated = atomic.CompareAndSwapUint32(&t.phase, prev, TaskPhase_BlockedUntilSupervised)
			}
		default:
			panic("supervisedTask cannot be Run more than once!")
		}
	}
	<-t.clearToStart
	atomic.StoreUint32(&t.phase, TaskPhase_Running)
}

func (t *supervisedTask) _phase_run() {
	t.err = t.task.Run(t.ctx)
}

func (t *supervisedTask) _panicCollector() {
	if err := recover(); err != nil {
		err2, ok := err.(error)
		if !ok {
			err2 = fmt.Errorf("non-error value panicked: %s", err)
		}
		t.err = fmt.Errorf("panic collected: %w", err2) // FIXME replace this with more typed and meaningful errors.  the error handler should be able to see it's a recovered panic.
	}
}

func (t *supervisedTask) _phase_notify() {
	t.parent.childCompletion <- t // FIXME this needs to not happen if the parent is aborting; nobody's listening and we shouldn't block.
	atomic.StoreUint32(&t.phase, TaskPhase_Done)
	t.resolveFn(t)
}
