# Privacy statement

Aurora is a local-first desktop and command-line application. It does not
include analytics, telemetry, advertising identifiers, or a background usage
reporting service.

The desktop application stores its SQLite database, logs, backups, exports,
and recent-document metadata under the user's local data directory. These
files remain on the user's machine unless the user explicitly copies or
shares them.

**This program will not transfer any information to other networked systems
unless specifically requested by the user or the person installing or
operating it.** In particular:

- Oracle data sources contact only the URLs configured by the operator.
- The REST API and web server listen only on the configured address; clients
  connect only when the operator runs them.
- Downloading a release, opening a documentation link, or using GitHub services
  contacts those services at the user's request.
- Signing providers receive release build artifacts and provenance metadata
  only as part of the maintainer-approved release workflow.

Third-party services used to distribute or sign releases have their own
privacy policies. Aurora does not embed their tracking SDKs in the desktop
application.
