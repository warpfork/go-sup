package sup

import (
	"sync"
	"sync/atomic"
)

// Promise controls read access to a value which may not have been produced yet,
// and offers various ways to wait for it to become availabe or be notified when it does.
//
// Promise offers many styles of use:
// polling for doneness,
// awaiting in a blocking fashion,
// attaching callbacks,
// getting a single channel which is closed as a signal,
// or supplying a channel which the Promise will submit itself to when it becomes done
// are all supported.
// Any (or all) of these styles may be used freely;
// they vary in personal preference as well as in how they scale
// (e.g. submitting the Promise to a channel is the most scalable way to do fan-in
// if there are many Promises you want to collect from, but is also generally
// the most syntactically burdensome to set up).
//
// Promise carries one value, which is defined by the type parameter.
// This may be any type; Promise does not care.
// Typically, it's either a "result" value of some kind,
// or an error value, or a structure that contains both.
// Access to Value without having awaited doneness in some way is a race condition.
//
// Create a Promise by use of NewPromise(), which returns both a Promise and a resolve func;
// assign the value and cause the Promise to notify all waiters by calling the resolve function with a value.
//
// Promise is for general use.
// It also appears in go-sup itself in APIs such as for waiting for a SupervisedTask to complete.
//
// Methods like WhenResolved and ReportTo may be used after the Promise is already completed;
// if so, they are called in a new short-lived goroutine
// (this is in order to ensure no algorithm is required to support reentrant locks,
// which would otherwise be a threat if we attempte to reuse the current goroutine).
//
// The order in which notifications about a Promise becoming resolved are sent is unspecified.
// They may be serviced by a single goroutine, and reactions should not block.
//
// Some good behavior is necessary when using variations of notification
// which may potentially allow you to block or to panic.  (In short: don't.)
// Panicking in a callback given to WhenResolved is extremely inadvisable:
// it will result in other notifications about Promise completion failing to be sent,
// and the goroutine which receives the panic is considered to be undefined.
// Blocking in a callback given to WhenResolved is also inadvisable,
// as it will result in delays sending other notifications,
// and may spread delays to unexpected parts of the program.
// Failure to quickly service the `chan Promise` used for ReportTo is similarly inadvisable,
// as it will result in delays sending other notifications,
// and may spread delays to unexpected parts of the program.
// If you need to perform reactions to a promise which may need to block or may themselves error,
// you should organize that code into other pieces of program logic,
// and use the unblockable notification mechanisms (namely, ResolvedCh)
// instead of using WhenResolved or ReportTo.
type Promise[V any] interface {
	Value() V                   // Access the value.  If the Promise is not yet resolved, this is undefined, and a race condition.
	Await(Context) (ready bool) // Blocks until the Promise is resolved or the Context is cancelled.  Returns false if it returned because of context cancellation (in which case accessing Value is still a race).
	IsResolved() bool
	ResolvedCh() <-chan struct{}
	WhenResolved(cb func())
	ReportTo(chan<- Promise[V])
}

func NewPromise[V any]() (_ Promise[V], resolve func(V)) {
	p := &promise[V]{ctrl: 1, ch: make(chan struct{}, 0)}
	return p, p.resolve
}

type promise[V any] struct {
	value V
	ctrl  uint32 // CAS on this.  1=wait; 2=mid-resolve,plzretry; 0=done.

	ch chan struct{} // always instantiated (so we don't have to use mu when consulting it).

	mu sync.Mutex // only needed for controlling changes to the notifiers below, when ctrl != 0.

	cb1    func()
	cbList []func() // the single value is used first to avoid an alloc in common case.

	rep1    chan<- Promise[V]
	repList []chan<- Promise[V] // the single value is used first to avoid an alloc in common case.
}

func (p *promise[V]) resolve(value V) {
	// Swap to mid-resolve state.  Panic if we're not currently in the wait state.
	swapped := atomic.CompareAndSwapUint32(&p.ctrl, 1, 2)
	if !swapped {
		panic("promise is already resolved, cannot resolve again")
	}
	// In the mid-resolve state,
	//  set the value -- (we needed a CAS before doing this so we don't overwrite the value racily if there was a previous resolve!)
	// Then swap to done state: value is now permissible to read.
	p.value = value
	swapped = atomic.CompareAndSwapUint32(&p.ctrl, 2, 0)
	if !swapped {
		panic("promise internal error: impossible concurrent transition")
	}
	// In the done state:
	// - Dispense the channel close signal first -- that one's easy.
	// - Then grab the mutex around all the other notification hooks
	//    (there should already be no more modifications to this, as any new incomings should see the done state,
	//     but we don't want to race with the tail end of an append that's already in progress).
	// - Then emit notifications for all the other mechanisms.
	//    Nil all those pointers afterwards, as help to the GC.
	//     (Someone may hang onto the promise well after it's become done,
	//      but that doesn't need to translate to prolonged lifetime for anything that was only referenced for notification purposes.)
	close(p.ch)
	p.mu.Lock()
	if p.cb1 != nil {
		p.cb1()
		p.cb1 = nil
	}
	for _, cb := range p.cbList {
		cb()
	}
	p.cbList = nil
	if p.rep1 != nil {
		p.rep1 <- p
		p.rep1 = nil
	}
	for _, rep := range p.repList {
		rep <- p
	}
	p.repList = nil
	// And we'll just... quietly never unlock that mutex again.
	// It's a library bug if it's ever consulted again; so as a form of defense in depth, let's make sure it's noticable if something does.
}

func (p *promise[V]) Value() V {
	return p.value
}

func (p *promise[V]) Await(interruptable Context) (ready bool) {
	select {
	case <-p.ResolvedCh():
		return true
	case <-interruptable.Done():
		return false
	}
}

func (p *promise[V]) IsResolved() bool {
	return atomic.LoadUint32(&p.ctrl) == 0
}

func (p *promise[V]) ResolvedCh() <-chan struct{} {
	return p.ch
}

func (p *promise[V]) WhenResolved(cb func()) {
	if p.IsResolved() {
		go cb()
		return
	}
	p.mu.Lock()
	// Double check, now under the lock, in case resolution was in progress concurrently and raced us to acquire the notifications hooks mutex:
	if p.IsResolved() {
		go cb()
		p.mu.Unlock()
		return
	}
	// Set the callback for later.
	if p.cb1 == nil {
		p.cb1 = cb
	} else {
		p.cbList = append(p.cbList, cb)
	}
	p.mu.Unlock()
}

func (p *promise[V]) ReportTo(repCh chan<- Promise[V]) {
	if p.IsResolved() {
		go func() { repCh <- p }()
		return
	}
	p.mu.Lock()
	// Double check, now under the lock, in case resolution was in progress concurrently and raced us to acquire the notifications hooks mutex:
	if p.IsResolved() {
		go func() { repCh <- p }()
		p.mu.Unlock()
		return
	}
	// Set the reporting channel for later.
	if p.rep1 == nil {
		p.rep1 = repCh
	} else {
		p.repList = append(p.repList, repCh)
	}
	p.mu.Unlock()
}
