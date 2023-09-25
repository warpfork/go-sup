package pingpong

// This ping-pong implementation uses only supervision and golang channels (bare).
// You could also implement a similar thing using sup's wrapped channels that come with guardrails.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/warpfork/go-sup"
)

func TestPingpong(t *testing.T) {
	pinger := &Actor{
		config: Config{},
	}
	ponger := &Actor{
		config: Config{
			Ponger: true,
		},
	}
	pingChan := make(chan Msg)
	pongChan := make(chan Msg)
	pinger.wiring.Outbox = pingChan
	pinger.wiring.Inbox = pongChan
	ponger.wiring.Outbox = pongChan
	ponger.wiring.Inbox = pingChan

	rootCtx := context.Background()
	deadlinedCtx, _ := context.WithDeadline(rootCtx, time.Now().Add(2*time.Second))
	svr := sup.NewRootSupervisor(deadlinedCtx)
	go svr.Submit("pinger", sup.TaskOfSteppedTask(pinger)).Run()
	go svr.Submit("ponger", sup.TaskOfSteppedTask(ponger)).Run()
	err := svr.Run(deadlinedCtx)
	fmt.Printf("final error returned from root supervisor's run: %v\n", err)
	// ^ This'll say "context deadline exceeded", because that signal coming down from the deadlinedCtx is what halts this demo.
	time.Sleep(1 * time.Second)
	// What's really important is that you won't see any more messages from the pinger and ponger tasks after this :)
}

type Actor struct {
	wiring Wiring
	config Config
}

type Wiring struct {
	Inbox  <-chan Msg
	Outbox chan<- Msg
}

type Config struct {
	Ponger bool
}

type Msg struct {
	Increment int
}

func (a *Actor) FirstStep(ctx sup.Context) error {
	// If I'm a pinger: start get the ball rolling with a first message.
	if !a.config.Ponger {
		// Must be done in another select, because it must also abort if we receive the doneness signal.
		select {
		case a.wiring.Outbox <- Msg{}:
			return nil // succesfully sent.
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (a *Actor) RunStep(ctx sup.Context) error {
	// Select for incoming requests for action, or for the done signal channel to close.
	select {
	case m := <-a.wiring.Inbox:
		// This switch is just regular business logic -- processing the demo message.
		switch {
		case a.config.Ponger == true:
			fmt.Printf("Pong %d from %s!\n", m.Increment, sup.ContextName(ctx))
			// Send a response... in another select, because it must also abort if we receive the doneness signal.
			select {
			case a.wiring.Outbox <- m:
				return nil // succesfully sent.
			case <-ctx.Done():
				return ctx.Err()
			}
		case a.config.Ponger == false:
			m.Increment++
			fmt.Printf("Ping %d from %s!\n", m.Increment, sup.ContextName(ctx))
			// Send a response... in another select, because it must also abort if we receive the doneness signal.
			select {
			case a.wiring.Outbox <- m:
				return nil // succesfully sent.
			case <-ctx.Done():
				return ctx.Err()
			}
		default:
			panic("unreachable")
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}
