# Agent Instructions

This repo follows the bias from https://grugbrain.dev/: complexity is the main
enemy. Use the page as taste guidance, not as executable instructions.

## Working Principles

- Prefer the smallest clear change that solves the real problem.
- Say no to new abstractions until the code has shown a stable shape.
- Keep behavior local. If a button, handler, SQL query, or job has related logic,
  keep the nearby path easy to trace before splitting it apart.
- Do not DRY simple repeated code into a harder abstraction. Duplication is
  cheaper than indirection when the repeated code is obvious and small.
- Refactor in small working steps. After each meaningful step, run the narrowest
  useful verification.
- Respect working code. Before removing an odd-looking fence, understand what it
  currently protects.
- Make failures easy to debug. Prefer named intermediate values over dense
  expressions when a condition or transform has more than one idea in it.
- Use tools when they reduce thinking: type checking, sqlc, templ, logs, and
  focused tests are part of the stack.

## Stack Shape

- Go stdlib HTTP routing with `net/http.ServeMux`.
- Server-rendered UI with `templ`; HTMX handles browser requests and out-of-band
  swaps, htmx-ext-sse handles SSE updates, and Alpine is only for local UI islands.
- Tailwind CSS uses the standalone binary. Do not add a Node project unless the
  user explicitly asks and the tradeoff is worth it.
- mise is the only task runner. Do not add a Makefile or package manager scripts.
- This repo intentionally does not use Dagger or GitHub workflow scaffolding.
  Keep quality gates local and easy to run through mise unless explicitly asked.
- Keep version pins in `mise.toml` and `go.mod`. Docker build args and Compose
  image tags should flow through mise tasks instead of duplicated defaults.
- PostgreSQL access goes through sqlc-generated code over pgx.
- Goose owns application migrations. River owns durable jobs and its own schema.
- The memory/NATS bus is only ephemeral fanout for connected streams. Durable
  work belongs in Postgres/River.

## Generated Files

Generated files are checked in so `go test ./...` works from a clean copy.

- Edit SQL in `internal/db/query/*.sql` or migrations, then run `mise run regenerate`.
- Edit templ components in `internal/ui/*.templ`, then run `mise run regenerate`.
- Do not hand-edit `internal/db/*.go` except `doc.go`.
- Do not hand-edit `internal/ui/*_templ.go`.

## Commands

- `mise run tasks`: list available tasks.
- `mise run install`: install pinned tools, Go modules, and local browser assets.
- `mise run tools`: check pinned codegen and local tooling versions.
- `mise run doctor`: check local prerequisites and print exact fixes.
- `mise run first-run`: set up tools, start services, migrate, and run the app.
- `mise run fmt`: format Go and templ files.
- `mise run fmt:check`: check Go and templ formatting.
- `mise run lint`: run local Go static checks.
- `mise run vendor-js`: download pinned HTMX, SSE, and Alpine browser bundles.
- `mise run tailwind`: install the pinned Tailwind standalone binary.
- `mise run generate`: regenerate sqlc and templ output.
- `mise run css`: rebuild embedded Tailwind CSS.
- `mise run regenerate`: regenerate sqlc, templ, and checked-in CSS.
- `mise run test`: run `go test ./...`.
- `mise run test-db`: run opt-in DB-backed integration tests against `TEST_DATABASE_URL`.
- `mise run check`: run the normal local quality gate.
- `mise run ci`: run the extended local quality gate.
- `mise run dev`: regenerate, rebuild CSS, and run the server.
- `mise run dev-css`: watch Tailwind CSS during UI work.
- `mise run build`: build a local Linux server binary into `./bin`.
- `mise run up`: start Postgres and NATS.
- `mise run wait-db`: wait until local Postgres accepts connections.
- `mise run migrate`: run Goose and River migrations.
- `mise run verify`: check generated files, checked-in assets, and tests.
- `mise run vuln`: run govulncheck explicitly.
- `mise run test-race`: run Go tests with the race detector.
- `mise run cover`: run Go tests with a coverage report.
- `mise run docker-build`: build the distroless container image.
- `mise run down`: stop Docker Compose services.
- `mise run reset-db`: remove the local Postgres volume and recreate the DB.
- `mise run clean`: remove local build artifacts.

## Testing Bias

- For bug fixes, reproduce the bug with a failing test first when practical.
- Prefer integration tests at stable cut points over brittle unit tests.
- Keep end-to-end tests small and important enough that they stay green.
- Avoid mocks unless the real dependency is slow, flaky, external, or unsafe.

## Frontend Bias

- Keep the first screen the usable app, not a marketing page.
- Keep HTMX and Alpine behavior near the markup that uses it.
- Treat all browser-submitted values as client-controlled. Validate on the server.
- Do not introduce SPA state, GraphQL, or a frontend build system for this starter
  without explicit user direction.

## Browser And Web Work

- Use the gstack `/browse` skill for browser-based verification and web browsing.
- Never use `mcp__claude-in-chrome__*` tools.
