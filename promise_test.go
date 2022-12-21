package sup

import (
	"context"
	"sync"
	"testing"
)

func TestPromise(t *testing.T) {
	p, resolve := NewPromise[int]()
	var wg sync.WaitGroup
	interactions := []func(){
		func() {
			p.Await(context.Background())
			if p.Value() != 9 {
				panic("wrong value")
			}
			wg.Done()
		},
		func() {
			<-p.ResolvedCh()
			if p.Value() != 9 {
				panic("wrong value")
			}
			wg.Done()
		},
		func() {
			p.WhenResolved(func() {
				if p.Value() != 9 {
					panic("wrong value")
				}
				wg.Done()
			})
		},
		func() {
			ch := make(chan Promise[int])
			p.ReportTo(ch)
			<-ch
			if p.Value() != 9 {
				panic("wrong value")
			}
			wg.Done()
		},
		func() {
			resolve(9)
		},
	}
	// future: shuffle for good measure
	wg.Add(len(interactions) - 1)
	for _, interaction := range interactions {
		go interaction()
	}
	wg.Wait()
}
