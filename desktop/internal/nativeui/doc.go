// Package nativeui implements the desktop application's interface with MyGo's
// own widgets instead of a web page.
//
// The previous shell hosted the same HTML UI the REST gateway serves, which
// meant the desktop app depended on web/ and on a browser engine to show six
// pages of forms. A view here is a function of application state that MyGo
// draws directly: no HTML, no JavaScript, no Content-Security-Policy, and
// nothing that can fail silently the way an Alpine directive did when the CSP
// forbade its evaluator.
//
// Every view reads and writes the same application services the REST layer
// uses through the adapter package, so a record created in the window is the
// record the CLI prints and the gateway returns.
package nativeui
