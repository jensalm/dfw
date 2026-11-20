package tray

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReopenedRunsTheHandlerOffTheEventLoop(t *testing.T) {
	t.Cleanup(func() { setReopenHandler(nil) })
	release := make(chan struct{})
	called := make(chan struct{}, 2)
	setReopenHandler(func() {
		called <- struct{}{}
		<-release
	})

	// the handler blocks, yet reopened returns: the event loop goes on.
	reopened()
	reopened()
	for range 2 {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("the handler was not called")
		}
	}
	close(release)
}

func TestReopenedWithNoHandlerDoesNothing(t *testing.T) {
	setReopenHandler(nil)
	assert.NotPanics(t, reopened)
}
