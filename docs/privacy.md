# Privacy statement

Aurora is a local-first desktop and command-line application. It does not
include analytics, telemetry, advertising identifiers, or a background usage
reporting service.

The desktop application stores its SQLite database, logs, and
recent-document metadata under the user's local data directory. Exports and
database backups are written wherever the user chooses. These files remain on
the user's machine unless the user explicitly copies or shares them. On
Unix-like systems the default application data directory is created (or
tightened) with mode `0700` so other local accounts cannot traverse the
database or logs; the SQLite files, WAL/SHM sidecars and log file are also
kept at mode `0600`. An explicit `--data-dir` keeps the permissions of an
existing operator-managed directory; a newly created explicit directory is
still owner-only. A new database backup set is created as an owner-only
`0700` directory whose database and metadata files are `0600`; an existing
backup directory keeps its own permissions while new files in it remain
owner-only. Restore keeps its hidden undo-snapshot directory at `0700`, its
database copy at `0600`, and promotes the restored live database with the same
owner-only file mode.

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
