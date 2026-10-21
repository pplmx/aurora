// Package appicon carries the desktop app's icon at build time and at run
// time.
//
// It deliberately lives outside desktop/resources: everything in that
// directory is a runtime resource that `mygo build` copies next to the
// executable and into the .deb/tarball. Keeping the Go package there shipped
// icon.go and icon_test.go with the app. mygo.config.ts names this file as
// the packaging icon; the embed below remains the window icon for an
// unpackaged build (`just desktop`, `go run`).
package appicon

import _ "embed"

// IconPNG is the app icon, 1024x1024 PNG with transparency.
//
//go:embed icon.png
var IconPNG []byte
