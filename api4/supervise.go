package gracefully

import (
	"context"
)

type Context = context.Context

type Func func(Context) error

type Task interface {
	// Do will blockingly evaluate the bound Func.
	//
	// Context is provided from the supervisor, and no error is returned
	// because the error handling goes through the supervisor.
	//
	// Typical invocation may be in a block like this:
	//
	//		mgr := sup.BuildFiniteSupervisor()
	//		go mgr.Assign("foo", fooFn).Do()
	//		go mgr.Assign("bar", barCfg.Run).Do()
	//		return mgr.Engage(ctx)
	//
	// Or for open-ended streams of tasks:
	//
	//		enroll, mgr := sup.BuildStreamSupervisor()
	//		go func() {
	//			for taskFn := range <-taskQueue {
	//				go enroll.Assign("%", taskFn).Do()
	//			}
	//			enroll.Close()
	//		}()
	//		return mgr.Engage(ctx)
	//
	// Or to implement an open-ended stream, but executing serially:
	//
	//		tasks := sup.OpenAssignmentGroup()
	//		go func() {
	//			for taskFn := range <-taskQueue {
	//				// just leave off the 'go' keyword here ;)
	//				tasks.Assign("%", taskFn).Do()
	//			}
	//			tasks.Close()
	//		}()
	//		return sup.Supervise(tasks).Engage(ctx)
	//
	// This should compose nicely.  Here we have an example of
	// building and launching one supervisor with two children;
	// the first of which launches another supervisor with two
	// grandchildren (which are leaf nodes; it stops there);
	// and second of which also launches another supervisor with two
	// grandchildren... of which one is open-ended for an indefinite
	// number of great-grandchildren, while its sibling implements
	// submission control for it (here, a placeholder taskQueue is
	// used to gather submissions from elsewhere in the program):
	//
	//		mgr := sup.BuildFiniteSupervisor()
	//		go mgr.Assign("group1", func(ctx Context) error {
	//			mgr := sup.BuildFiniteSupervisor()
	//			go mgr.Assign("foo", fooFn).Do()
	//			go mgr.Assign("bar", barCfg.Run).Do()
	//			return mgr.Engage(ctx)
	//		}).Do()
	//		go mgr.Assign("group2", func(ctx Context) error {
	//			mgr := sup.BuildFiniteSupervisor()
	//			enroll, mgr_mill := sup.BuildStreamSupervisor()
	//			go mgr.Assign("mill", mgr_mill.Engage).Do()
	//			go mgr.Assign("ctrl", func(ctx Context) error {
	//				defer enroll.Close()
	//				for {
	//					select {
	//					case taskFn := <-taskQueue:
	//						go enroll.Assign("%", taskFn).Do()
	//					case <-ctx.Done():
	//						return nil
	//				}
	//			}).Do()
	//			return mgr.Engage(ctx)
	//		}).Do()
	//
	Do()
}

// this seems to be the existing internal boundTask thing.
type task struct {
	owner *supervisor
	name  string
	fn    Func
}

func (t task) Do() {
	t.owner.awaitEngaged() // could be wrapped into 'launch', but nice to see this line on stacks if debugging hangs.
	t.owner.launch(t.fn)   // wraps the fn with panichandling then invokes it with the owner's context.
}

type Supervisor interface {
	Engage(Context) error // launch supervision: all tasks are free to execute, and we'll eventually return the dominant error after no more tasks are enrollable and all are done.  note that this implements Func.
}

type Enroller interface {
	Assign(name string, fn Func) Task // announce a new task.  think of this like a chan send.  panics if the supervisor is complete.  accepts tasks if it's canceling; just immediately calls them with a canceled context.
	Complete()                        // no more tasks are coming.  call precisely once.  think of this as a chan close.
}

func BuildFiniteSupervisor(SupervisionOptions) Supervisor {
	return nil
}

type SupervisionOptions struct {
	TaskErrors func(error) error // default is return the arg, but you can replace it with e.g. a function that sends it to a channel and returns nil, or whatever.
}

//
// -- internal engine bits --
//

type supervisor struct{}

func (eng supervisor) awaitEngaged() {}
func (eng supervisor) launch(Func)   {}

//
// -- convenience methods for common assemblies --
//

func LaunchGroup(Supervisor, []Func) {
	// loop over `go mgr.NewTask("%").Bind(fn).Do()
}

// upon the result just call Engage!
func PrimeSupervisedGroup(SupervisionOptions, []Func) Supervisor {
	return nil
}
