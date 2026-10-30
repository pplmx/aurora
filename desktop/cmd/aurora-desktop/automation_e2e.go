//go:build e2e

package main

// Importing MyGo's automation plugin makes a debug build listen on the Unix
// socket named by MYGO_AUTOMATION: another process can then read the window's
// accessibility tree, act on it and capture it as a PNG. The e2e build tag
// keeps the listener out of every release build, and a build without the tag
// behaves exactly as before.
import _ "github.com/egoist/mygo/plugins/automation"
