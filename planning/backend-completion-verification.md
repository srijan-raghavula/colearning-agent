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

## Report back

- Any build/vet/test/race failures with file + line.
- Whether the restart preserved state.
- Whether HTTP status/envelope behaviors match `docs/api.md`.
- Anything out of scope — any real AI-provider or UI code should **not** be present.

Do not fix anything yet — just report what passes and what fails.
