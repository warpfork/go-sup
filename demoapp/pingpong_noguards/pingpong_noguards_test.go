package pingpong

// This ping-pong implementation uses only supervision and golang channels (bare).
// You could also implement a similar thing using sup's wrapped channels that come with guardrails.

import (
	"fmt"

	"github.com/warpfork/go-sup"
)

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
