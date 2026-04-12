# classtrack

A local web app for tracking coursework — assignments, quizzes, exams, and discussions — across multiple classes in a semester. Built around a SQLite event log so completion history is never lost.

## Features

- **Dashboard** — per-class progress, pace vs. deadline, streak, overdue alerts
- **Class detail** — tasks grouped by due date with subtask support, semester label, archive toggle
- **Weekly view** — Sun–Sat calendar with tasks color-coded by class
- **Burndown chart** — task-centric completion curve over the semester
- **Activity heatmap** — GitHub-style completion history across all semesters
- **Canvas importer** — paste the Canvas calendar export, preview, then import
- **Event log** — every toggle is recorded; toggling a task back to incomplete preserves its completion history
- **Daily reminder** — optional macOS notification via launchd

## Stack

| Layer | Tech |
|---|---|
| API server | Go 1.25 + Gin |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Frontend | SvelteKit + Tailwind CSS |
| Charts | Chart.js |

## Requirements

- Go 1.25+
- Node 18+ / npm

## Setup

### 1. Install frontend dependencies

```bash
cd web && npm install
```

### 2. Run

```bash
./start.sh          # Go API on :4000, SvelteKit dev server on :5173
```

Or separately:

```bash
# Terminal 1
cd server && go run .

# Terminal 2
cd web && npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

### 3. (Optional) Migrate from markdown todo files

If you have existing `*Todo.md` files, import them into the database:

```bash
cd server
go run . --migrate --vault ~/path/to/your/todo/directory
```

This walks the directory for `*Todo.md` files and inserts all classes, tasks, and historical `✅ YYYY-MM-DD` completion dates. Safe to re-run after a partial failure. After a successful migration, `migrate.go` can be deleted from the server directory.

## Database

The database lives at `~/.classtrack.db` by default. Override with the `--db` flag:

```bash
go run . --db /path/to/classtrack.db
```

## Adding a class

Go to **Import** → **+ New class** → enter a name and optional semester → Create.

Or via the API:

```bash
curl -X POST http://localhost:4000/api/classes \
  -H 'Content-Type: application/json' \
  -d '{"name": "CS 101", "semester": "Fall 2026"}'
```

## Importing from Canvas

1. In Canvas, open the course calendar and select all → copy
2. In classtrack, go to **Import**, select the target class
3. Paste into the text area → **Preview** → **Import**

The parser strips type prefixes (`Assignment`, `Quiz`, `Discussion Topic`), carries dates forward for multi-task rows, skips administrative rows (refund deadlines, etc.), and deduplicates by task text + due date so re-importing the same paste is safe.

## Archiving a semester

On any class detail page, click the **⚙** gear icon to open class settings. Toggle **Archived** and save. The class disappears from the dashboard and weekly view but its completion history remains in the heatmap.

## Project structure

```
classtrack/
├── server/
│   ├── main.go       # DB init, --migrate flag, gin server
│   ├── db.go         # Store, SQLite schema, all query methods
│   ├── migrate.go    # one-time markdown→SQLite importer
│   ├── models.go     # Task, Class, Stats types; classifyTask
│   ├── canvas.go     # Canvas TSV paste parser
│   ├── stats.go      # on-pace, streak, burndown, heatmap, summary
│   └── handlers.go   # HTTP route handlers
├── web/
│   └── src/
│       ├── routes/
│       │   ├── +page.svelte              # dashboard
│       │   ├── week/+page.svelte         # weekly view
│       │   ├── burndown/+page.svelte     # burndown chart
│       │   ├── heatmap/+page.svelte      # activity heatmap
│       │   ├── import/+page.svelte       # Canvas importer + class creation
│       │   └── class/[id]/+page.svelte   # class detail + settings
│       └── lib/
│           ├── api.js     # fetch wrapper and helpers
│           └── stores.js  # shared nav badge state
└── start.sh                         # launches both processes
```

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/classes` | Active classes with stats |
| `GET` | `/api/classes?all=true` | All classes including archived |
| `POST` | `/api/classes` | Create a class |
| `GET` | `/api/classes/:id` | Single class with full task list |
| `PATCH` | `/api/classes/:id` | Update semester / archived flag |
| `PATCH` | `/api/tasks/:id/toggle` | Toggle task completion (appends event) |
| `GET` | `/api/week` | Tasks for current Sun–Sat week |
| `GET` | `/api/stats/burndown` | Burndown time series |
| `GET` | `/api/stats/heatmap` | Completion counts per day |
| `GET` | `/api/stats/summary` | Overdue / due-today / due-this-week counts |
| `POST` | `/api/import/preview` | Parse Canvas paste without writing |
| `POST` | `/api/import` | Parse and insert Canvas tasks |
