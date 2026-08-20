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

The validated distribution targets **Linux x86-64 / amd64** and is provided as the compiled package:

```text
client-followup-linux-amd64.tar.gz
```

End users do not need Go or the `sqlite3` CLI to run the application.

Expected runtime dependencies:

- `systemctl` with user-service support;
- `curl`;
- `xdg-open`.

### Download

The distributable version is available from the repository's **Releases** section:

https://github.com/MTaranto/client-followup/releases

For the first stable distribution, use release:

```text
v1.0.0
```

Download:

```text
client-followup-linux-amd64.tar.gz
```

### Installation

After downloading the file, open a terminal in the directory where it was saved and run:

```bash
tar -xzf client-followup-linux-amd64.tar.gz
cd client-followup-linux-amd64
./install.sh
```

Installation is entirely user-level and **does not use `sudo`**.

After installation, the application is available as **Client Follow-up** in the application menu.

The `systemd --user` service starts on demand when the application is opened. It is not configured for session autostart.

Executable application files are installed under:

```text
~/.local/share/client-followup/app/
```

The database is stored separately from replaceable application files:

```text
~/.local/share/client-followup/data/client-followup.db
```

Backups and recovery points are stored under:

```text
~/.local/share/client-followup/backups/
```

## Backup and recovery

The mechanism uses **complete SQLite snapshots**, not incremental backups.

The normal recovery window keeps storage bounded:

- **1 daily baseline** — `client-followup-YYYY-MM-DD.db`, representing the state at the first application start of the day;
- **up to 3 rotating recovery snapshots** — `recent-1.db`, `recent-2.db` and `recent-3.db`, preserving states from before the latest persisted changes;
- **1 pre-restore protection** — `pre-restore.db`, created during a restore to preserve the database that was active immediately before restoration.

`recent-1` represents the state immediately before the latest persisted change, followed by `recent-2` and `recent-3`.

### Display available recovery points

Run:

```bash
~/.local/share/client-followup/restore-backup.sh
```

The command displays usage information and the currently available recovery points.

### Restore a recovery point

Examples:

```bash
~/.local/share/client-followup/restore-backup.sh recent-1
~/.local/share/client-followup/restore-backup.sh recent-2
~/.local/share/client-followup/restore-backup.sh recent-3
~/.local/share/client-followup/restore-backup.sh daily
~/.local/share/client-followup/restore-backup.sh 2026-08-20
```

The script:

1. validates the selected recovery point;
2. stops the service;
3. preserves the active database as `pre-restore.db`;
4. restores the selected snapshot;
5. removes SQLite WAL/SHM files;
6. restarts the service;
7. waits for the `/health` endpoint to respond.

After a successful restore, refresh the browser with `F5`.

The state that was active immediately before restoration is also made available as `recent-1`, providing an immediate way to undo the restore itself.

## Uninstall

Run:

```bash
~/.local/share/client-followup/uninstall.sh
```

Uninstall removes:

- the `systemd --user` service;
- application-menu launcher;
- application icon;
- executable application files.

The database and backups are **preserved** under:

```text
~/.local/share/client-followup/
```

A future installation can therefore reuse the existing data.

## Quality

The project includes automated tests for business rules and persistence, together with `go vet`, race-detector checks, JavaScript validation and GitHub Actions.

The main workflows have also been manually validated in the browser, including client creation, search, duplicate-name resolution, phone flows, follow-up lifecycle operations, dashboard synchronization, reports, responsive layouts and printing.

The Linux distribution has also been validated for:

- Linux amd64 bundle build;
- user-level installation without `sudo`;
- startup from the application menu;
- on-demand service startup;
- data persistence;
- recovery-point rotation;
- restore;
- restore undo;
- reinstall while preserving data;
- uninstall while preserving the database and backups.

## S.C.A.L.E. Method

The project follows the **S.C.A.L.E. Method**, a methodology I developed based on proportional engineering: adopt the simplest professional solution that fully solves the validated problem, expanding controls, tests and documentation according to the actual risk.

The implementation favors small changes, explicit states, local dependencies, reproducible validation, clear rollback paths and human acceptance before integration.
