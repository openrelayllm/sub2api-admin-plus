# MVP 0 Baseline

## Goal

MVP 0 creates a runnable Sub2API Admin Plus baseline by copying the full Sub2API frontend/backend architecture and UI design into this repository.

This phase intentionally keeps most Sub2API modules intact. The goal is a stable development base, not immediate business pruning.

## Source

| Field | Value |
|-------|-------|
| Source path | `/Users/coso/Documents/dev/go/sub2api` |
| Source commit | `4a5665da5b2c6b83c4597844ea6e573746c821b1` |
| Target path | `/Users/coso/Documents/dev/ai/openrelayllm/sub2api-admin-plus` |

## Copied Areas

- `backend/`
- `frontend/`
- `deploy/`
- `.github/`
- root Docker and build files
- `assets/`
- `skills/`
- `tools/`

The source `.git/`, `node_modules/`, `dist/`, and `.DS_Store` files were not copied.

## Intentional Carryovers

- Backend module path remains `github.com/Wei-Shaw/sub2api`.
- Existing backend package imports remain unchanged.
- Existing frontend route/component structure remains unchanged.
- Existing deployment files remain mostly unchanged.

These carryovers keep the copied baseline buildable. Rename/import migration and feature pruning should be handled as explicit later tasks.

## MVP 0 Cleanup Done

- Root `README.md` now describes Sub2API Admin Plus.
- Root `DEV_GUIDE.md` now describes the Admin Plus development baseline.
- `frontend/package.json` package name changed to `sub2api-admin-plus-frontend`.
- `.gitignore` explicitly allows Admin Plus docs and frontend lockfile to be tracked.

## Verification

| Check | Result | Note |
|-------|--------|------|
| `cd backend && go build -o bin/server ./cmd/server` | PASS | Local Go is `go1.24.3`; Go toolchain downloaded/used `go1.26.4` required by `backend/go.mod`. |
| `cd frontend && pnpm run typecheck` | PASS | TypeScript/Vue type check passed. |
| `cd frontend && pnpm run build` | PASS | Build passed with a non-blocking Vite chunk-size warning. |

Generated artifacts are intentionally ignored:

- `backend/bin/`
- `backend/internal/web/dist/`
- `frontend/node_modules/`
- `frontend/*.tsbuildinfo`
- `frontend/vite.config.d.ts`
- `frontend/vite.config.js`

## Next Cleanup Candidates

- Add Admin Plus login proxy routes.
- Add Admin Plus navigation entry points.
- Decide whether to keep or disable upstream Sub2API release workflows.
- Rename Docker image/container/service names after the baseline build is verified.
- Remove or hide non-MVP user-facing pages only after route and auth impact is understood.
