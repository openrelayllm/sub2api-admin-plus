# Sub2API Admin Plus

Sub2API Admin Plus is an operations automation extension built from the Sub2API codebase.

MVP 0 keeps the Sub2API frontend/backend architecture, UI conventions, build scripts, and deployment layout as a runnable baseline. The current business layer already includes the first real operations APIs and pages; Chrome extension collection and scheduler automation are still pending.

## Scope

- Keep the Sub2API Go/Gin backend structure.
- Keep the Sub2API Vue/Vite/Tailwind frontend structure and UI style.
- Reuse Sub2API admin authentication semantics.
- Reuse PostgreSQL and Redis infrastructure, with Admin Plus data isolated by database and Redis key prefix.
- Add operations automation features incrementally.

## Current Status

Implemented:

- Supplier parent records.
- Supplier account/key child bindings to local Sub2API `accounts.id`.
- Rate, balance, health, promotion, billing, reconciliation, extension task, and action recommendation APIs.
- Local Sub2API read adapter for real `accounts` and `usage_logs`.
- Admin Plus operation pages, including supplier bindings, billing reconciliation, and local usage.
- API E2E script using real HTTP and PostgreSQL.

Not implemented yet:

- Real Chrome extension login/scraping/export.
- 10-minute scheduler jobs.
- Sub2API Redis concurrency read adapter.
- Notification and audit execution loop.
- Confirmed action execution through Sub2API Admin API.

## MVP 0 Rules

- Do not modify the upstream Sub2API repository at `/Users/coso/Documents/dev/go/sub2api`.
- Do not rewrite the Go module path yet; the backend still imports `github.com/Wei-Shaw/sub2api` internally to keep the cloned baseline buildable.
- Do not delete large Sub2API backend/frontend modules until the baseline is verified.
- Keep product and architecture notes in `docs/`.

## Source Baseline

- Source path: `/Users/coso/Documents/dev/go/sub2api`
- Source commit: `4a5665da5b2c6b83c4597844ea6e573746c821b1`

## Development

Backend:

```bash
cd backend
go test ./...
go build -o bin/server ./cmd/server
```

Frontend:

```bash
cd frontend
pnpm install
pnpm run typecheck
pnpm run build
```

Focused verification:

```bash
cd backend
go test ./internal/adminplus/... ./internal/handler/adminplus/... ./internal/server/routes/...

cd ../frontend
pnpm run typecheck
pnpm run test:run -- src/router/__tests__/admin-plus-routes.spec.ts

cd ..
node tools/admin-plus-e2e.mjs
```

E2E defaults:

- `ADMIN_PLUS_BASE_URL=http://localhost:3000`
- `ADMIN_PLUS_E2E_EMAIL=admin@sub2api-admin-plus.local`
- `ADMIN_PLUS_E2E_PASSWORD=AdminPlus@123456`
- `ADMIN_PLUS_E2E_DB_URL=postgresql://root:root@127.0.0.1:5432/sub2api_admin_plus?sslmode=disable`

The E2E script creates `e2e-*` rows in PostgreSQL to verify real API and DB paths. These rows are test fixtures, not mock production collection.

## Sub2API Read Integration

Admin Plus writes its own data to the Admin Plus database. To read real local Sub2API accounts and usage from another database, set:

```bash
export SUB2API_READONLY_DATABASE_URL='postgresql://root:root@127.0.0.1:5432/sub2api?sslmode=disable'
```

If this variable is not set, the backend falls back to the current database connection for local MVP verification.

## Documentation

- Product requirements: `docs/sub2api-admin-plus-prd.md`
- Code structure plan: `docs/code-structure.md`
- MVP baseline/progress: `docs/mvp0-baseline.md`
