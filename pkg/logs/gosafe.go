package logs

import (
	"fmt"
	"runtime/debug"
	"sync"
)

// GoSafe runs fn in a new goroutine, recovering from panics so a single
// background task cannot crash the whole process. The name is included in the
// recovery log to aid debugging.
func GoSafe(name string, fn func()) {
	go func() {
		defer recoverPanic(name)
		fn()
	}()
}

// WGGoSafe is GoSafe but registers the goroutine with a WaitGroup so callers
// can wait for it during graceful shutdown.
func WGGoSafe(wg *sync.WaitGroup, name string, fn func()) {
	wg.Go(func() {
		defer recoverPanic(name)
		fn()
	})
}

func recoverPanic(name string) {
	if r := recover(); r != nil {
		Errorf("panic in %s: %v\n%s", name, r, debug.Stack())
		_ = fmt.Sprint(r)
	}
}
