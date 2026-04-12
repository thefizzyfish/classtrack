# classtrack

A local web app for tracking NVCC coursework from Obsidian markdown todo files. Reads and writes directly to your vault — no separate database, no sync conflicts.

## Features

- **Dashboard** — per-class progress bars, pace vs. deadline, streak, overdue alerts
- **Class detail** — all tasks by due date with subtask support and toggle
- **Weekly view** — Sun–Sat calendar with tasks color-coded by class
- **Burndown chart** — task-centric completion curve over the semester
- **Activity heatmap** — GitHub-style completion history across all semesters
- **Canvas importer** — paste the Canvas calendar export, preview, and import in one step
- **Live sync** — file watcher reloads instantly when Obsidian edits a todo file
- **Daily reminder** — optional macOS notification via launchd at 8am

## Stack

| Layer | Tech |
|---|---|
| API server | Go 1.22 + Gin |
| File watching | fsnotify |
| Frontend | SvelteKit + Tailwind CSS |
| Charts | Chart.js |
| Storage | Obsidian Markdown (source of truth) |

## Requirements

- Go 1.22+
- Node 18+ (via nvm)
- Obsidian vault with NVCC todo files at `~/Obsidian/Obsidian_Vault/NVCC/`

## Running

```bash
cd ~/classtrack
./start.sh          # starts Go API on :4000 and SvelteKit on :5173
```

Then open [http://localhost:5173](http://localhost:5173).

Or run separately:

```bash
# Terminal 1 — API
cd server && go run .

# Terminal 2 — UI
cd web && npm run dev
```

## Vault structure

classtrack auto-discovers any file matching `*Todo.md` anywhere under the NVCC vault directory. Each file should follow this format:

```markdown
---
tags: [nvcc, todo]
created: 2026-03-21
semester: Spring 2026   # optional
archived: true          # optional — hides from dashboard
---
# Class Name Todo

## Mar 29
- [x] Some completed task ✅ 2026-03-22
- [ ] An incomplete task

## Apr 5
- [ ] Another task
    - [ ] A subtask
```

The `✅ YYYY-MM-DD` completion marker is written by the Obsidian Tasks plugin (or by classtrack when you toggle a task in the UI).

## Adding a new class

**Option A — from the UI:** Go to Import → click **+ New class** → enter the class name → Create. The directory and `Todo.md` are created automatically.

**Option B — manually:** Create `NVCC/<Class Name>/<Class Name> Todo.md` in your vault with the frontmatter above. classtrack picks it up within 500ms.

## Importing from Canvas

1. In Canvas, open the course calendar and select all → copy
2. Go to classtrack → Import
3. Select the target class (create it first if needed)
4. Paste into the text area → Preview → Import

The parser strips type prefixes (`Assignment`, `Quiz`, `Discussion Topic`), carries dates forward for multi-task rows, and skips administrative rows (refund dates, withdrawal deadlines, etc.).

## Archiving a semester

Add `archived: true` to a class's frontmatter. It disappears from the dashboard and all active views, but its completion history remains in the heatmap.

## Daily reminder (optional)

```bash
cp ~/classtrack/scripts/com.classtrack.remind.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.classtrack.remind.plist

# Test immediately
launchctl start com.classtrack.remind
```

Fires a macOS notification at 8am with overdue/due-today counts. Silently skips if the Go server isn't running.

## Project structure

```
classtrack/
├── server/
│   ├── main.go       # server setup, file watcher, Store
│   ├── models.go     # Task, Class, Stats, Summary types
│   ├── parser.go     # vault discovery, markdown parsing
│   ├── writer.go     # atomic task toggle write-back
│   ├── canvas.go     # Canvas TSV paste parser + file appender
│   ├── stats.go      # on-pace, streak, burndown, heatmap, summary
│   └── handlers.go   # all HTTP route handlers
├── web/
│   └── src/
│       ├── routes/
│       │   ├── +page.svelte          # dashboard
│       │   ├── week/+page.svelte     # weekly view
│       │   ├── burndown/+page.svelte # burndown chart
│       │   ├── heatmap/+page.svelte  # activity heatmap
│       │   ├── import/+page.svelte   # Canvas importer
│       │   └── class/[id]/+page.svelte # class detail
│       └── lib/
│           ├── api.js     # typed API client
│           └── stores.js  # shared nav badge state
├── scripts/
│   ├── remind.sh                    # osascript notification
│   └── com.classtrack.remind.plist  # launchd plist (8am daily)
└── start.sh                         # launches both servers
```
