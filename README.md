go-sup
======

Supervisors, thread-pools, and other concurrency goodies for golang!


What go-sup is
--------------

Go-sup aims to be a bunch of "just enough" additional helpful features around concurrency to be useful.

The addition of `context.Context` to the golang standard library solved many problems in a nicely standardized way;
we aim to simply extend that a little further.

Go-sup gives you easy ways to work with _trees_ of `Context`.
You package your code into "tasks" and give it to a "supervisor";
the "supervisor" handles creating trees of `Context` and calling your tasks,
and in return that integration, it also automatically handles cancellation whenever errors start being returned up the tree.

Go-sup _can_ offer you thread pools, but it's optional.
You can also still use the `go` keyword yourself.
(In fact, for most common usages, we recommend you _do_ still use the `go` keyword yourself.)

Go-sup _can_ offer you some handy common concurrency patterns like a `Promise` type,
as well as some variants of `chan` which are automatically aware of the supervision context
and will therefore always respond to quit signal sensibly (without you needing to write the boilerplate for it).
You can also still use `select` and bare `chan` yourself however you please.

Go-sup _hopes_ the conventions it offers will help you write more code correctly the first time,
and that having some additional standard structure beyond bare `Context` will also result in code that is easier to read and review.
But ultimately, it still trusts you to write correct and sensible code.



What go-sup is not
------------------

Go-sup is not an over-bearing actor framework with strong opinions.
It's not going to require you to use actor patterns.
It's *certainly* not going to try to hew you into having just one "mailbox"
or any other academic straightjackets.

(We do have a guide to how to build actor-like patterns in go-sup, though --
see the "[Actor Conventions](./RECOMMENDATIONS.md#actor-conventions)" heading
in the [RECOMMENDATIONS](./RECOMMENDATIONS.md) doc.)

Go-sup does not have a single "system" orchestrator at all.
There is no "registering" and no globals.
Control flow is explicit.
Every supervisor is made by your code, while you're in control;
every time you do this, you get to pick the `Context` value that will be the basis for this tree;
every supervisor, when all its children are done, still just returns an error value and yields control flow back to you.
No globals involved.  No magic.

Go-sup is not meant to handle cross-process actors or distributed computation.
There's no serialization in go-sup.
Go-sup is purely for helping organize golang code within a single process.



API
---

### Overview

For flow control, we have just a few key types:

- `Task` -- for any function you can call.  `Task.Run` should accept a `Context` and may return an `error`.
- `Supervisor` -- create one of these and involve it in launching `Task`s, and it'll make nice sensible `Context` trees and give you good monitoring and default error propagation reactions.
- `SupervisedTask` -- get this by submitting a `Task` to a `Supervisor`; it lets you inspect the system somewhat.
- `Engine` -- these are to help managing the launch and tracking of lots of similar `Task`s, and launch things (with rate limits) and automatically set them up with a `Supervisor`.

In short: define a `Task`, `Submit(Task)` on a `Supervisor` to receive `SupervisedTask`, and you can run that `SupervisedTask` as well as await-completion/send-cancellations/monitor/etc via that same `SupervisedTask` handle.
And, you can expect the `Supervisor` to notice and help handle any errors.
And `Supervisor` itself is a `Task` so you can make trees of all these things very naturally.
(And `Engine`?  Actually, you don't have to use `Engine` at all; it's just helpful wrappers that put the others together in a standard way if you want to have job pools.)

Some information about tasks and their supervision becomes ubiqituously available via the `Context`.  For example:

- you can ask for `sup.ContextSupervisor(ctx)` to learn the nearest parent `Supervisor` (if there is one).
- you can ask for `sup.ContextTask(ctx)` to learn what `SupervisedTask` we're currently part of (if there is one).
- you can ask for `sup.ContextName(ctx)` to get a string that's a descriptive name of the current `SupervisedTask` (and its parentage in the supervision tree) -- very handy for logging!

For communication and coordination, golang channels and other synchronization primitives still work in exactly the same way as usual.

However, we also offer a few more type for communication and coordination, which though optional, you may also find useful:

- `SenderChannel` and `ReceiverChannel` -- these match the golang native `chan<-` and `<-chan`, but add a few more features.
- `sup.Select()` -- this corresponds to the golang native `select {}` clause,  but works with go-sup's types, and accepts variadic arguments.
- `Promise` -- this is a value that can be set once, and has a standard form of awaiting that occurence.

These communication and coordination types are generally comparable to golang native features,
but attempt to offer some boilerplate reduction, and some improvements in safety-of-use.
For example, the types that wrap channels all take `Context` parameters in all their methods,
and automatically abort in response Context cancellation signals -- something you almost always want to do, but can easily forget.
These types something you can optionally use to supplement uses of `chan` and features like `sync.WaitGroup`,
but don't have to replace or compete with those if you're comfortable using them.



Recommended Usage Conventions
-----------------------------

See [RECOMMENDATIONS.md](RECOMMENDATIONS.md).

Also check out the demo applications in the
[demoapp](./demoapp/) directory.



License
-------

go-sup is open source and free to use.

More concretely, it's available as either Apache2 or MIT license, at your option.

Or if you're a machine, please enjoy this structured text saying the same thing:

SPDX-License-Identifier: Apache-2.0 OR MIT