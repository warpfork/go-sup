Perspective of the Design
=========================

Overview
--------

Any computation you can do which may be long takes the form `func(Context) error`.
(You can get anything into this form by currying any other parameters.)

Given a bunch of these things, which may or may not be concurrent, we want to:

- A) assemble them into pools where failure of one cancels the others;
- B) assemble them into pools where the first one to fail reports its error
   (but the others are also gathered);
- C) assemble them into pools where we can wait for all of them (either to
   return or be cancelled before they began);



Why ${choice[*]}...?
----------------------

### Why the whole NewTask/Do complexity?

Decoupling _supervision_ from _sequencing and concurrency_.

This means you can:

- implement task pools and rate-limiting;
- go further to implement advanced ordering like priority heaps;
- or go the other way and opt out of concurrency entirely, doing a bunch of
  tasks serially on a single goroutine.

... any of which can be done without changing a single thing about your
cancellation, supervision, and error handling strategies.

Some of these choices, like foregoing concurrency entirely, may seem odd at
first glance.  However, some situations demand it: for example, suppose you're
working on graphics programming: this typically requires a bunch of operations
to all be performed on the same thread.  It's important that you should be able
to do this without giving up on supervision, or really changing anything at all
about how your program uses supervision.

Hoisting the `Do` method into the caller's responsibilities has another bonus:
it makes the `go` keyword appear in the business logic rather than in
the engine code of go-sup.  This is neat for two reasons:
First, the sheer human niceity of having syntax highlighting around something
significant as a new goroutine;
Second, if the runtime emits stacktraces which say where a goroutine
was launched from, they'll say something meaningful and point to line numbers
that have something to do with your tasks, rather than always referring you to
an opaque and abstract location inside the go-sup library code.

There are also helper methods for submitting a whole array of tasks at once,
and immediately launching each of them in their own concurrent goroutine, since
this is admittedly one of the most common use cases.  However, be advised of
what you're giving up on if you use this syntactic sugar.

### Why are there no getter methods on tasks for watching it?

Observing the state of a task is a matter of business logic, and if you need
to inspect this, then you should write business logic in your task that
cooperates with your needs.

Task is a builder: it's there to guide you through the various optional
configurations that can be applied before launch.
If you need to supervise it... that's what supervisors are for.

(You may appreciate `sup.Promise` for this, though: it allows non-blocking
result checks, blocking result checks, and scalable channel/select-based
waiting on groups.)

### Why doesn't Task.Do launch immediately?

It would be irresponsible to start working on a task before the supervision
and error reporting tree has been engaged.

// FIXME i haven't the heart to throw the following text away, but it
//  is confused on whether we're talking about blocking or belated trampolining.

We've already covered the reason why `Do()` is a blocking method (e.g., so
that you can choose-your-adventure on sequencing and concurrency);
so that you can't call `someTask.Do()` and `itsSupervisor.Engage()` on
the same goroutine, since both block.
This is a feature.

It's important that `someTask.Do()` doesn't invoke the taskFunc until the
supervisor is engaged because A) we don't have a Context yet, so in addition
to being simply impossible to write such code, we'd be semantically in the
wrong because we'd have started something without a way to cancel it;
and B) if you failed to then call `itsSupervisor.Engage()`, error handling
would quietly become lost, and it's important that the library aims you away
from the possibility of mistakes like this.

### Why do I have to say NewSupervisor and supervisor.Engage?  Why not both in one?

You *could* call those two immediately sequentially and continue to submit new
tasks afterward.  But since Engage parks the goroutine until all the supervised
tasks return, you'd need another goroutine to do so.

Separating the creation of the supervisor (so we can start assigning tasks to it)
from engaging it makes it possible to use the current goroutine to assign tasks,
*then* park that same current goroutine in the supervision work -- no additional
goroutines necessary.

### Why can't I create a task and assign it to a supervisor later?

If tasks could be created with no supervision, it would make for an API which
has more states which are invalid, which is a worse API.

For example, adding a task with a duplicate name would become an error which
only shows up after you've finished all of the assembly work.

Even more importantly, gating creation of a task on the supervisor means we can
immediately reject potential new tasks if the supervisor is already closing,
rather than leaving that to be a race condition.

You can stack up configurations of a task with a pattern of customization funcs
like `func MyCustomize(Task) Task`, and use like
`go Customize(mgr.NewTask("name")).Do()`, and this composes reasonably well.

### Why do tasks get launched even if the supervisor is already cancelled?

So that if they're a task with a Promise, they can cancel it.

If you have a badly-written task that doesn't respond to cancellations promptly
if it gets launched, then you would be playing with fire by hoping cancellation
of the supervisor would race to beat launch of the task anyway.

### Why are names assigned at NewTask?  Why isn't there a Task interface that can export a Name function?

It's correct to defer naming to the code above; the code above has more
semantic context and knows what will be meaningful.

Forcing a task to be only code and not have a name also steers away from
situations where one might implement a task which is entirely reusable... until
it became burdened with a const name.

Task as an interface in general sets expectations wrong.  If your task is a
method, then it should be a non-pointer one, and treat the struct as args;
if it has any state, it should put that in another struct, internally; it
should not mutate its original args.  This is somewhat academic, because go-sup
isn't *going* to run your func multiple times... however, following this rule
of thumb will probably cause you to write better code which you may only later
discover to be nicely reusable.

// FIXME above is misguided.  taskfunc is fully curried; so unless it's
// side-effecting, the number of reasons to multipley submit it are *none*.
// the main thing that's dumb is a name func which tends to make you write
// ... you know what it's not even wrong, it's just that you usually don't
// want to write a correct generation function.  it's either number-shrug
// in the pool or the caller should have a name for it.
// just stop this whole thread it's wrong and getting wronger and weedier.

It's also syntactically more pleasant to be able to take a method and return it
as a taskfunc than it is to have a pure function and need to wrap it in a task
obj.

// TODO: supervisor builders themselves are guilty of this.  they should return
// a task that, when called, *launches a new supervisor* -- why is it oneshot?
//
// ... this is awfully hard to get around.
// forkjoiners could become multishot, because launch means submissioncomplete.
// streamengines are, well, stateful.  we could forbid accumulating tasks before
// engagement, but that's not necessarily helpful either; Engage would have to
// return another value you can use for submitting (which would make it not
// implement Func; undesired), or you'd have to build the supervisor with a
// callback that gets the submission interface (which inevitably makes a mess
// because naturally you want that submission interface in your controlling
// actor, which means your callback is now just a setter and it races oh why),
// or have another method that gets the submitter but this actually fixes nothing.
