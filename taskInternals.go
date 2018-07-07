package sup

import (
	"fmt"
)

// boundTask is the internal implementation of tasks.  Any Task interface
// supplied to a supervisor is converted into a boundTask immediately,
// and properties like name will be determined at this time and kept in
// the boundTask struct (so that we have no question as to their immutability).
//
// boundTask should always be seen as a pointer.  We use the uniqueness of the
// address as a key for many internal bookkeeping operations.
type boundTask struct {
	original Task
	name     string
}

func bindTask(original Task) *boundTask {
	t := &boundTask{original: original}

	// Sample or generate a name.
	switch o2 := original.(type) {
	case NamedTask:
		t.name = o2.Name()
	default:
		t.name = fmt.Sprintf("%p", t)
	}

	return t
}

func bindTasks(original []Task) []*boundTask {
	v := make([]*boundTask, len(original))
	for i, o := range original {
		v[i] = bindTask(o)
	}
	return v
}
