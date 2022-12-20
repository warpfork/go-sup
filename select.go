package sup

import (
	"time"
)

// Select offers a form of "safe select",
// which has similar purpose to the native golang select feature,
// but offers additional guard rails.
//
// Specifically, Select will ensure that context cancellation is checked
// along with any of the other given selectables.
// Select will also optionally include warnings for send operations
// that take an excessive amount of time to complete.
func Select(ctx Context, doThese ...Selectable) error {
	panic("todo")
}

type Selectable interface {
	Name() string

	// Deadlines as handled by this mechanism are imprecise, best-effort, and may be freely coalesced by the system.
	// Their intended use is for logging and detection of bad behaviors, not for normal control flow.
	SetOverdueReaction(deadline time.Time, callback func(Selectable)) Selectable

	// Set a callback to be invoked after the Selectable is completed.
	SetFollowup(func(Selectable)) Selectable

	// _internal is a marker method ensuring this type isn't implemented outside this library.
	// There are behaviors beyond what the interface can describe.
	// (Concretely, things implementing Selectable must contain something we can bind into a `reflect.SelectCase.Chan` value.)
	_internal()
}
