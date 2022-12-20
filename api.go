package sup

type Runnable interface {
	Run(Context) error
}

type TaskMonitor interface {
	PeekState() TaskState // State is an atomic read of the task's state.
	PeekError() error
	Await()
	Notify(chan<- TaskMonitor) // n.b. if this send takes an unreasonably long time, the supervisor will be notified.  TODO use a type to wrap this channel so it can be named?

	// TODO the await and notify stuff should really just be an incident of Promise.
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

func RunnableOfFunc(func(Context) error) Runnable {
	panic("todo")
}
