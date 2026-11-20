package tray

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrayShowsTitle(t *testing.T) {
	assert.False(t, trayShowsTitle("darwin"), "macOS draws the title beside the icon")
	assert.True(t, trayShowsTitle("linux"))
	assert.True(t, trayShowsTitle("windows"))
}
