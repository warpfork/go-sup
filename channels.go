package sup

import "reflect"

// SenderChannel wraps a `chan<-` and can be used together with `sup.Select()`
// to ensure that any send operations are also matched with a check for cancellation,
// and (optionally) an alert if the send blocks for an excessive amount of time.
// Panics for sending on a closed channel are also automatically captured
// and formatted into an error value that is returned calmly.
//
// Think of SenderChannel as enabling a "safe send", in short.
//
// SenderChannel is not strictly necessary to use in order to use other features
// of go-sup (like e.g. supervisors).  Regular go channels can still be used directly.
// SenderChannel just offers additional features.
//
// The performance overhead of using SenderChannel should be as minimal as possible.
// However, reflection is used internally to implement `sup.Select`;
// to add the features we do while reaching absolute zero overhead is not possible.
// (Sending a value without having that value escape to the heap is also not possible,
// due to limitations of handling values with reflection.)
type SenderChannel[T any] struct {
	Chan chan<- T
}

func (ch *SenderChannel[T]) Send(v T) Selectable {
	panic("todo")
}

func (ch *SenderChannel[T]) SendAndThen(v T, cb func() error) Selectable {
	panic("todo")
}

type selectableSend struct {
	ch  reflect.Value
	val reflect.Value
}

// CHALLENGE: I have no idea how, syntactically, we'll loop over a set of selectables for sending, without generating an obscene amount of garbage.
// When golang native syntax does this, it's not producing any values, and certainly not forcing them onto the heap.
// When we do it?  Much trickier.
//  - creating a Selectable creates a heap escape, almost unavoidably, because interface boxing.
//    - ... we already made Selectable a closed interface; maybe we should consider making it a concrete type entirely.
//  - the callback is considerably likely to cause an allocation if it encloses over any values at all.  And it usually will.
//    - this is the biggest part of the challenge, because afaik, golang will generate a *new* closure object each time this is encountered.  (TODO: verification needed.)
//  - reflect.ValueOf isn't free itself.
//    - ... although this might not be the worst.  reflect.ValueOf is actually fairly cheap and returns a struct.
//      (It does force the value to escape, but mind the subtle distinction: that's the _referenced value_ being forced to escape, not the struct that's returned by ValueOf.  So repeating it is of no consequence; it already escaped the first time.)

// CHALLENGE: receive has a similar challenge to send, in that it might produce garbage: it's gonna have to bind a callback.
// That'll almost certainly cause a garbage allocation if it enclosures over anything -- and in the syntactically normal and obvious ways to write things, it will.
// Alternative: the whole Select function can return the selected value, and some indicator of the case.  (And that's what the lowest level feature does.)  But this would result in you needing to write... another whole switch.  And with ugly cases.
// Perhaps we can do both and let the user pick?

// For both of the above: the next step one is necessarly just "implement it, benchmark it, and we'll see".
// Because the rest of the supervision library components don't need to depend on this directly in any way,
// whether or not this works at extreme performance isn't a blocker for determining whether this project as a whole is worth-while.

type ReceiverChannel[T any] struct {
	Chan <-chan T
}

// Recv receives a message, but does nothing with it, discarding it.
// Use RecvAndThen to specify a function that receives the value.
func (ch *ReceiverChannel[T]) Recv() Selectable {
	panic("todo")
}

func (ch *ReceiverChannel[T]) RecvAndThen(cb func(T) error) Selectable {
	panic("todo")
}

// TODO there's no clear way to distinguish send of a nil from a shutdown in this receive API yet.

// REVIEW: the function names on senders and receivers.  It's concerning that they sound imperative -- they don't _sound_ like you still need to submit them to selection.
//
// It would be neat if we *could* make a "Send" and "Recv" that are imperative one-liners that do their own Select together with the quit channels.
//  But the value of this might be somewhat limited, because we'd still either wish we had some kind of macro for the "if err := Send(...); err != nil {` boilerplate,
//   or... we'd have to consider introducing a convention of using panics to carry quits out.
//    I suppose we're in a decent position to declare that convention in this library, and offer support for cleaning up after those panics too, but oof.

// NewChannel plus a package var that lets you force all capacity requests greater than 1 to be forced to 1.
