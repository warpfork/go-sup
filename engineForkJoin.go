package sup

import (
	"context"
	"sync"
)

type superviseFJ struct {
	name     string
	tasks    []*boundTask
	mu       sync.Mutex
	phase    phase
	awaiting map[*boundTask]struct{}
	results  map[*boundTask]error
}

func (superviseFJ) _Supervisor() {}

func (mgr superviseFJ) init(tasks []Task) Supervisor {
	mgr.phase = phase_init
	mgr.tasks = bindTasks(tasks)
	mgr.awaiting = make(map[*boundTask]struct{}, len(tasks))
	mgr.results = make(map[*boundTask]error, len(tasks))
	return &mgr
}

func (mgr superviseFJ) Name() string {
	return mgr.name
}

func (mgr *superviseFJ) Run(parentCtx context.Context) error {
	// Enforce single-run under mutex for sanity.
	mgr.mu.Lock()
	if mgr.phase != phase_init {
		panic("supervisor can only be Run() once!")
	}
	mgr.phase = phase_collecting
	mgr.mu.Unlock()

	// Build the child status channel we'll be watching,
	// and the groupCtx which will let us cancel all children in bulk.
	reportCh := make(chan reportMsg)
	groupCtx, groupCancel := context.WithCancel(parentCtx)

	// Launch all child goroutines.
	for _, task := range mgr.tasks {
		mgr.awaiting[task] = struct{}{}
		go childLaunch(groupCtx, reportCh, task)
	}

	// Watch reports.
	//  This is the happy-path loop.
	//  If anyone errors or we're cancelled, jump down.
	var firstErr error
watch:
	for range mgr.tasks {
		select {
		case report := <-reportCh:
			delete(mgr.awaiting, report.task)
			mgr.results[report.task] = report.result
			if report.result != nil {
				mgr.phase = phase_halting
				firstErr = report.result
				break watch
			}
		case <-parentCtx.Done():
			mgr.phase = phase_halting
			firstErr = parentCtx.Err()
			break watch
		}
	}
	// Did we collect all reports without getting unhappy?  Nice; return.
	if mgr.phase == phase_collecting {
		mgr.phase = phase_halt
		return nil
	}

	// We're halting, not entirely happily.  Cancel all children.
	groupCancel()

	// Keep watching reports.
	//  This is the *un*happy loop (so we're not watching for parent cancel
	//   anymore; we're already moody and want to get the heck outta here).
	//  It's important to do this so we don't have goroutine leaks, and so
	//   we can gather all the child errors and report them if asked.
	for len(mgr.awaiting) > 0 {
		report := <-reportCh
		delete(mgr.awaiting, report.task)
		mgr.results[report.task] = report.result
	}
	mgr.phase = phase_halt
	return firstErr
}
