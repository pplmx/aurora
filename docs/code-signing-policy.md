# Aurora code signing policy

Aurora publishes release artifacts only from the public
[`pplmx/aurora`](https://github.com/pplmx/aurora) repository. The release
workflow builds on the native operating system, records the source tag and
commit, and verifies the resulting version and build time before publishing.

## SignPath Foundation

**Free code signing provided by SignPath.io, certificate by SignPath Foundation.**

This statement applies only after SignPath accepts Aurora into its open-source
program. Until that approval is recorded in this repository, Aurora does not
claim that any Windows artifact is signed by SignPath.

SignPath signing is limited to Windows binaries and installers produced by
Aurora's GitHub Actions release workflow. Every signing request requires a
manual approval; local builds and arbitrary uploads are never signed. The
signed files must come from the public repository and use the product name
`Aurora` and the release version as their file metadata.

### Team roles

- **Committers and reviewers:** [@pplmx](https://github.com/pplmx). Pull
  requests from outside contributors are reviewed by the maintainer.
- **Approvers:** [@pplmx](https://github.com/pplmx). The approver decides
  whether a tagged build may be submitted for signing.

## macOS

SignPath Foundation is not an Apple Developer account and cannot supply an
Apple Developer ID identity. macOS release signing and notarization therefore
remain gated on an Apple Developer Program membership controlled by the
maintainer. The exact CI secret contract is documented in
[desktop.md](desktop.md#macos-发布签名).

## Privacy

The desktop application does not collect telemetry. See the
[privacy statement](privacy.md) for the data and network-access boundary.

## Verification

Release users can verify the published SHA-256 values in `checksums.txt`.
Windows releases must also pass the workflow's `Get-AuthenticodeSignature`
checks, including the expected signer and an RFC 3161 timestamp. macOS releases
must pass `codesign`, Gatekeeper, and `stapler validate` before publication.
