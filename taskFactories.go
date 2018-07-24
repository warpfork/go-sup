package sup

import (
	"context"
	"reflect"
)

func TasksFromSlice(
	theSlice interface{},
	taskFn func(context.Context, interface{}) error,
) []Task {
	panic("not yet implemented")
}

func TasksFromMap(
	theMap interface{},
	taskFn func(ctx context.Context, k, v interface{}) error,
) []Task {
	theMap_rv := reflect.ValueOf(theMap)
	if theMap_rv.Kind() != reflect.Map {
		panic("usage")
	}
	keys_rv := theMap_rv.MapKeys()
	tasks := make([]Task, len(keys_rv))
	for i, k_rv := range keys_rv {
		tasks[i] = mapEntryTask{
			k_rv.Interface(),
			theMap_rv.MapIndex(k_rv).Interface(),
			taskFn,
		}
	}
	return tasks
}

type mapEntryTask struct {
	k  interface{}
	v  interface{}
	fn func(ctx context.Context, k, v interface{}) error
}

func (t mapEntryTask) Run(ctx context.Context) error {
	return t.fn(ctx, t.k, t.v)
}

// TaskGen is a channel which yields tasks until closed.
// A TaskGen channel is used to feed work into a supervisor that runs
// unbounded numbers of tasks (it's not useful for SupervisorForkJoin,
// for example, because all the tasks are known up front in that scenario).
type TaskGen <-chan Task

func TaskGenFromChannel(
	theChan interface{},
	taskFn func(context.Context, interface{}) error,
) TaskGen {
	panic("not yet implemented")
}

func TaskGenFromTasks(tasks []Task) TaskGen {
	ch := make(chan Task, len(tasks))
	for _, t := range tasks {
		ch <- t
	}
	close(ch)
	return ch
}
