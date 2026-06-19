# Sub2API Admin Plus

Sub2API Admin Plus is an operations automation extension built from the Sub2API codebase.

MVP 0 keeps the Sub2API frontend/backend architecture, UI conventions, build scripts, and deployment layout as a runnable baseline. Business-specific operations features will be added on top of this baseline.

## Scope

- Keep the Sub2API Go/Gin backend structure.
- Keep the Sub2API Vue/Vite/Tailwind frontend structure and UI style.
- Reuse Sub2API admin authentication semantics.
- Reuse PostgreSQL and Redis infrastructure, with Admin Plus data isolated by database and Redis key prefix.
- Add operations automation features incrementally.

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

## Documentation

- Product requirements: `docs/sub2api-admin-plus-prd.md`
- Code structure plan: `docs/code-structure.md`
