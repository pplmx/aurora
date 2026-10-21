import { execSync } from "node:child_process";

// The version comes from git, the same ref the justfile passes to `go build`
// (-X main.Version) and to `just api`. `mygo build` links this into the
// bundled app, so aurora-desktop --version, the .deb's Version field and the
// tarball's name all report the commit the binary was built from.
const version = execSync("git describe --tags --always --dirty", {
  encoding: "utf8",
}).trim();

export default {
  name: "Aurora",
  identifier: "com.pplmx.aurora",
  version,
  main: "./cmd/aurora-desktop",
  // mygo build supplies its own -ldflags, so this runs before resources are
  // copied and leaves the only build timestamp a packaged binary can read.
  buildCommand: "node scripts/stamp-build-time.mjs",
  // appicon/ holds the Go package that embeds this file for the window icon.
  // It is deliberately outside resources/: every file there is copied into
  // the shipped app, which would otherwise include Go source and tests.
  icon: "./appicon/icon.png",
  out: "../dist/desktop",
  copyright: "Aurora contributors",

  // aurora://lottery?id=<draw> opens the app on that draw. Runtime
  // registration (App.RegisterURLScheme) covers Linux and Windows without an
  // installer; this is what macOS bundles need, and what makes an installed
  // Linux desktop entry the default handler for aurora:// links.
  urlSchemes: ["aurora"],

  linux: {
    maintainer: "Aurora contributors <noreply@pplmx.dev>",
    comment: "Blockchain toolkit: provably fair lotteries, voting, tokens, NFTs and oracles",
    categories: ["Utility"],
  },
};
