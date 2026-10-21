package resources

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIconIsASquarePNG is the packaging contract: `mygo build` decodes
// resources/icon.png and renders the .icns, .ico and .desktop icons from it,
// and Window.SetIcon decodes the embedded copy. A non-PNG or a non-square
// file fails the build or draws a stretched icon, so pin both.
func TestIconIsASquarePNG(t *testing.T) {
	require.NotEmpty(t, IconPNG, "resources/icon.png must be embedded")

	img, err := png.Decode(bytes.NewReader(IconPNG))
	require.NoError(t, err, "MyGo decodes the icon as PNG")

	b := img.Bounds()
	require.Equal(t, b.Dx(), b.Dy(), "the icon must be square")
	require.GreaterOrEqual(t, b.Dx(), 512,
		"macOS renders ic10 (1024) from this file; a smaller source would be upscaled")
}
