package sup_test

import (
	"context"
	"sync"
	"testing"

	"github.com/warpfork/go-sup"
)

func TestPromises(t *testing.T) {
	t.Run("getnow should return empty", func(t *testing.T) {
		p := sup.NewPromise()
		val, err := p.GetNow()
		shouldEqual(t, val, nil)
		shouldEqual(t, err, sup.Nonblock)
	})
	t.Run("get should return after resolve", func(t *testing.T) {
		p := sup.NewPromise()
		go p.Resolve(14)
		res := p.Get(context.Background())
		shouldEqual(t, res.Value, 14)
		shouldEqual(t, res.Error, nil)
	})
	t.Run("get should return after cancel", func(t *testing.T) {
		p := sup.NewPromise()
		go p.Cancel()
		res := p.Get(context.Background())
		shouldEqual(t, res.Value, nil)
		shouldEqual(t, res.Error, context.Canceled)
	})
	t.Run("get should be cancellable", func(t *testing.T) {
		p := sup.NewPromise()
		ctx, cancel := context.WithCancel(context.Background())
		go cancel()
		res := p.Get(ctx)
		shouldEqual(t, res.Value, nil)
		shouldEqual(t, res.Error, sup.Nonblock)
	})
	t.Run("resolve after cancel should noop", func(t *testing.T) {
		p := sup.NewPromise()
		p.Cancel()
		p.Resolve(14)
		res := p.Get(context.Background())
		shouldEqual(t, res.Value, nil)
		shouldEqual(t, res.Error, context.Canceled)
	})
	t.Run("cancel after resolve should noop", func(t *testing.T) {
		p := sup.NewPromise()
		p.Resolve(14)
		p.Cancel()
		res := p.Get(context.Background())
		shouldEqual(t, res.Value, 14)
		shouldEqual(t, res.Error, nil)
	})
	t.Run("waitSelectably should fan-in", func(t *testing.T) {
		p1, p2, p3 := sup.NewPromise(), sup.NewPromise(), sup.NewPromise()
		gatherCh := make(chan sup.Promise)
		var r1, r2, r3 sup.ResolvedPromise
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			for {
				select {
				case p := <-gatherCh:
					switch p {
					case p1:
						r1.Value, r1.Error = p.GetNow()
					case p2:
						r2.Value, r2.Error = p.GetNow()
					case p3:
						r3.Value, r3.Error = p.GetNow()
					}
				}
				wg.Done()
			}
		}()
		p1.WaitSelectably(gatherCh)
		p2.WaitSelectably(gatherCh)
		p3.WaitSelectably(gatherCh)
		go p1.Resolve(1)
		go p3.Resolve(3)
		go p2.Cancel()
		wg.Wait()
		shouldEqual(t, r1.Value, 1)
		shouldEqual(t, r1.Error, nil)
		shouldEqual(t, r2.Value, nil)
		shouldEqual(t, r2.Error, context.Canceled)
		shouldEqual(t, r3.Value, 3)
		shouldEqual(t, r3.Error, nil)
	})
}
