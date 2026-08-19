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
- automatic local SQLite backups.

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

## Installation

The final Linux distribution will be provided as a **compiled binary**, so end users will not need to install Go or the `sqlite3` utility to run the application.

The installation layer is currently being prepared and will be completed before final distribution. It is intended to provide:

- a Linux `.desktop` launcher with its own name and icon in the application menu and/or Desktop;
- normal application startup through the launcher;
- an optional configuration for automatic startup with the user's session.

Final installation instructions will be added here after this stage is completed and validated.

## Quality

The project includes automated tests for business rules and persistence, together with `go vet`, race-detector checks, JavaScript validation and GitHub Actions.

The main workflows have also been manually validated in the browser, including client creation, search, duplicate-name resolution, phone flows, follow-up lifecycle operations, dashboard synchronization, reports, responsive layouts and printing.

## S.C.A.L.E. Method

The project follows the **S.C.A.L.E. Method**, a methodology I developed based on proportional engineering: adopt the simplest professional solution that fully solves the validated problem, expanding controls, tests and documentation according to the actual risk.

The implementation favors small changes, explicit states, local dependencies, reproducible validation, clear rollback paths and human acceptance before integration.
