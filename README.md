# Boring Stack

Boring Stack is a small Go starter for building server-rendered web apps with a
fast local loop, clear defaults, and very little ceremony.

The taste is intentionally close to [grugbrain.dev](https://grugbrain.dev/):
complexity is the enemy, abstractions must earn their keep, and working software
that a tired developer can understand is better than clever software that needs
a tour guide. The goal is not to be the strictest possible starter. The goal is
to be a calm base that can build most ordinary apps well: internal tools, CRUD
apps, SaaS dashboards, realtime collaboration surfaces, admin panels, job-backed
workflows, and product prototypes that should be allowed to become real.

It also treats agent-driven development as a first-class use case. Agents do
better when a repo has obvious commands, checked-in generated code, local
instructions, boring file boundaries, and repeatable verification. This repo is
shaped so a human or coding agent can arrive, run `mise run doctor`, read
`AGENTS.md`, and make a small correct change without reverse-engineering a
private workflow.

## The Thesis

Most web apps need the same backbone:

- accept HTTP requests;
- validate client-controlled input;
- read and write relational data;
- render useful HTML;
- update the browser without a full SPA;
- run durable background jobs;
- stream small realtime updates;
- package the app without a pile of build systems;
- make the local loop obvious.

Boring Stack chooses one plain answer for each of those jobs.

It does not try to be a meta-framework. There is no Dagger module, no GitHub
workflow scaffold, no Node project, no Makefile, no GraphQL layer, no frontend
state framework, and no app generator hidden behind a second command surface.
The command surface is `mise run ...`, the server is Go, the UI is HTML, the
database is Postgres, and the default mental model is "follow the request."

## First Run

Install the baseline system packages inside Linux or WSL:

```bash
sudo apt update
sudo apt install -y build-essential ca-certificates curl git unzip
```

Install Docker either through Docker Desktop with WSL integration enabled, or
Docker Engine inside Linux. Confirm:

```bash
docker version
docker compose version
```

Install mise and trust this repo:

```bash
curl https://mise.run | sh
exec $SHELL -l
mise trust
```

Then run:

```bash
mise run first-run
```

Open:

```text
http://localhost:8080
```

`first-run` creates `.env` from `.env.example` when needed, installs pinned
tools, starts Postgres and NATS, runs Goose and River migrations, regenerates
checked-in assets, and starts the app.

If another local project already owns the default service ports, edit `.env`:

```dotenv
POSTGRES_PORT=5433
NATS_PORT=4223
NATS_HTTP_PORT=8223
```

The app derives local `DB_URL` and `NATS_URL` defaults from those ports unless
you set explicit overrides.

## What The Showcase Shows

The included app is a live stack cockpit. It is intentionally over-featured for
a starter, because the first screen should prove what this boring stack can do
without sending you to a slide deck.

- Add, toggle, and delete workflow items backed by Postgres and sqlc.
- Click `Seed flow` to idempotently add a small app-building workflow.
- Click `Broadcast pulse` to write an event and fan out a server snapshot.
- Click `Run River job` to enqueue durable work, write job output, and refresh
  connected browsers.
- Use the local Alpine island to change browser-only UI state without touching
  trusted server state.
- Open two browser tabs and watch both tabs converge through HTMX SSE updates.

The page also includes capability, recipe, request-path, command-loop, runtime,
and version panels. Those panels are still server-rendered HTML from the same
templ file, not a separate docs app or frontend build.

## Stack Choices

### Go

Go is the application language because it is boring in the right way. It gives
the repo a single static binary, fast tests, a strong standard library, simple
concurrency, and predictable deployment. Agents also handle Go well because the
language has fewer hidden runtime conventions than many web stacks.

This repo pins Go in `go.mod` and `mise.toml`. The `toolchain` line is the
source of truth for Docker builds, local tooling, and doctor checks.

### `net/http.ServeMux`

The router is the Go standard library `http.ServeMux` with method and path
patterns. That keeps routing visible and local:

```go
a.mux.HandleFunc("GET /{$}", a.home)
a.mux.HandleFunc("POST /todos", a.createTodo)
a.mux.HandleFunc("POST /todos/{id}/toggle", a.toggleTodo)
```

No framework is introduced just to route requests. For this kind of starter,
the stdlib gives enough shape without creating a second vocabulary.

### Server-Rendered HTML

The app renders HTML on the server because most product UI is still forms,
tables, lists, panels, filters, and detail pages. Server-rendered UI keeps data
ownership simple: the server loads state, validates input, writes data, and
returns the next screen or fragment.

That makes the request path easy to debug. It also gives coding agents a smaller
state space. They can trace a route handler to a templ component to a SQL query
without also reconstructing a client-side cache, hydration boundary, API schema,
and SPA state machine.

### templ

`templ` is used for type-checked Go components. It keeps HTML close to Go data
without hiding the markup behind string concatenation or a browser-only build.

Generated templ files are checked in. That is deliberate:

- `go test ./...` works from a clean copy;
- agents can inspect generated output when debugging rendering issues;
- CI is not required to discover that generated files drifted;
- a broken generator version does not make the whole repo unreadable.

Edit `.templ` files, then run:

```bash
mise run regenerate
```

Do not hand-edit `internal/ui/*_templ.go`.

### HTMX

HTMX handles browser requests, swaps, and out-of-band updates. It is here because
it solves the common "make this server-rendered app feel live" problem without
turning the whole project into a frontend app.

Mutation handlers return HTML fragments. HTMX swaps those fragments into the
page. The browser remains mostly declarative markup:

```html
hx-post="/todos"
hx-target="#composer-panel"
hx-swap="outerHTML"
```

This is good for developer velocity because a change usually touches one handler
and one component, not a handler, JSON DTO, client hook, mutation cache, component
state, and optimistic update path.

### htmx-ext-sse

The SSE extension gives the app realtime updates while keeping the server model
simple. `GET /stream` sends named events containing the same out-of-band fragments
that normal HTMX mutations return.

The app sends full UI snapshots over SSE instead of tiny imperative patches. That
is less clever and more reliable. If a tab misses an event or receives two events
close together, the next snapshot still describes the full current browser state.

### Alpine.js

Alpine is included for local UI islands only. It is useful for small behaviors
like a dropdown, disclosure, tab switcher, or ephemeral local control. It is not
the app state owner.

The rule of thumb is simple: if the state must be trusted, stored, shared, or
used by another request, it belongs on the server. Alpine is for browser-local
convenience.

### Tailwind CSS Standalone

Tailwind is used through the standalone binary. There is no `package.json`, no
Node install, and no frontend package manager.

That choice is about focus. Tailwind gives a fast styling vocabulary and keeps
CSS production-ready, but a Node project would add another dependency graph and
another command surface. For this starter, that tradeoff is not worth it.

The Tailwind binary is pinned in `mise.toml`, installed into ignored `bin/`, and
verified with a checksum on Linux x64. The generated CSS is checked in at
`web/assets/css/app.css`.

Edit templates or CSS sources, then run:

```bash
mise run regenerate
```

`mise run verify` checks that the CSS is current.

### mise

mise is the only task runner. It owns tool versions, environment loading, and
task names. That gives humans and agents one place to look for "how do I do the
thing?"

Common tasks:

```bash
mise run tasks          # list tasks
mise run doctor         # check local prerequisites and exact fixes
mise run first-run      # install, start services, migrate, and run
mise run dev            # regenerate, rebuild CSS, and run server
mise run regenerate     # update sqlc, templ, and checked-in CSS
mise run verify         # check generated files, assets, and tests
mise run check          # normal local quality gate
mise run ci             # extended gate: check + govulncheck + race tests
```

There is intentionally no Makefile. There are intentionally no package manager
scripts. There is intentionally no Dagger pipeline. The local path stays small.

### PostgreSQL

Postgres is the primary durable store because most apps eventually need real
relational data: constraints, transactions, indexes, joins, migrations, and
operational familiarity.

The demo schema is tiny, but the foundation is production-shaped:

- app data lives in normal tables;
- durable jobs live in Postgres through River;
- migrations are explicit;
- integration tests can run against an isolated `TEST_DATABASE_URL`.

### pgx

`pgx` is the Postgres driver. It is widely used, direct, and works cleanly with
sqlc, Goose, and River. The app uses a `pgxpool.Pool` instead of wrapping the
database behind a generic repository layer.

That is intentional grug energy. Database access is already an abstraction. Do
not add another one until repeated app code proves it needs one.

### sqlc

sqlc generates Go methods from raw SQL. This keeps SQL honest and visible while
still giving the Go code typed inputs and outputs.

The SQL lives in:

```text
internal/db/query/*.sql
```

Generated Go lives in:

```text
internal/db/*.go
```

Edit SQL, run `mise run regenerate`, and commit both the SQL and generated Go.
Do not hand-edit generated sqlc files except `internal/db/doc.go`.

### Goose

Goose owns application migrations. It is simple, explicit, and close to the SQL.
Migration files live in:

```text
internal/db/migrations
```

They are embedded into the migrate command, so the built image can run migrations
without needing loose SQL files on disk.

### River

River owns durable background jobs. It uses Postgres, which keeps the starter
from introducing Redis or a separate worker database on day one.

The showcase's `Run River job` button enqueues a River job. The worker writes an
event row and publishes a realtime notification. That path demonstrates a common
app shape: user action, durable job, database side effect, browser update.

River owns its own schema through River's migrator. Goose owns the app schema.
That boundary stays clear.

### Memory Bus And NATS

The realtime bus is ephemeral fanout for connected browser streams. It is not a
durable queue and should not be used as one.

By default the app uses an in-memory bus:

```bash
mise run dev
```

To test cross-process fanout, use NATS:

```bash
BUS=nats mise run dev
```

Run two app processes on different ports to see why NATS exists:

```bash
ADDR=:8080 BUS=nats mise run dev
ADDR=:8081 BUS=nats mise run dev
```

Both processes can publish changes through the same NATS subject while durable
state remains in Postgres.

### `slog`

The app uses Go's `log/slog` for structured logs. JSON logs are boring, useful,
and easy to ship into any log collector later. No logging framework is needed.

### Embedded Assets And Migrations

Assets and migrations are embedded into the binary. This keeps local execution
and container execution close to each other:

- the server can serve CSS and vendor JS from the binary;
- the migrate command can apply embedded SQL migrations;
- the distroless container does not need a copied source tree.

Checked-in generated files plus embedded runtime files make the repo easier for
agents too. There are fewer "it works only after this hidden generation step"
states.

### Docker

The Docker image is distroless and contains two binaries:

- `/server`, the default entrypoint;
- `/migrate`, for applying Goose and River migrations.

Build it through mise so all build args come from the same pins:

```bash
mise run docker-build
```

Run migrations from the image with an explicit entrypoint:

```bash
docker run --rm --entrypoint /migrate \
  -e DB_URL='postgres://app:app@host.docker.internal:5432/app?sslmode=disable' \
  boring-stack:local
```

Then run the server with the normal entrypoint:

```bash
docker run --rm -p 8080:8080 \
  -e ADDR=':8080' \
  -e DB_URL='postgres://app:app@host.docker.internal:5432/app?sslmode=disable' \
  boring-stack:local
```

## Agent-Driven Development

This repo uses the [AGENTS.md](https://agents.md/) convention so coding agents
can discover project-specific instructions. `CLAUDE.md` points at the same file
so different agent surfaces read one source of truth.

The agent contract is practical:

- prefer the smallest clear change;
- keep related behavior near the route, template, query, or job that uses it;
- avoid clever abstractions until the code earns them;
- use `mise run doctor` before guessing at environment problems;
- use `mise run regenerate` after SQL, templ, or CSS source edits;
- use `mise run verify`, `mise run check`, or `mise run ci` depending on risk;
- do not add GitHub workflows or Dagger unless explicitly asked.

The repo also follows agent-friendly web guidance: server-rendered HTML preserves
real semantic structure, browser interactions are normal forms and links where
possible, and the UI remains inspectable from raw HTML and the accessibility
tree. That helps users, search, and agents at the same time.

## Developer Velocity

Velocity here means "how quickly can a developer make a correct change and know
it worked?"

Boring Stack optimizes that path:

- one command installs tools and vendors browser assets;
- one command starts services, migrates, and runs the app;
- one command regenerates all checked-in generated outputs;
- one command verifies generated drift, vendored browser checksums, and tests;
- Docker port conflicts include owner hints and support simple alternate ports;
- generated files are checked in so a fresh clone can run tests immediately;
- the router, handlers, templates, SQL, migrations, jobs, and realtime bus all
  live in obvious directories.

The normal edit loop is:

```bash
mise run dev
# edit handler/template/sql/css
mise run regenerate
mise run verify
```

For bigger changes:

```bash
mise run check
mise run ci
```

`check` is the day-to-day gate. `ci` is the heavier local gate with vulnerability
and race checks. Keeping those separate is intentional. The repo favors a fast
loop by default, with stricter tools available when the risk is higher.

## Commands

```bash
mise run tasks          # list available tasks
mise run install        # install pinned tools, Go modules, and browser assets
mise run tools          # check pinned codegen and local tooling versions
mise run doctor         # check local prerequisites and exact fixes
mise run first-run      # setup, start services, migrate, and run the app
mise run start          # alias for first-run
mise run setup          # alias for install
mise run fmt            # format Go and templ files
mise run fmt:check      # check formatting
mise run lint           # Go static checks and module verification
mise run vendor-js      # download pinned HTMX, SSE, and Alpine bundles
mise run tailwind       # install the pinned Tailwind standalone binary
mise run up             # start Postgres and NATS
mise run wait-db        # wait for local Postgres
mise run migrate        # Goose app migrations plus River schema migrations
mise run generate       # sqlc plus templ codegen
mise run css            # Tailwind standalone build
mise run regenerate     # generate plus CSS
mise run dev            # regenerate, rebuild CSS, and run server
mise run dev-css        # watch Tailwind CSS during UI work
mise run build          # build a local Linux server binary into ./bin
mise run test           # run Go tests
mise run test-db        # run opt-in DB-backed integration tests
mise run verify         # generated drift, asset checksum, and tests
mise run check          # fmt check, lint, verify
mise run ci             # check, govulncheck, race tests
mise run go:check       # Go-only formatting, linting, and tests
mise run vuln           # run govulncheck explicitly
mise run test-race      # run Go tests with the race detector
mise run cover          # write a coverage profile to coverage/go.out
mise run docker-build   # build distroless image
mise run down           # stop containers
mise run reset-db       # remove local Postgres volume and recreate DB
mise run clean          # remove local build artifacts
```

Opt-in DB-backed tests require an isolated database:

```bash
TEST_DATABASE_URL='postgres://app:app@localhost:5432/app_test?sslmode=disable' mise run test-db
```

## File Layout

```text
cmd/server              app entrypoint
cmd/migrate             Goose and River migration entrypoint
compose.yaml            local Postgres and NATS service model
internal/config         environment config
internal/server         HTTP routes, handlers, middleware-ish helpers
internal/ui             templ components and generated templ Go
internal/db/query       handwritten SQL for sqlc
internal/db/migrations  Goose migrations
internal/db             sqlc generated Go
internal/jobs           River workers
internal/realtime       memory and NATS fanout implementations
web/assets              embedded CSS and vendored browser JS
scripts                 mise task helpers
AGENTS.md               project instructions for coding agents
mise.toml               tool pins and task surface
```

## Version Pins

The main pins are centralized in `mise.toml` and `go.mod`. Docker builds receive
their build args from `mise run docker-build`; prefer that task over calling
`docker build` directly.

| Component | Version |
|---|---:|
| Go | 1.26.4 |
| HTMX | 2.0.10 |
| htmx-ext-sse | 2.2.4 |
| Alpine.js | 3.15.12 |
| templ | v0.3.1020 |
| Tailwind CSS standalone CLI | v4.3.1 |
| sqlc | v1.31.1 |
| pgx | v5.10.0 |
| goose | v3.27.1 |
| River | v0.39.0 |
| nats.go | v1.52.0 |
| nats-server Docker image | 2.14.2-alpine |
| PostgreSQL Docker image | 18.4-alpine |

## Editing Rules

- Edit SQL in `internal/db/query/*.sql`, then run `mise run regenerate`.
- Edit templ components in `internal/ui/*.templ`, then run `mise run regenerate`.
- Edit Tailwind sources or classes, then run `mise run regenerate`.
- Do not hand-edit generated sqlc files except `internal/db/doc.go`.
- Do not hand-edit `internal/ui/*_templ.go`.
- Keep HTMX and Alpine behavior near the markup that uses it.
- Treat every browser-submitted value as client-controlled.
- Use River for durable work.
- Use the memory/NATS bus only for ephemeral fanout to connected streams.

## What This Starter Deliberately Leaves To The App

Some choices should be made by the real product, not by a starter:

- authentication and sessions;
- CSRF policy;
- authorization and tenant boundaries;
- user/account/org tables;
- API versioning and JSON shape;
- file uploads;
- email provider;
- payment provider;
- production deploy target;
- observability vendor;
- worker/server process split;
- frontend design system beyond the demo surface.

Those are important. They are also app-specific. The stack gives you the path to
add them without forcing a premature answer.

## Troubleshooting

Start here:

```bash
mise run doctor
```

If a port is already in use, doctor prints the likely owner. If another local
project owns the default Compose ports, set alternate ports in `.env`:

```dotenv
POSTGRES_PORT=5433
NATS_PORT=4223
NATS_HTTP_PORT=8223
```

If Postgres is not ready:

```bash
mise exec -- docker compose ps
mise run wait-db
mise run migrate
```

If Docker Desktop on WSL reports a closed pipe or a distro home directory error,
reset the Desktop/WSL bridge:

```powershell
wsl.exe --shutdown
```

Then reopen Ubuntu, start Docker Desktop, enable this distro under
`Settings > Resources > WSL integration`, and confirm:

```bash
docker version
docker compose version
mise run doctor --quick
```

If browser interactions do nothing:

```bash
mise run vendor-js
```

If CSS looks plain:

```bash
mise run css
```

If generated files are stale:

```bash
mise run regenerate
```
