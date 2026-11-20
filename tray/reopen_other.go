//go:build !darwin

package tray

// only macOS reopens a running application rather than starting another.
func installReopenHandler() {}
