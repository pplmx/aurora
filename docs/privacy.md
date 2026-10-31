# Privacy statement

Aurora is a local-first desktop and command-line application. It does not
include analytics, telemetry, advertising identifiers, or a background usage
reporting service.

The desktop application stores its SQLite database, logs, recent-document
metadata, the window geometry the UI toolkit remembers, and a small
`desktop-settings.json` (the live-updates and appearance choices) under the
user's local data directory. Exports and database backups are written wherever
the user chooses. These files remain on the user's machine unless the user
explicitly copies or shares them.

On Unix-like systems every one of them is owner-only; the table below lists
each path with the mode Aurora gives it.

## Files and permissions

Everything Aurora writes stays under the user's data directory unless the
operator names another path. On Unix-like systems the defaults are owner-only:

| Path                                | Mode            | What it holds                                                                                          |
| ----------------------------------- | --------------- | ------------------------------------------------------------------------------------------------------ |
| `data directory`                    | `0700`          | Created (or tightened) so other local accounts cannot traverse in                                      |
| `aurora.db`                         | `0600`          | The SQLite database: every module's records, the hash chain, oracle sources with their request headers |
| `aurora.db-wal`, `-shm`, `-journal` | `0600`          | SQLite sidecars; `-journal` appears only outside WAL mode                                              |
| `aurora.log`                        | `0600`          | The structured log                                                                                     |
| `desktop-settings.json`             | `0600`          | Live-updates and appearance choices                                                                    |
| `backup directory (new)`            | `0700`          | A backup set created by File → Back Up Database                                                        |
| `files inside a backup`             | `0600`          | The database copy and its metadata                                                                     |
| `restore undo snapshot`             | `0700` / `0600` | The temporary pre-restore copy; removed when the restore settles                                       |
| `window-state.json`                 | MyGo's choice   | Window geometry; written by the UI toolkit, not by Aurora                                              |

The last row is the one Aurora does not set: MyGo writes `window-state.json`
itself, and the recent-document list lives in the platform's own recent store
rather than in this directory. Both hold only what the user opened and how the
window was sized.

An explicit `--data-dir` keeps the permissions of an existing directory the
operator already manages; a directory Aurora creates there is still owner-only.
A database symlink is resolved before its sidecars are hardened, because SQLite
names the sidecars after the link target rather than the link. Disk-backed
`file:` URIs are decoded to their filesystem path; in-memory and empty-name
temporary databases are skipped.

**This program will not transfer any information to other networked systems
unless specifically requested by the user or the person installing or
operating it.** In particular:

- Oracle data sources contact only the URLs configured by the operator.
- The REST server listens only on the configured address; clients connect only
  when the operator runs it.
- Downloading a release, opening a documentation link, or using GitHub services
  contacts those services at the user's request.
- Signing providers receive release build artifacts and provenance metadata
  only as part of the maintainer-approved release workflow.

Third-party services used to distribute or sign releases have their own
privacy policies. Aurora does not embed their tracking SDKs in the desktop
application.
