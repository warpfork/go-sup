Recommendations for structuring concurrent code with go-sup
===========================================================

(And just some general recommendations, too, regardless of if you're using go-sup.)



Pitfalls!
---------

We've tried to design the API of go-sup to have minimal opportunities to do something wrong,
or to communicate clearly when something isn't assembled correctly,
but we couldn't handle every possible situation.

#### Be careful to 'go' on the right things.

Make sure that when launching a task, the `go` keyword is applying
on the `Run()` function.

Good:

```go
go thesupervisor.Submit("taskname", task).Run()
```

Not good:

```go
go thesupervisor.Submit("taskname", task)
```

Unfortunately, there's no way the library can call out the latter situation.



Actor Conventions
-----------------

We suggest that most complex concurrency code be structured around a design pattern called "actors".

"Actors" are pieces of code that maintain some internal state, and manage all interactions by passing messages.
This can be a good design pattern because whenever you want to know what logic applied on some piece of state,
the answer is of nicely limited scope: it's some code in the actor.
Actors also tend to scale nicely.

When writing an actor-style piece of code, here are some conventions we recommend:

- Make a whole package to contain the actor.
- Have one type which is the actor.
- Have a type to collect all the "wiring" -- all the other channels and callbacks the actor will use to connect to other parts of the system.
- Have a type for "config" -- keep this distinct from "wiring" just because it improves readability.
- The type which is the actor should combine the "wiring" struct, the "config" struct, and then add whatever state it needs from there as private fields.

Something roughly like this:

```go
package maestro

type Actor struct {
	log    somekindof.Logger
	wiring Wiring
	config Config

	state recordsStruct
}

type Wiring struct {
	FooEventInbox <-chan FooEvent
	FrobnozService chan<- frobnoz.Request
}

type Config struct {
	Color string
	Size int
}

type recordsStruct struct {
	// whatever private state the actor maintains...
}

// Run is implemented by Maestro so that it can be treated as a go-sup task,
// and can be easily supervised!
func (a *Actor) Run(ctx Context) error {
	// business logic goes here...
}

```

(We recommend literally using the names "Wiring" and "Config" for these structures.
If you followed the recommendation about using a whole package for the actor,
then these names are already unambiguous... and following a convention can reduce unnecessary cognitive overhead.)

Then, either make the "wiring" and "config" fields (and "log", if you've chosen to include that separately from the others; or, perhaps you choose to treat that as just more "wiring"?) public,
so there's no need for a constructor function; or, make a constructor function, something like this:

```go
package maestro

func New(log somekindof.Logger, wiring Wiring, config Config) *Actor {
	a := &Actor{
		log:     log,
		wiring:  wiring,
		config:  config,
		records: initializeRecords(),
	}
	return a
}
```

Note that we don't recommend putting a "name" field in the actor.
This is because -- if you're using `go-sup` -- we pass names down in the Context.
(We designed `go-sup` to handle names this way because it lets the calling code name things,
rather than relying on all code to advertise its own naming systems,
and we believe it tends to result in the development of more easily reusable components.)

This pattern can vary.
For example, putting state in the Actor struct at all is totally unnecessary:
an Actor might choose to keep the entirety of its state encapsulated in variables scoped to the `Run` function,
and never save it to a struct at all!

### Yet more actor conventions...

Some actor systems literature considers it standard to attach the sender
(or some channels to send messages to them) to every piece of data sent to another actor.
This probably goes almost without saying, but for clarity anyway:
in go-sup, of course we may no such enforcement.
If sending some kind of return-channels is a good idea in your application, go for it; if it's not, don't.
Your messages may contain whatever they want.


Agent Conventions
-----------------

"Agent" conventions are roughly the opposite of "actor" patterns:
instead of some logic owning a specific piece of memory exclusively,
and managing all interactions by passing messages,
an "agent" is a piece of logic that is applied to a larger piece of shared memory,
under either some kind of mutex or some kind of update application rules.

There's less to say about agent patterns because they're much more "it depends".

Some agent design patterns are truly just one big mutex,
with a `ApplyMe(func (old State) (new State))` feature.
Others may be more complex and granular.

Agent patterns may be based on mutexing systems, or,
may instead take the approach of requring the logic of an agent be side-effect-free,
so that the agent can be invoked repeatedly and used within other forms of concurrency control that involve retries.

Because agent patterns can be so diverse, we have no particular guidance to offer,
other than identifying the pattern with a name, in case the reader is not yet familiar with it.
