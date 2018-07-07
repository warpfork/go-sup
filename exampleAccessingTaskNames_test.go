package sup_test

import (
	"context"
	"fmt"

	"github.com/warpfork/go-sup"
)

// myTask is a very primitive example task.
type myTask struct {
	name string
}

// This is the method running in the task.
// (Pretend it doesn't have access to t.name, if you like.)
func (t myTask) Run(ctx context.Context) error {
	fmt.Printf("hi from task %v -- my supervision path is %v :)\n",
		sup.CtxTaskName(ctx),
		sup.CtxTaskPath(ctx),
	)
	return nil
}

// This method is how the task declares its name in the first place.
func (t myTask) Name() string {
	return t.name
}

// This example shows some user-defined Task implementation with custom names,
// and how to access the name of your task from Context objects.
func ExampleSuperviseForkJoin_accessingTaskNames() {
	sup.SuperviseRoot(context.Background(),
		sup.SuperviseForkJoin("main",
			[]sup.Task{
				myTask{"one"},
				myTask{"two"},
				myTask{"three"},
			},
		),
	)

	// Unordered Output:
	//
	// hi from task one -- my supervision path is main/one :)
	// hi from task two -- my supervision path is main/two :)
	// hi from task three -- my supervision path is main/three :)
}
