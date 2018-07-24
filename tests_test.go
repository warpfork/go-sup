package sup

import (
	"context"
	"fmt"
	"testing"

	//	. "github.com/warpfork/go-wish"
)

func TestPanicCalming(t *testing.T) {
	err := superviseStream{name: "groupname"}.init(TaskGenFromTasks(TaskFromFunc(func(_ context.Context) error {
		panic(fmt.Errorf("foo"))
	}))).Run(context.Background())
	//Wish(t, err, ShouldEqual, &ErrChild{fmt.Errorf("foo"), true})
	t.Logf("%v", err)
}
