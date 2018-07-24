package sup

import (
	"context"
	"sync/atomic"
)

type superviseFJ struct {
	name        string
	tasks       []*boundTask
	phase       uint32
	reportCh    <-chan reportMsg
	groupCancel func()
	awaiting    map[*boundTask]struct{}
	results     map[*boundTask]error
	firstErr    error
}

func (superviseFJ) _Supervisor() {}

func (mgr superviseFJ) init(tasks []Task) Supervisor {
	mgr.phase = uint32(phase_init)
	mgr.tasks = bindTasks(tasks)
	return &mgr
}

func (mgr superviseFJ) Name() string {
	return mgr.name
}

func (mgr *superviseFJ) Run(parentCtx context.Context) error {
	// Enforce single-run under mutex for sanity.
	ok := atomic.CompareAndSwapUint32(&mgr.phase, uint32(phase_init), uint32(phase_running))
	if !ok {
		panic("supervisor can only be Run() once!")
	}

	// Allocate statekeepers.
	mgr.awaiting = make(map[*boundTask]struct{}, len(mgr.tasks))
	mgr.results = make(map[*boundTask]error, len(mgr.tasks))

	// Step through phases (the halting phase will return a nil next phase).
	for phase := mgr._running; phase != nil; {
		phase = phase(parentCtx)
	}

	return mgr.firstErr
}

func (mgr *superviseFJ) _running(parentCtx context.Context) phaseFn {
	// Build the child status channel we'll be watching,
	// and the groupCtx which will let us cancel all children in bulk.
	reportCh := make(chan reportMsg)
	mgr.reportCh = reportCh
	groupCtx, groupCancel := context.WithCancel(parentCtx)
	mgr.groupCancel = groupCancel

	// Launch all child goroutines... then move immediately on to "collecting".
	//  The joy of a fork-join pattern is this loop is simple.
	for _, task := range mgr.tasks {
		mgr.awaiting[task] = struct{}{}
		go childLaunch(groupCtx, reportCh, task)
	}
	return mgr._collecting
}

func (mgr *superviseFJ) _collecting(parentCtx context.Context) phaseFn {
	atomic.StoreUint32(&mgr.phase, uint32(phase_collecting))

	// We're not accepting new tasks anymore, so this loop is now only
	//  for collecting results or accepting a group cancel instruction;
	//  and it can move directly to halt if there are no disruptions.
	for len(mgr.awaiting) > 0 {
		select {
		case report := <-mgr.reportCh:
			delete(mgr.awaiting, report.task)
			mgr.results[report.task] = report.result
			if report.result != nil {
				mgr.firstErr = report.result
				return mgr._halting
			}
		case <-parentCtx.Done():
			mgr.firstErr = parentCtx.Err()
			return mgr._halting
		}
	}
	return mgr._halt
}

func (mgr *superviseFJ) _halting(_ context.Context) phaseFn {
	atomic.StoreUint32(&mgr.phase, uint32(phase_halting))

	// We're halting, not entirely happily.  Cancel all children.
	mgr.groupCancel()

	// Keep watching reports.
	for len(mgr.awaiting) > 0 {
		report := <-mgr.reportCh
		delete(mgr.awaiting, report.task)
		mgr.results[report.task] = report.result
	}

	// Move on.
	return mgr._halt
}

func (mgr *superviseFJ) _halt(_ context.Context) phaseFn {
	atomic.StoreUint32(&mgr.phase, uint32(phase_halt))
	return nil
}
