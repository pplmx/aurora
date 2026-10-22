package deskapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot walks up from the test's working directory to the checkout root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "justfile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "no checkout root above %s", dir)
		dir = parent
	}
}

// desktopGuide is the shipped desktop documentation. The release contract
// below is user-facing: the secrets tables and the packaging rules must be
// explained somewhere a reader can find, so the guard reads the page that is
// supposed to carry them instead of a heading in the front-page README.
func desktopGuide(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "desktop.md"))
	require.NoError(t, err, "docs/desktop.md must document the desktop release contract")
	return string(data)
}

// TestCIWiresTheDesktopModule is the guard for the gap this module lived in:
// `go test ./...` at the root never enters a nested module, so a broken
// window layer passed every gate silently. The fix is a `desktop` job in
// ci.yml; this test fails if that job disappears or stops running the same
// commands as `just desktop-check`, which is what a contributor runs locally.
func TestCIWiresTheDesktopModule(t *testing.T) {
	root := repoRoot(t)

	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	require.NoError(t, err)
	workflow := string(ci)

	require.Contains(t, workflow, "  desktop:", "ci.yml must define the desktop job")
	require.Contains(t, workflow, "go-version-file: desktop/go.mod",
		"the desktop job needs MyGo's Go version, not the core module's")
	require.Contains(t, workflow, "working-directory: desktop")

	for _, step := range []string{"gofmt -l .", "go build ./...", "go vet ./...", "go test -race ./..."} {
		assert.Contains(t, workflow, step,
			"the desktop job must run %q, like `just desktop-check`", step)
	}

	recipe, err := os.ReadFile(filepath.Join(root, "justfile"))
	require.NoError(t, err)
	local := string(recipe)

	require.Contains(t, local, "desktop-check:", "the local entry point must exist")
	for _, step := range []string{"gofmt -l .", "go build ./...", "go vet ./...", "go test -race ./..."} {
		assert.Contains(t, local, step,
			"`just desktop-check` must run %q, like the CI job", step)
	}
}

// TestRootGateCoversTheNestedModule pins the reason the job exists, so the
// explanation in ci.yml and the justfile does not become folklore: the root
// recipes really do skip desktop/, and it really is a separate module.
func TestRootGateCoversTheNestedModule(t *testing.T) {
	root := repoRoot(t)

	recipe, err := os.ReadFile(filepath.Join(root, "justfile"))
	require.NoError(t, err)
	assert.NotContains(t, string(recipe), "go test ./desktop/",
		"a root-relative path cannot cross into a nested module; use cd desktop")

	gomod, err := os.ReadFile(filepath.Join(root, "desktop", "go.mod"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(gomod), "module github.com/pplmx/aurora/desktop\n"),
		"desktop/ must stay its own module, which is why the root gate cannot see it")
}

// TestDesktopIsPackagedOnItsOwnPlatform pins why the CI job builds instead of
// packaging, and why `just desktop-package` is not run with a foreign
// -platform: the app's storage is cgo, so a cross build fails in runtime/cgo
// long before our own code. If storage ever becomes pure Go this test will
// say so, and the packaging matrix can start cross-building.
func TestDesktopIsPackagedOnItsOwnPlatform(t *testing.T) {
	root := repoRoot(t)

	recipe, err := os.ReadFile(filepath.Join(root, "justfile"))
	require.NoError(t, err)
	assert.Contains(t, string(recipe), "CGO_ENABLED=1 go tool mygo build",
		"MyGo defaults CGO_ENABLED to 0, which cannot build the cgo storage")

	// The reason: the core's database driver is cgo-only, and the desktop
	// module imports the core.
	for _, file := range []string{
		filepath.Join("internal", "infra", "sqlite", "lottery.go"),
		filepath.Join("internal", "domain", "blockchain", "init.go"),
	} {
		src, err := os.ReadFile(filepath.Join(root, file))
		require.NoError(t, err)
		assert.Contains(t, string(src), `"github.com/mattn/go-sqlite3"`,
			"%s is expected to be the cgo SQLite dependency that blocks cross builds", file)
	}
}

// TestCIPackagesTheDesktopApp pins the packaging gate. The desktop job only
// compiles the module; without this job a bad mygo.config.ts, a missing
// desktop entry, or a package that silently drops the .deb/tarball would
// reach a release tag. The commands are deliberately the same as
// `just desktop-package` (CGO_ENABLED=1 + go tool mygo build).
func TestCIPackagesTheDesktopApp(t *testing.T) {
	root := repoRoot(t)

	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	require.NoError(t, err)
	workflow := string(ci)

	require.Contains(t, workflow, "  desktop-package:", "ci.yml must package the desktop app")
	require.Contains(t, workflow, "fetch-depth: 0",
		"the packaging gate must fetch tags for git describe")
	require.Contains(t, workflow, "go-version-file: desktop/go.mod")
	require.Contains(t, workflow, "node-version: '22'")
	require.Contains(t, workflow, "CGO_ENABLED: '1'")
	require.Contains(t, workflow, "CC: gcc")
	require.Contains(t, workflow, "go tool mygo build -platform linux/amd64")
	require.Contains(t, workflow, "libwebkit2gtk-4.1-0 libgtk-3-0 xvfb",
		"the package smoke test needs the runtime webview and a display")
	require.Contains(t, workflow, "name: Smoke-test the packaged app")
	require.Contains(t, workflow, "aurora.db")
	require.Contains(t, workflow, `kill -0 "$pid"`,
		"the app must still be alive after initializing its database")
	require.Contains(t, workflow, "aurora --version")
	require.Contains(t, workflow, `!= "0.0.1"`)
	require.Contains(t, workflow, `s/^Build Time: //p`)
	require.Contains(t, workflow, `!= "unknown"`)
	for _, artifact := range []string{"aurora", "aurora.desktop", "install.sh", "*.deb", "*.tar.gz"} {
		assert.Contains(t, workflow, artifact, "the packaging gate must verify %q", artifact)
	}
	assert.Contains(t, workflow, "build-time.txt",
		"the packaging gate must verify the packaged build timestamp")
	assert.Contains(t, workflow, "MimeType=x-scheme-handler/aurora;")
	assert.Contains(t, workflow, "actions/upload-artifact@v7")
}

// TestReleasePackagesEveryDesktopPlatform pins the release matrix. A release
// that only uploads the CLI leaves the MyGo app unavailable even though the
// repository can build it locally; each target needs its own native runner
// because the SQLite dependency is cgo.
func TestReleasePackagesEveryDesktopPlatform(t *testing.T) {
	root := repoRoot(t)

	release, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	require.NoError(t, err)
	releaseWorkflow := string(release)
	require.Contains(t, releaseWorkflow, "uses: ./.github/workflows/desktop-build.yml")
	require.Contains(t, releaseWorkflow, "needs: [build, desktop-build]")
	require.NotContains(t, releaseWorkflow, "release:\n    types: [published]",
		"a release event would rebuild the tag after its own run creates the release")
	require.Contains(t, releaseWorkflow, "workflow_dispatch:")
	require.Contains(t, releaseWorkflow, "tag_name: ${{ inputs.tag || github.ref_name }}")
	require.Contains(t, releaseWorkflow, "overwrite_files: true",
		"a manual republish must replace files from an interrupted run")
	require.Contains(t, releaseWorkflow, "permissions:\n  contents: write")
	assert.NotContains(t, releaseWorkflow, "files: artifacts/**/*",
		"the release must list installable artifacts, not every file in the .app bundle")
	for _, releaseFile := range []string{
		"artifacts/aurora-linux-amd64/aurora-linux-amd64",
		"artifacts/aurora-desktop-darwin-amd64/*.dmg",
		"artifacts/aurora-desktop-linux-amd64/*.deb",
		"artifacts/aurora-desktop-linux-amd64/*.tar.gz",
		"artifacts/aurora-desktop-linux-amd64/install.sh",
		"artifacts/aurora-desktop-windows-amd64/*Setup*.exe",
	} {
		assert.Contains(t, releaseWorkflow, releaseFile,
			"the release must publish %s", releaseFile)
	}

	desktop, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "desktop-build.yml"))
	require.NoError(t, err)
	desktopWorkflow := string(desktop)
	require.Contains(t, desktopWorkflow, "CGO_ENABLED: '1'")
	require.Contains(t, desktopWorkflow, "fetch-depth: 0",
		"the release package must fetch tags for git describe")
	require.Contains(t, desktopWorkflow, "Package (Windows)")
	require.Contains(t, desktopWorkflow, "CC: gcc")
	require.Contains(t, desktopWorkflow, "go tool mygo build")
	require.Contains(t, desktopWorkflow, "go-version-file: desktop/go.mod")
	for _, platform := range []string{
		"linux/amd64",
		"linux/arm64",
		"darwin/arm64",
		"darwin/amd64",
		"windows/amd64",
	} {
		assert.Contains(t, desktopWorkflow, platform, "release matrix must package %s", platform)
	}
	for _, osName := range []string{"ubuntu-latest", "ubuntu-24.04-arm", "macos-latest", "macos-15-intel", "windows-latest"} {
		assert.Contains(t, desktopWorkflow, osName, "release matrix must use a native %s runner", osName)
	}
	assert.Contains(t, desktopWorkflow, "choco install mingw nsis")
	assert.Contains(t, desktopWorkflow, "C:\\ProgramData\\mingw64\\mingw64\\bin")
	assert.Contains(t, desktopWorkflow, "Verify package")
	assert.Contains(t, desktopWorkflow, `-name '*.deb'`)
	assert.Contains(t, desktopWorkflow, `-name '*.dmg'`)
	assert.Contains(t, desktopWorkflow, `-name '*Setup*.exe'`)
	assert.Contains(t, desktopWorkflow, "Contents/Resources/build-time.txt",
		"the macOS package must carry the build stamp in its bundle resources")
	assert.Contains(t, desktopWorkflow, `"${{ matrix.path }}/build-time.txt"`,
		"Linux and Windows packages must carry the build stamp next to the executable")

	// Every matrix leg used to stop at "the files exist". A package can pass
	// that and still die before its window appears (a missing WebKitGTK
	// runtime, a broken embedded UI), so the Linux legs - the only ones a
	// GitHub runner can give a display - must start the real artifact.
	for _, step := range []string{
		"libwebkit2gtk-4.1-0 libgtk-3-0 xvfb",
		"name: Smoke-test the packaged app",
		`if: runner.os == 'Linux'`,
		`app="${{ matrix.path }}/aurora"`,
		`"$app" --version`,
		`!= "0.0.1"`,
		`!= "unknown"`,
		`xvfb-run -a "$app" --data-dir "$data"`,
		`test -f "$data/aurora.db"`,
		`kill -0 "$pid"`,
	} {
		assert.Contains(t, desktopWorkflow, step,
			"the release package gate must include %q", step)
	}
	for _, target := range []string{"linux/amd64", "linux/arm64"} {
		assert.Contains(t, desktopWorkflow, target,
			"the startup smoke test must cover %s", target)
	}
	assert.Contains(t, releaseWorkflow, "artifacts/aurora-desktop-windows-amd64/Aurora.exe")
}

// TestReleaseRequiresSignedAndNotarizedMacOSPackages pins the macOS release
// contract: an ad-hoc-signed DMG runs only on the runner that built it, so a
// tag release must fail before packaging when the Developer ID and notary
// credentials are absent or incomplete. The workflow imports the certificate
// into a temporary keychain, and the package gate checks the resulting app
// with Gatekeeper rather than trusting that `mygo build` signed it.
func TestReleaseRequiresSignedAndNotarizedMacOSPackages(t *testing.T) {
	root := repoRoot(t)

	desktop, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "desktop-build.yml"))
	require.NoError(t, err)
	desktopWorkflow := string(desktop)

	require.Contains(t, desktopWorkflow, "require_signed_macos:")
	require.Contains(t, desktopWorkflow, "default: false")
	for _, secret := range []string{
		"MACOS_CERTIFICATE",
		"MACOS_CERTIFICATE_PASSWORD",
		"MACOS_SIGNING_IDENTITY",
		"APPLE_ID",
		"APPLE_TEAM_ID",
		"APPLE_APP_PASSWORD",
	} {
		assert.Contains(t, desktopWorkflow, secret,
			"the reusable workflow must declare the %s secret", secret)
	}
	for _, step := range []string{
		"Configure macOS signing",
		"security create-keychain",
		"security import",
		`grep -Fq "\"$MACOS_SIGNING_IDENTITY\"" <<<"$identity_out"`,
		`xcrun notarytool store-credentials "$notary_profile"`,
		"MACOS_SIGNING=enabled",
		"This release requires signed and notarized macOS packages",
		"macOS signing secrets are partially configured",
		"codesign --verify --deep --strict",
		"spctl --assess --type execute",
		"xcrun stapler validate",
	} {
		assert.Contains(t, desktopWorkflow, step,
			"the macOS release contract must include %q", step)
	}

	release, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	require.NoError(t, err)
	releaseWorkflow := string(release)
	require.Contains(t, releaseWorkflow, "secrets: inherit",
		"the release caller must pass the signing secrets to the reusable workflow")
	require.Contains(t, releaseWorkflow, "require_signed_macos: true",
		"a tag release must not publish an ad-hoc-signed macOS package")

	config, err := os.ReadFile(filepath.Join(root, "desktop", "mygo.config.ts"))
	require.NoError(t, err)
	configText := string(config)
	for _, env := range []string{
		"process.env.MACOS_SIGNING_IDENTITY",
		"process.env.NOTARY_KEYCHAIN_PROFILE",
		"process.env.NOTARY_KEYCHAIN",
	} {
		assert.Contains(t, configText, env,
			"the MyGo config must pass %s through to signing", env)
	}
	assert.Contains(t, configText, `signingIdentity: signingIdentity || "-"`,
		"local packages without credentials must remain ad-hoc signed")
	assert.Contains(t, configText, `...(signingIdentity && notaryProfile`,
		"notarization must be enabled only with a real signing identity")

	guideText := desktopGuide(t)
	require.Contains(t, guideText, "macOS 发布签名")
	require.Contains(t, guideText, "`macos.signingIdentity`")
	require.Contains(t, guideText, "`macos.notarize.keychainProfile`")
	for _, secret := range []string{
		"MACOS_CERTIFICATE",
		"MACOS_CERTIFICATE_PASSWORD",
		"MACOS_SIGNING_IDENTITY",
		"APPLE_ID",
		"APPLE_TEAM_ID",
		"APPLE_APP_PASSWORD",
	} {
		assert.Contains(t, guideText, secret,
			"docs/desktop.md must document the exact %s secret", secret)
	}
}

// TestReleaseRequiresAuthenticodeSignedWindowsPackages pins the Windows
// counterpart of the macOS contract. An unsigned installer makes SmartScreen
// warn on every download and gives users no publisher to verify, so a tag
// release must fail before packaging when the certificate is absent or
// partial, and the package gate must validate the actual Authenticode
// signatures rather than trust that `mygo build` signed them.
func TestReleaseRequiresAuthenticodeSignedWindowsPackages(t *testing.T) {
	root := repoRoot(t)

	desktop, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "desktop-build.yml"))
	require.NoError(t, err)
	desktopWorkflow := string(desktop)
	signingAction, err := os.ReadFile(filepath.Join(root, ".github", "actions", "windows-signing", "action.yml"))
	require.NoError(t, err)
	signingContract := desktopWorkflow + string(signingAction)

	require.Contains(t, desktopWorkflow, "require_signed_windows:")
	for _, secret := range []string{
		"WINDOWS_CERTIFICATE",
		"WINDOWS_CERTIFICATE_PASSWORD",
	} {
		assert.Contains(t, signingContract, secret,
			"the reusable workflow must declare the %s secret", secret)
	}
	require.Contains(t, desktopWorkflow, "uses: ./.github/actions/windows-signing",
		"desktop packaging must use the shared signing setup")
	for _, step := range []string{
		"$env:REQUIRE_SIGNED_WINDOWS",
		"WINDOWS_CERTIFICATE_FILE",
		"MYGO_WINDOWS_CERTIFICATE_PASSWORD",
		"Convert]::FromBase64String",
		"$certificate.HasPrivateKey",
		"WINDOWS_SIGNING=enabled",
		"Windows signing is not configured; the artifact is unsigned",
		"This release requires Authenticode-signed Windows packages",
		"Windows signing secrets are partially configured",
		"signtool.exe was not found",
		"Verify Windows signatures",
		"Get-AuthenticodeSignature",
		`$signature.Status -ne "Valid"`,
		"$signature.SignerCertificate.Subject",
		"$signature.SignerCertificate.Thumbprint",
		"$signature.TimeStamperCertificate",
	} {
		assert.Contains(t, signingContract, step,
			"the Windows release contract must include %q", step)
	}
	assert.Contains(t, desktopWorkflow, `Get-ChildItem "${{ matrix.path }}" -File -Filter "*Setup*.exe"`,
		"every Windows installer must be signature-checked")
	require.Contains(t, desktopWorkflow,
		`if ($env:REQUIRE_SIGNED_WINDOWS -eq "true" -and $env:WINDOWS_SIGNING -ne "enabled") {`,
		"a release must fail before the development-only signature skip")

	release, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	require.NoError(t, err)
	releaseWorkflow := string(release)
	require.Contains(t, releaseWorkflow, "secrets: inherit",
		"the release caller must pass the signing secrets to the reusable workflow")
	require.Contains(t, releaseWorkflow, "require_signed_windows: true",
		"a tag release must not publish an unsigned Windows package")

	config, err := os.ReadFile(filepath.Join(root, "desktop", "mygo.config.ts"))
	require.NoError(t, err)
	configText := string(config)
	require.Contains(t, configText, "process.env.WINDOWS_CERTIFICATE_FILE",
		"the MyGo config must pass the decoded certificate path to signing")
	require.Contains(t, configText, "windows: {",
		"the MyGo config must declare Windows signing options")
	require.Contains(t, configText, "...(windowsCertificate ? { certificate: windowsCertificate } : {})",
		"local packages without credentials must remain unsigned")

	guideText := desktopGuide(t)
	require.Contains(t, guideText, "Windows 发布签名")
	require.Contains(t, guideText, "`windows.certificate`")
	for _, secret := range []string{
		"WINDOWS_CERTIFICATE",
		"WINDOWS_CERTIFICATE_PASSWORD",
	} {
		assert.Contains(t, guideText, secret,
			"docs/desktop.md must document the exact %s secret", secret)
	}
}

// TestReleaseCLIBinariesAreNativeCgoBuilds pins the release CLI contract.
// build.yml used to cross-compile with CGO_ENABLED=0, which silently linked
// the mattn/go-sqlite3 stub: `aurora --help` worked, but every database
// command failed. The release job must also verify Go's embedded build
// metadata, sign the Windows executables, and publish checksums generated
// without splitting on spaces in macOS DMG names.
func TestReleaseCLIBinariesAreNativeCgoBuilds(t *testing.T) {
	root := repoRoot(t)

	build, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "build.yml"))
	require.NoError(t, err)
	buildWorkflow := string(build)

	action, err := os.ReadFile(filepath.Join(root, ".github", "actions", "windows-signing", "action.yml"))
	require.NoError(t, err)
	actionText := string(action)
	require.Contains(t, actionText, "shell: pwsh", "the Windows signing script must use a pinned shell")

	require.Contains(t, buildWorkflow, "uses: mlugg/setup-zig@v2.2.1", "Zig setup must use a pinned major version")
	require.Contains(t, buildWorkflow, "ZIG_VERSION: '0.14.1'", "Zig must be pinned independently of the action default")

	require.Contains(t, buildWorkflow, "runs-on: ${{ matrix.runner }}")
	for _, target := range []struct {
		goos   string
		goarch string
		runner string
	}{
		{"darwin", "amd64", "macos-15-intel"},
		{"darwin", "arm64", "macos-latest"},
		{"linux", "amd64", "ubuntu-latest"},
		{"linux", "arm64", "ubuntu-24.04-arm"},
	} {
		assert.Contains(t, buildWorkflow, "goos: "+target.goos,
			"the CLI matrix must build %s", target.goos)
		assert.Contains(t, buildWorkflow, "goarch: "+target.goarch,
			"the CLI matrix must build %s/%s", target.goos, target.goarch)
		assert.Contains(t, buildWorkflow, "runner: "+target.runner,
			"%s/%s must use its native runner", target.goos, target.goarch)
	}
	require.Contains(t, buildWorkflow, "CGO_ENABLED: 1")
	require.Contains(t, buildWorkflow, "REF: ${{ inputs.ref }}",
		"the build must derive its version from the requested tag, not the caller's branch")
	assert.Contains(t, buildWorkflow, `if [[ "$REF" == refs/tags/* ]]; then`)
	assert.Contains(t, buildWorkflow, `if ($env:REF -like "refs/tags/*") {`)
	assert.NotContains(t, buildWorkflow, "CGO_ENABLED=0",
		"the release CLI cannot link the SQLite stub")

	for _, windows := range []string{
		"goarch: amd64",
		"goarch: arm64",
		"zig_target: x86_64-windows-gnu",
		"zig_target: aarch64-windows-gnu",
		"uses: ./.github/actions/windows-signing",
		"CC: zig cc -target ${{ matrix.zig_target }}",
		"signtool @signArgs",
		"Get-AuthenticodeSignature",
		`$signature.Status -ne "Valid"`,
		"$signature.SignerCertificate.Subject",
		"$signature.SignerCertificate.Thumbprint",
		"$signature.TimeStamperCertificate",
	} {
		assert.Contains(t, buildWorkflow, windows,
			"the Windows CLI contract must include %q", windows)
	}
	assert.NotContains(t, buildWorkflow, "$args",
		"$args is a PowerShell automatic variable; use a named array")

	release, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	require.NoError(t, err)
	releaseWorkflow := string(release)
	require.Contains(t, releaseWorkflow, "name: Verify release artifacts",
		"the release job must validate native CLI artifacts before publication")
	for _, step := range []string{
		"secrets: inherit",
		"require_signed_windows: true",
		"name: Verify release artifacts",
		`grep -Fq $'\tbuild\tCGO_ENABLED=1'`,
		`grep -Fq $'\tbuild\tGOOS='`,
		`grep -Fq $'\tbuild\tGOARCH='`,
		`artifacts/aurora-linux-amd64/aurora-linux-amd64 version`,
		`grep -Fq "Version: $RELEASE_VERSION"`,
		`grep -Fq "Build Time: unknown"`,
		"name: Generate release checksums",
		"-print0",
		"sort -z",
		"xargs -0 -r sha256sum",
		"sha256sum -c checksums.txt",
		`sed -E 's#  .*/#  #' checksums.txt`,
		`! grep -Eq '  .*/' checksums.txt`,
		"artifacts/checksums.txt",
	} {
		assert.Contains(t, releaseWorkflow, step,
			"the release workflow must include %q", step)
	}
}

// TestPackagedResourcesHoldNoGoSources guards MyGo's resource contract:
// everything under desktop/resources is copied next to the executable and
// into the .deb/tarball. The icon used to live there together with its Go
// package and test, so every package carried resources.go and
// resources_test.go. Keep the directory free of build-only files, and make
// the packaging gates reject them if they return.
func TestPackagedResourcesHoldNoGoSources(t *testing.T) {
	root := repoRoot(t)

	entries, err := os.ReadDir(filepath.Join(root, "desktop", "resources"))
	if err == nil {
		for _, entry := range entries {
			assert.False(t, strings.HasSuffix(entry.Name(), ".go"),
				"desktop/resources/%s would ship as a runtime resource", entry.Name())
		}
	} else {
		require.ErrorIs(t, err, os.ErrNotExist,
			"desktop/resources may be absent, but an unreadable directory is a packaging bug")
	}

	for _, workflow := range []string{"ci.yml", "desktop-build.yml"} {
		src, err := os.ReadFile(filepath.Join(root, ".github", "workflows", workflow))
		require.NoError(t, err)
		assert.Contains(t, string(src), "-name '*.go'",
			"%s must reject Go sources in packaged artifacts", workflow)
	}

	config, err := os.ReadFile(filepath.Join(root, "desktop", "mygo.config.ts"))
	require.NoError(t, err)
	assert.Contains(t, string(config), `icon: "./appicon/icon.png"`,
		"the packaging icon must stay outside desktop/resources")
	assert.Contains(t, string(config), `buildCommand: "node scripts/stamp-build-time.mjs"`,
		"packaging must stamp the build time before resources are copied")

	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(ignore), "/desktop/resources/build-time.txt",
		"the generated build stamp is packaging state, not source")
}

// TestDockerPublishesOnlyReviewedCommits pins the Docker publishing contract:
// pull requests must be validation-only (fork PRs cannot read repository
// secrets), and a published image must be tested by the tag that was actually
// built. The workflow used to push on every event and then smoke-test the
// long commit SHA even though metadata-action emitted the short SHA.
func TestDockerPublishesOnlyReviewedCommits(t *testing.T) {
	root := repoRoot(t)

	src, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "docker.yml"))
	require.NoError(t, err)
	workflow := string(src)

	require.Contains(t, workflow, "pull_request:")
	require.NotContains(t, workflow, "release:\n    types: [published]",
		"the tag push already builds the release image; a release event duplicates it")
	require.Contains(t, workflow, "PUBLISH: ${{ github.event_name != 'pull_request' }}")
	require.Contains(t, workflow, "push: ${{ env.PUBLISH == 'true' }}")
	for _, step := range []string{
		"name: Login to Docker Hub\n        if: env.PUBLISH == 'true'",
		"name: Login to GitHub Container Registry\n        if: env.PUBLISH == 'true'",
	} {
		assert.Contains(t, workflow, step,
			"pull requests must not authenticate to an image registry")
	}
	require.Contains(t, workflow, "type=sha,format=short,prefix=",
		"the explicit short SHA tag must match the smoke-tested image")
	require.Contains(t, workflow, "type=semver,pattern={{version}}",
		"tag releases must publish a version image, not only branch and SHA tags")
	require.Contains(t, workflow, "type=semver,pattern={{major}}.{{minor}}")
	require.Contains(t, workflow, "aurora-smoke-test:${{ github.sha }}")
	require.NotContains(t, workflow, "${{ env.DOCKER_IMAGE }}:${{ github.sha }}",
		"the image is tagged with the short SHA, not the full commit SHA")
}
