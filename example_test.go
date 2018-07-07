package sup_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/warpfork/go-sup"
)

// ExampleSuperviseForkJoin shows a variation on the common
// fan-out-then-collect model of basic parallel computation.
//
// In plain Go, you would write much the same thing -- declare some variable
// to hold your gathered results, a waitgroup to wait for total completion,
// and a mutex to keep your gathering of results race-free; then launch
// off all your goroutines.
//
// In Go with go-sup, this task is almost the same -- declaring the variable
// to hold your gathered results and mutexing the gather is still considered
// your application logic.  Go-sup handles the goroutine launch and waitgroup.
// In this example, we used a TasksFromMap helper function to generate tasks,
// but you can take manual control over this or use other helpers.
//
// In addition, go-sup takes care of:
//
//   - if any task errors, go-sup will immediately send cancellation to
//     all the other tasks.
//   - if any task fails to return in a timely fashion after cancellation,
//     go-sup can log this (configurably).
//   - if any task panics, go-sup can convert this to a regular returned error,
//     and manage cancellation as with other errors (configurably).
//
// So while fan-out-then-collect / fork-join computations are almost exactly
// the same amount of work to write using go-sup versus plain Go, in order to
// get the same features under adverse conditions, the amount of boilerplate
// (and yet tricky) channel wiring required in plain Go would be significant.
func ExampleSuperviseForkJoin() {
	var foobarIn = map[string]int{
		"a": 1, "b": 2, "c": 3,
	}

	var foobarOut = map[string]int{}
	var mu sync.Mutex

	// this must
	//   - handle the first error
	//     - that includes catching and reflowing panics -- configurable?
	//   - cancel all the siblings
	//   - accept their errors and sanity check that they're cancel-halts
	//     - do ??? if they're not -- something configurable, i guess
	//   - return the first error.
	err := sup.SuperviseForkJoin("main",
		sup.TasksFromMap(foobarIn, func(ctx context.Context, k_, v_ interface{}) error {
			k, v := k_.(string), v_.(int)

			// pretend this is slow :)
			v += 4

			// gather sync and logic is still up to you.
			mu.Lock()
			defer mu.Unlock()
			foobarOut[k] = v
			return nil
		}),
	).Run(context.Background())

	fmt.Printf("whee\n")
	fmt.Printf("%s", mapToStr(foobarOut))
	fmt.Printf("%v\n", err)

	// Output:
	//
	// whee
	//   - "a": 5
	//   - "b": 6
	//   - "c": 7
	// <nil>
}
