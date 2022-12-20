package sup

// EngineBuilder is WIP, draft, and entirely non-final.
type EngineBuilder[TaskAsk any] interface {
	SetupTaskSource(
		newAsksChannel ReceiverChannel[TaskAsk],
		nameSuggester func(TaskAsk) string,
	)

	// SetLauncher can be used to set your own source of goroutines.
	// This can be as simple as:
	//
	//		theEngine.SetLauncher(func(t SupervisedTask) { go t.Invoke() })
	//
	// Setting a launcher func is optional!
	// The main reason to do this is if you want to see a specific line number
	// appear as the origin of a goroutine in case it should come up
	// in any panics or other golang runtime debugging mechanisms.
	// If you don't set your own launcher, the line numbers appearing
	// in such situations will always be from somewhere inside the go-sup package,
	// which may be less informative.
	//
	// The launcher func will be called for each launch of a task.
	// (Goroutines are not reused.)
	//
	// It is generally expected that the launcher func should return immediately,
	// and is implemented by launching a new goroutine.
	// If you want to control scheduling in a more fine-grained way,
	// it is also an option to use supervisors directly yourself,
	// as they do not enforce any scheduling opinions
	// and leave the running of SupervisedTask entirely in your hands.
	SetLauncher(func(SupervisedTask))
}

// `Engine` may or may not be a useful interface; if it is, it's probably a superset of `Supervisor`.
// I'm on the fence about the builder pattern above.  It works fine, but we haven't used that elsewhere (e.g. `Supervisor` already takes a "i'm your one-stop-shop" philosophy and freely mixes configuration, sender funcs, and so on).

// Actual init for an engine would be roughly:
// - make a supervisor
// - make a submission controller actor, and add it to the supervisor
// - make a pool supervisor, and add it to the supervisor
// - wire those two together in the obvious way
// - SetReturnOnEmpty(false) on the pool supervisor
// - have the submission controller actor SetReturnOnEmpty(true) on the pool supervisor when it's told to spin down
// - ready; run the whole tree

// EngineBuilder is WIP, draft, and entirely non-final.
type EngineSubmitter interface {
	// May actually just be a SenderChannel and not much else.
	// Closing the channel is sufficient to indicate that it's time to wind down the pool.
	// May not be necessary to declare a whole type just for this.
}
