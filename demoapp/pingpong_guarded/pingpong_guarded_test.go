package pingpong

// This ping-pong implementation uses our guardrail'd channel types.
// You could also implement a similar thing using golang channels directly.

import (
	"context"
	"fmt"
	"testing"

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

	rootCtx := context.Background()
	svr := sup.NewSupervisor(rootCtx)
	go svr.Submit("pinger", sup.TaskOfSteppedTask(pinger)).Run()
	go svr.Submit("ponger", sup.TaskOfSteppedTask(ponger)).Run()
	err := svr.Run(rootCtx)
	if err != nil {
		panic(err)
	}
}

type Actor struct {
	wiring Wiring
	config Config
}

type Wiring struct {
	Inbox  sup.ReceiverChannel[Msg]
	Outbox sup.SenderChannel[Msg]
}

type Config struct {
	Ponger bool
}

type Msg struct {
	Increment int
}

func (a *Actor) RunStep(ctx sup.Context) error {
	// Might not look like much of a "select", with only one case in it!
	// But it *is* still technically a select, because we're implicitly also considering if it's time to quit!
	// (The internals of `sup.Select` and our wrapped channels are doing these checks for us.)
	return sup.Select(ctx,
		a.wiring.Inbox.RecvAndThen(func(m Msg) error {
			// This switch is just regular business logic -- processing the demo message.
			switch {
			case a.config.Ponger == true:
				fmt.Printf("Pong %d from %s!\n", m.Increment, sup.ContextName(ctx))
				// Send a response... in another select, because it must also abort if we receive the doneness signal.
				return sup.Select(ctx,
					a.wiring.Outbox.Send(m),
				)
			case a.config.Ponger == false:
				m.Increment++
				fmt.Printf("Ping %d from %s!\n", m.Increment, sup.ContextName(ctx))
				// Send a response... in another select, because it must also abort if we receive the doneness signal.
				return sup.Select(ctx,
					a.wiring.Outbox.Send(m),
				)
			default:
				panic("unreachable")
			}
		}),
	)
}
