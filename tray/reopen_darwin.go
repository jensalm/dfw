//go:build darwin

package tray

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

void dfwInstallReopenHandler(void);
*/
import "C"

// the handler itself is in reopen_darwin.m: a file that exports to C may
// only declare in its preamble.

//export dfwTrayReopened
func dfwTrayReopened() {
	reopened()
}

// installReopenHandler answers the reopen Apple Event, which macOS sends to
// a running application that is opened again. it takes the place of
// NSApplication's own, which asks a delegate systray does not give one.
func installReopenHandler() {
	C.dfwInstallReopenHandler()
}
