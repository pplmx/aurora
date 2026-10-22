package aurora

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readRepoFile resolves a checkout file relative to this test package.
func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(parts...))
	require.NoError(t, err)
	return string(data)
}

// TestGoModVersionIsPinnedEverywhere guards the three-way contract between
// go.mod, the CI runners and the release image. These drifted apart before:
// go.mod said 1.26 while CI and Docker installed different versions, and the
// Docker build failed for reasons nobody could reproduce locally. A plain
// `go test ./...` now catches any one of them moving alone.
func TestGoModVersionIsPinnedEverywhere(t *testing.T) {
	goMod := readRepoFile(t, "go.mod")
	m := regexp.MustCompile(`(?m)^go (\d+\.\d+(\.\d+)?)$`).FindStringSubmatch(goMod)
	require.NotNil(t, m, "go.mod must declare a pinned go directive")
	goVersion := m[1]
	majorMinor := strings.Join(strings.Split(goVersion, ".")[:2], ".")
	assert.Equal(t, "1.27.2", goVersion,
		"this assertion pins the release toolchain; update it together with go.mod")

	for _, name := range []string{"ci.yml", "build.yml", "release.yml"} {
		workflow := readRepoFile(t, ".github", "workflows", name)
		assert.Contains(t, workflow, "GO_VERSION: '"+goVersion+"'",
			"%s must install the go.mod toolchain", name)
	}

	dockerfile := readRepoFile(t, "Dockerfile")
	assert.Contains(t, dockerfile, "FROM golang:"+majorMinor+"-alpine",
		"the Docker builder must track go.mod's Go major.minor")
}

// TestDockerBuildRequiresCgo locks the fix for the red Docker job: the image
// built with CGO_ENABLED=0, which does not compile — the nocgo build of
// mattn/go-sqlite3 is a stub without sqlite3.Error. If someone "simplifies"
// the Dockerfile back to CGO_ENABLED=0, this test fails before CI does.
func TestDockerBuildRequiresCgo(t *testing.T) {
	dockerfile := readRepoFile(t, "Dockerfile")
	assert.NotRegexp(t, `(?m)^ENV CGO_ENABLED=0`, dockerfile,
		"the sqlite driver needs cgo; CGO_ENABLED=0 does not compile")
	assert.Contains(t, dockerfile, "CGO_ENABLED=1")
	assert.Contains(t, dockerfile, "musl-dev",
		"the static cgo build needs musl's gcc in the alpine builder")
}

// TestGolangciLintVersionIsPinned pins the workflow's golangci-lint binary.
// `version: latest` jumped to the v2 line and rejected the config schema the
// repo still used, which is how lint started failing with "additional
// properties 'exclude-rules' not allowed" without any local change.
func TestGolangciLintVersionIsPinned(t *testing.T) {
	ci := readRepoFile(t, ".github", "workflows", "ci.yml")
	m := regexp.MustCompile(`version: (v\S+)`).FindStringSubmatch(ci)
	require.NotNil(t, m, "ci.yml must pin the golangci-lint version, not use latest")
	assert.NotEqual(t, "latest", m[1])
}
