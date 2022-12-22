package sup

// NameSelectionStrategy is just a syntax hack to gather some function names --
// methods on this type match the needs of the Supervisor.SetNameSelectionStrategy callback system.
var NameSelectionStrategy = struct {
	Default func(requested, attempted string, attempts int) (proposed string)
}{
	Default: func(requested, attempted string, attempts int) (proposed string) {
		if attempts > 0 {
			return attempted + "+1"
		}
		// TODO: the described shenanigans about "%" replacement.
		return requested
	},
}

// FUTURE: something based on ulid sounds like a nice idea.  although I don't want to take on a dep for it.  and it would be a bit textually large.

// FUTURE: consider using %p; it's awfully handy.
