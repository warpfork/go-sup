package sup_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/warpfork/go-sup"
)

// This eaxmple demonstrates a fan-out of goroutines in which one of the
// tasks errors... and go-sup automatically cancels any remaining tasks,
// while also ensuring only the errors from the first task are returned.
//
// This is the same basic content as ExampleSuperviseForkJoin, but now
// exercises all those extra safety features you just wouldn't get if you
// wrote this in plain Go yourself without some nice library assistance.
//
func ExampleSuperviseForkJoin_errorsTriggerSiblingCancellationg() {
	var foobarIn = map[string]int{
		"a": 1, "b": 2, "c": 3, "d": 4,
	}

	// We'll use this to force order in our child goroutines.
	// Without it, the example would run so fast we'd have no way to
	// expect which of the child tasks would successfully complete
	// versus be slow enough to get cancelled!
	var lockstepper int = 1
	var mu sync.Mutex

	// Our second task is a bomb: it'll return an error.
	// This will cause the later tasks to be cancelled!
	err := sup.SuperviseRoot(context.Background(),
		sup.SuperviseForkJoin("main",
			sup.TasksFromMap(foobarIn, func(ctx context.Context, k_, v_ interface{}) error {
				k, v := k_.(string), v_.(int)

				for {
					// Busy wait.  (But this is just a test; we won't be here for long.)
					mu.Lock()
					if lockstepper == v {
						defer func() {
							lockstepper++
							mu.Unlock()
						}()
						break
					}
					mu.Unlock()
					time.Sleep(1)
				}

				// ..... We can't really get this to wait until the supervisor
				//  is certain to have finished acknowledging the error, and also
				//   then to be certain the context quit channel close has propagated,
				//  unless we produce so much additional mutexing that it would
				//   actually make the scheduler behavior itself non-load-bearing.
				// So there's a hacky sleep here, I guess.
				// This gets this test to pass the "majority" of the time.
				//  And I believe if we had race-detector'able bugs, it would still
				//   trigger, since a sleep *is not* a memory fence; so that's good.
				time.Sleep(100)

				if ctx.Err() != nil {
					fmt.Printf("Oh no!  My context is %v!\n", ctx.Err())
					return ctx.Err()
				}

				if k == "b" {
					fmt.Printf("This task errors!\n")
					return fmt.Errorf("Boom!")
				}

				fmt.Printf("The task for %q completed :)\n", k)
				return nil
			}),
		),
	)

	fmt.Printf("final error: %v\n", err)

	// Output:
	//
	// The task for "a" completed :)
	// This task errors!
	// Oh no!  My context is context canceled!
	// Oh no!  My context is context canceled!
	// final error: Boom!
}
