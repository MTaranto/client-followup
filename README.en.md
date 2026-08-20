# Client Follow-up

[Português](README.md) | **English**

Local application for tracking client follow-ups, deadlines, history and next actions. The project was developed for a real operational workflow, with a focus on simplicity, clarity and reliability.

It runs locally on Linux, with data persisted in SQLite and a web interface served by the application itself.

## Main features

- client creation and editing;
- creation, editing and deletion of pending follow-ups;
- incremental client search with case- and accent-insensitive matching;
- explicit handling of clients with the same name;
- controlled confirmation flows for phone changes and duplicate phone numbers;
- start date, due date, priority, description, forwarding destination and notes;
- operational dashboard with indicators and deadline alerts;
- lifecycle `PENDING → COMPLETED → PENDING` and `COMPLETED → ARCHIVED`;
- individual client view with operational history;
- record search and filtering by date range, client, forwarding destination, priority and status;
- browser printing for PDF export;
- responsive interface for desktop, tablet and mobile;
- automatic local backups with rotating SQLite recovery points.

Archived follow-ups leave the main operational workflow but remain available for historical consultation.

## Technologies

- **Go** — HTTP server, business rules and data access;
- **SQLite** — local persistence;
- **HTMX** — dynamic interactions without a frontend framework;
- **HTML templates** — server-side rendering;
- **JavaScript** — used only where interface behavior requires it;
- **CSS** — responsive layout and presentation;
- **GitHub Actions** — continuous project validation.

HTMX is stored locally in the project, so the application does not depend on a CDN during normal use.

## Architecture

```text
Browser
   │
   ▼
Go application
   │
   ├── HTML templates
   ├── HTMX
   └── minimal JavaScript
   │
   ▼
SQLite
```

The application listens locally on `127.0.0.1`, keeping the workflow and data under the user's own control.

## Linux distribution

The validated distribution targets **Linux x86-64 / amd64** and is built as the compiled `client-followup-linux-amd64.tar.gz` bundle. End users do not need Go or the `sqlite3` CLI to run the application.

Expected runtime dependencies:

- `systemctl` with user-service support;
- `curl`;
- `xdg-open`.

### Installation

For a version published through GitHub Releases, download and extract the bundle, then run the installer from the extracted directory:

```bash
tar -xzf client-followup-linux-amd64.tar.gz
cd client-followup-linux-amd64
./install.sh
```

Installation is entirely user-level and **does not use `sudo`**. The application is added to the application menu as **Client Follow-up**.

The `systemd --user` service starts on demand when the application is opened. It is not configured for session autostart.

Application files are installed under:

```text
~/.local/share/client-followup/app/
```

The database is stored separately from the replaceable application files:

```text
~/.local/share/client-followup/data/client-followup.db
```

Backups and recovery points are stored under:

```text
~/.local/share/client-followup/backups/
```

## Backup and recovery

The mechanism uses complete SQLite snapshots, not incremental backups.

The normal recovery window keeps storage bounded:

- **1 daily baseline** — `client-followup-YYYY-MM-DD.db`, representing the state at the first application start of the day;
- **up to 3 rotating recovery snapshots** — `recent-1.db`, `recent-2.db` and `recent-3.db`, preserving states from before the latest persisted changes;
- **1 pre-restore protection** — `pre-restore.db`, created during a restore to preserve the database that was active immediately before restoration.

`recent-1` is the state immediately before the latest persisted change, followed by `recent-2` and `recent-3`.

### Restoring a recovery point

To display usage and the currently available recovery points:

```bash
~/.local/share/client-followup/restore-backup.sh
```

Restore examples:

```bash
~/.local/share/client-followup/restore-backup.sh recent-1
~/.local/share/client-followup/restore-backup.sh recent-2
~/.local/share/client-followup/restore-backup.sh recent-3
~/.local/share/client-followup/restore-backup.sh daily
~/.local/share/client-followup/restore-backup.sh 2026-08-20
```

The script validates the selected point, stops the service, preserves the active database as `pre-restore.db`, restores the snapshot, removes SQLite WAL/SHM files, restarts the service and waits for the `/health` endpoint to respond.

After a successful restore, refresh the browser with `F5`. The state that existed immediately before restoration is also made available as `recent-1`, providing an immediate undo path for the restore itself.

## Uninstall

Run:

```bash
~/.local/share/client-followup/uninstall.sh
```

Uninstall removes the service, launcher, icon and executable application files, while **preserving the database and backups** under:

```text
~/.local/share/client-followup/
```

## Quality

The project includes automated tests for business rules and persistence, together with `go vet`, race-detector checks, JavaScript validation and GitHub Actions.

The main workflows have also been manually validated in the browser, including client creation, search, duplicate-name resolution, phone flows, follow-up lifecycle operations, dashboard synchronization, reports, responsive layouts and printing. The Linux distribution has also been validated end to end for user-level installation, launcher startup, data persistence, Backup & Recovery and uninstall with data preservation.

## S.C.A.L.E. Method

The project follows the **S.C.A.L.E. Method**, a methodology I developed based on proportional engineering: adopt the simplest professional solution that fully solves the validated problem, expanding controls, tests and documentation according to the actual risk.

The implementation favors small changes, explicit states, local dependencies, reproducible validation, clear rollback paths and human acceptance before integration.
