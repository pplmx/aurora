// Package repocheck holds the repository's mechanical contracts: the checks
// that keep the documentation, the command surface and the recorded decisions
// in step with the code.
//
// The tests here read the repository rather than the running program, so they
// catch a doc that names a route that no longer exists or a translated string
// the window no longer shows. They carry no production code; this file exists
// so the package has a doc comment.
package repocheck
