# Backend Completion — Verification Brief

A handoff brief for an agent (or developer) verifying the `feat/backend-completion`
branch. The changes were committed but **never compiled or test-run** in the
environment where they were written (no Go/Docker there). The task is to verify
they actually build and behave correctly.

## Context

`colearning-agent` is a Go web app: a teacher-grounded human–AI co-learning tool.
It uses in-memory repositories with a file-backed JSON snapshot for durability, a
deterministic stub tutor gateway (no real AI provider yet), and a demo
role/identity seam via query params / headers. Go version is 1.24.1 (see
`go.mod`). It ships a `Dockerfile` and `docker-compose.yml`.

A backend hardening pass was done on branch `feat/backend-completion`. Scope was
deliberately limited to the backend — **no real AI/provider integration and no UI
changes**, since those are owned separately. Governing docs: `docs/project-architecture.md`,
`docs/api.md`, and `planning/backend-sdlc.md` (see the Phase 2a row).

## What was changed (verify each against the code and docs)

1. **Persistence** (`src/adapter/persist/learning_store.go`, `src/bootstrap/app.go`):
   candidate snapshot is validated before atomic temp-write+rename; unsupported
   snapshot versions are preserved (never overwritten) and cause startup to fail
   closed; corrupt current-version snapshots are quarantined before an explicit
   reseed; prior v1 snapshots still load, including historical evidence with
   empty/missing turn references. `bootstrap.NewApp` now returns `(App, error)`
   and `main` handles that error.
2. **Workflow concurrency** (`src/usecase/learning_portal.go`): a reference-counted
   keyed lock — start keyed by student/course, turns/checks keyed by session — so
   unrelated learners aren't serialized across the provider call. Check for
   deadlocks, lock-order issues (start → session), and that keyed map entries are
   cleaned up at zero users.
3. **Validation / retry**: session ownership/status/course/concept and effective
   tutor mode are validated before any turn mutation; an unmatched student turn is
   retained on provider failure and reused only on an identical body+mode retry (a
   different message is rejected until answered); session mode is reconciled after
   a policy change.
4. **HTTP** (`src/routes/*`, `src/controller/learning_controller.go`,
   `src/controller/api_dto.go`): exact root route so unknown paths don't return the
   landing page; JSON `404`/`405`/role-forbidden envelopes on `/api/v1`; strict
   bounded `application/json` decoding (media type, 1 MiB limit, unknown-field
   rejection, single JSON value); `sources` always serializes as `[]`, not omitted.
5. **Deploy** (`docker-compose.yml`): durable named volume at `/app/data` with a
   non-root (UID/GID 10001) ownership-init step.
6. **One repository port changed**: `LearningSessionRepository.FindActive` gained a
   concept argument. Confirm all implementers/callers match.

## Commands to run and report

Static + tests:

```bash
git checkout feat/backend-completion
gofmt -l ./src        # expect no files listed
go vet ./...
go build ./...
go test ./...
go test -race ./...   # concurrency changes make this valuable
```

Runtime smoke test:

```bash
docker compose up -d --build
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/api/v1/unknown                         # expect JSON 404
curl -i -X PUT http://localhost:8080/api/v1/student/dashboard        # expect JSON 405
curl -i "http://localhost:8080/api/v1/student/dashboard?role=teacher" # expect JSON 403
docker compose restart app && curl -i http://localhost:8080/healthz   # state should survive restart
docker compose down
```

## Verification results — 2026-10-25

### Environment and permissions

- The host has no `go` executable. Go checks were run in the official
  `golang:1.24-alpine` container (Go 1.24.13); no Go toolchain was installed on
  the host.
- Docker Engine 29.8.0 and Docker Compose 2.40.3 were available. Compose warned
  that Buildx was missing, but the image build completed successfully using the
  Docker driver.
- The race test needs a C compiler on Alpine, so `gcc` and `musl-dev` were
  installed only in its disposable test container. No host packages were
  installed.
- `scripts/smoke-persistence.sh` was not run verbatim because it invokes host
  `go run`; its persistence scenario was exercised against the Compose app
  instead.

### Checks

- `gofmt -l ./src`: passed; no files listed.
- `go vet ./...`, `go build ./...`, `go test ./...`, and `go test -race ./...`:
  all passed in the Go container.
- `docker compose build app`: passed.
- Runtime contract: `/healthz` returned `200`; an unknown `/api/v1` path
  returned JSON `404`; `PUT` on the student dashboard returned JSON `405`; and
  a teacher role on the student dashboard returned JSON `403`.
- Runtime JSON checks passed for missing/wrong media type (`415`), malformed,
  unknown-field, multiple-value, and empty/null bodies (`400`), and a body over
  1 MiB (`413`). Error responses used the JSON error envelope; turn `sources`
  serialized as `[]`.
- Persistence: a posted student turn was present after restarting the app, and
  startup logged that it restored the learning snapshot. Compose created the
  `colearning-agent_learning-data` volume during this run, so this verifies
  persistence across restart on that volume, not migration of data from the
  previously running container.
- The service already running before the rebuild was stale: unknown routes and
  wrong methods returned the landing page with `200`. Recreating it from the
  checked-out Compose configuration corrected both behaviors.

### Regression tests added

- Keyed workflow locks: same-key serialization, independent keys, and registry
  cleanup after the final user releases a key.
- Tutor retry behavior: a failed provider request retains the student turn;
  only an identical message and mode can retry it.
- Snapshot handling: unsupported versions remain untouched, and corrupt
  snapshots are quarantined with their original bytes preserved.
- JSON decoding and DTOs: strict media type/size/shape handling and empty
  `sources` serialization.

No backend implementation defect was found in the verified scope, so no
production behavior was changed. The stub tutor remains in use; no real
AI-provider integration or UI work was added. The Compose service was left
running after the smoke checks.
