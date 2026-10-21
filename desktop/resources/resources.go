// Package resources carries files the desktop app needs at build time and at
// run time.
//
// icon.png lives here because `mygo build` picks up resources/icon.png by
// convention (desktop/mygo.config.ts sets no explicit icon): it becomes the
// .icns/.ico/.desktop icon, and the .deb installs it under
// usr/share/icons/hicolor. Embedding the same file keeps one source of truth
// for the window icon of an unpackaged build (`just desktop`, `go run`),
// which otherwise shows the toolkit's placeholder.
package resources

import _ "embed"

// IconPNG is the app icon, 1024x1024 PNG with transparency.
//
//go:embed icon.png
var IconPNG []byte
