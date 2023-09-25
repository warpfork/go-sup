package tasktree

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/warpfork/go-sup"
)

func Test(t *testing.T) {
	rootCtx := context.Background()

	svr := sup.NewSupervisor(rootCtx)
	// First, just a regular task submission.
	go svr.Submit("bapper-0-5", &Bapper{0, 5}).Run()
	// Now, we'll create a sub-tree of supervision... starting with just a regular task func,
	//  and building a new supervisor inside it.  Not much magic.
	go svr.Submit("subtree", sup.TaskOfFunc(func(ctx context.Context) error {
		fmt.Printf("subtree task launched, named %s\n", sup.ContextName(ctx))
		subtreeSvr := sup.NewSupervisor(ctx)
		go subtreeSvr.Submit("bapper-5-10", &Bapper{5, 5}).Run()
		go subtreeSvr.Submit("bapper-10-15", &Bapper{10, 5}).Run()
		err := subtreeSvr.Run(ctx)
		fmt.Printf("subtree supervisor returned\n")
		return err
	})).Run()
	// Of course, the above is a pretty common thing to want to do!
	//  And at least three of those lines looked pretty extremely boilerplatey, right?
	//  So let's do a very similar thing now, but more tersely, with another helper function:
	// TODO define such a thing :D
	svr.Run(rootCtx)
}

type Bapper struct {
	start int
	count int
}

func (b *Bapper) Run(ctx context.Context) error {
	// Be a good citizen and always check if we were already cancelled, even just as we begin.
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for i := b.start; i < b.count+b.start; i++ {
		// Do our task.  It's a silly little side-effect.
		fmt.Printf("bap! %d from %s\n", i, sup.ContextName(ctx))
		// Idle a while (as a simulated placeholder for some other hard work),
		// or always be ready to accept that we've been cancelled.
		select {
		case <-time.After(1000 * time.Millisecond):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
