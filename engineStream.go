package sup

import (
	"context"
	"sync/atomic"
)

type superviseStream struct {
	name        string
	taskGen     TaskGen
	phase       uint32
	reportCh    <-chan reportMsg
	groupCancel func()
	awaiting    map[*boundTask]struct{}
	results     map[*boundTask]*ErrChild
	firstErr    error
}

func (mgr superviseStream) Phase() Phase {
	return Phase(atomic.LoadUint32(&mgr.phase))
}

func (mgr superviseStream) init(tg TaskGen) Supervisor {
	mgr.phase = uint32(Phase_init)
	mgr.taskGen = tg
	return &mgr
}

func (mgr superviseStream) Name() string {
	return mgr.name
}

func (mgr *superviseStream) Run(parentCtx context.Context) error {
	// Enforce single-run under mutex for sanity.
	ok := atomic.CompareAndSwapUint32(&mgr.phase, uint32(Phase_init), uint32(Phase_running))
	if !ok {
		panic("supervisor can only be Run() once!")
	}

	// Allocate statekeepers.
	mgr.awaiting = make(map[*boundTask]struct{})
	mgr.results = make(map[*boundTask]*ErrChild)

	// Step through phases (the halting phase will return a nil next phase).
	for phase := mgr._running; phase != nil; {
		phase = phase(parentCtx)
	}

	return mgr.firstErr
}

func (mgr *superviseStream) _running(parentCtx context.Context) phaseFn {
	// Build the child status channel we'll be watching,
	// and the groupCtx which will let us cancel all children in bulk.
	reportCh := make(chan reportMsg)
	mgr.reportCh = reportCh
	groupCtx, groupCancel := context.WithCancel(parentCtx)
	mgr.groupCancel = groupCancel

	// Loop selecting over new task submissions, result collection, or
	//  accepting a group cancel instruction.  We'll only break out on
	//  errors, cancels, or if the taskgen channel is closed.
	for {
		select {
		case newTask, ok := <-mgr.taskGen:
			if !ok {
				return mgr._collecting
			}
			task := bindTask(newTask)
			mgr.awaiting[task] = struct{}{}
			go childLaunch(groupCtx, reportCh, task)
		case report := <-reportCh:
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
}

func (mgr *superviseStream) _collecting(parentCtx context.Context) phaseFn {
	atomic.StoreUint32(&mgr.phase, uint32(Phase_collecting))

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

func (mgr *superviseStream) _halting(_ context.Context) phaseFn {
	atomic.StoreUint32(&mgr.phase, uint32(Phase_halting))

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

func (mgr *superviseStream) _halt(_ context.Context) phaseFn {
	atomic.StoreUint32(&mgr.phase, uint32(Phase_halt))
	return nil
}
