package tray

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrayShowsTitle(t *testing.T) {
	assert.False(t, trayShowsTitle("darwin", true), "macOS draws the title beside the icon")
	assert.True(t, trayShowsTitle("darwin", false), "without an icon the title is all the item shows")
	for _, goos := range []string{"linux", "windows"} {
		assert.True(t, trayShowsTitle(goos, true), goos)
		assert.True(t, trayShowsTitle(goos, false), goos)
	}
}
