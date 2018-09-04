package sup

import (
	"context"
	"errors"
	"sync"
)

type Promise interface {
	Cancel()             // cancels the promise, effectively resolving it with nil.
	Resolve(interface{}) // sets the value.  panics on repeat use.

	Get(Context) ResolvedPromise   // blocking.  waits and returns access to the resolved value.
	GetNow() (interface{}, error)  // nonblocking.  returns (nil,promise.Nonblock) if not yet resolved; error may be context.Canceled or promise.Nonblock or nil if resolved.
	Wait(Context)                  // blocking.
	WaitSelectably(chan<- Promise) // nonblocking.  cause ourself to be sent to this channel when we become resolved.  multiple use panics.
	WaitCallback(func(Promise))    // nonblocking.  alternative to WaitSelectably which you can use if e.g. you need to send to multiple chans without waiting on each other or otherwise control rejection.  multiple use panics.
}

type ResolvedPromise struct {
	Value interface{} // the resolved value.  if nil, you may also need to check Error() to see if the promise was canceled.
	Error error       // nil or context.Canceled (or promise.Nonblock if Get was called and its context canceled).
}

var Nonblock = errors.New("promise nonblock")

// NewPromise returns a new unresolved promise.
// You can start waiting on it immediately, and resolve it (or hand it off
// to someone else to resolve) at your leisure.
func NewPromise() Promise {
	return &promise{waitCh: make(chan struct{})}
}

// NewDiscardingPromise returns a dummy promise where resolved values are
// discarded and all reader and waiter methods panic.
// (Resolve still has the set-once check but remembers no content.)
func NewDiscardingPromise() Promise {
	return &discardPromise{}
}

type promise struct {
	ResolvedPromise
	mu      sync.Mutex
	waitCh  chan struct{}
	afterCh chan<- Promise
	afterFn func(Promise)
}

func (p *promise) Cancel() {
	p.mu.Lock()
	if p.Value != nil || p.Error != nil {
		p.mu.Unlock()
		return
	}
	p.Error = context.Canceled
	p.notifyAndUnlock()
}
func (p *promise) Resolve(v interface{}) {
	p.mu.Lock()
	if p.Error != nil {
		// i've been raced.  drop my effect.
		p.mu.Unlock()
		return
	}
	if p.Value != nil {
		// i've been misused!  rage.
		p.mu.Unlock()
		panic("multiple Resolve() calls on Promise")
	}
	p.Value = v
	p.notifyAndUnlock()
}
func (p *promise) Get(ctx Context) ResolvedPromise {
	select {
	case <-p.waitCh:
		return p.ResolvedPromise
	case <-ctx.Done():
		return ResolvedPromise{nil, Nonblock}
	}
}
func (p *promise) GetNow() (v interface{}, err error) {
	p.mu.Lock()
	v, err = p.Value, p.Error
	p.mu.Unlock()
	if v == nil && err == nil {
		err = Nonblock
	}
	return
}
func (p *promise) Wait(ctx Context) {
	select {
	case <-p.waitCh:
	case <-ctx.Done():
	}
}
func (p *promise) WaitSelectably(afterCh chan<- Promise) {
	p.mu.Lock()
	if p.afterCh != nil {
		p.mu.Unlock()
		panic("multiple WaitSelectably() calls on Promise")
	}
	p.afterCh = afterCh
	if p.Value != nil || p.Error != nil {
		afterCh <- p
	}
	p.mu.Unlock()
}
func (p *promise) WaitCallback(afterFn func(Promise)) {
	p.mu.Lock()
	if p.afterFn != nil {
		p.mu.Unlock()
		panic("multiple WaitCallback() calls on Promise")
	}
	p.afterFn = afterFn
	if p.Value != nil || p.Error != nil {
		afterFn(p)
	}
	p.mu.Unlock()
}
func (p *promise) notifyAndUnlock() {
	afterCh, afterFn := p.afterCh, p.afterFn
	p.mu.Unlock()
	close(p.waitCh)
	if afterCh != nil {
		afterCh <- p
	}
	if afterFn != nil {
		afterFn(p)
	}
}

type discardPromise struct {
	mu       sync.Mutex
	resolved bool
}

func (p *discardPromise) Cancel() {}
func (p *discardPromise) Resolve(interface{}) {
	p.mu.Lock()
	if p.resolved {
		p.mu.Unlock()
		panic("multiple Resolve() calls on Promise")
	}
	p.resolved = true
	p.mu.Unlock()
}
func (p discardPromise) Get(Context) ResolvedPromise   { panic("discardpromise") }
func (p discardPromise) GetNow() (interface{}, error)  { panic("discardpromise") }
func (p discardPromise) Wait(Context)                  { panic("discardpromise") }
func (p discardPromise) WaitSelectably(chan<- Promise) { panic("discardpromise") }
func (p discardPromise) WaitCallback(func(Promise))    { panic("discardpromise") }
