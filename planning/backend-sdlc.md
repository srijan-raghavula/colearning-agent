# Backend SDLC Tracker — Co-Learning Learning Domain

Single tracker for the phased backend work. Kept brief; status per phase only.
Constraint: no new dependencies (`go get` unavailable); extend existing packages.

| Phase | Scope | Status | Notes |
|---|---|---|---|
| 1. Learning domain foundation | Domain entities (session, turn, evidence, understanding check, course/concept/objective/material, tutor policy), repository ports, in-memory adapters, tutor + understanding-check services, student/teacher usecases, controllers/routes, bootstrap wiring. Current UI and `/api/v1/*` verified against it. | ✅ Done | Docs: `docs/project-architecture.md`, `docs/api.md`. Legacy assessment slice untouched and secondary. |
| 2. Minimal durable persistence (in progress) | `repository.LearningSnapshot` (versioned, provider-neutral), snapshot export/restore on the memory adapters, `adapter/persist.LearningStore` (atomic JSON write via temp+rename), after-change hook on the mutating usecases, boot-time load-or-seed, graceful-shutdown flush, `LEARNING_STORE_PATH` config (`off` to disable). No repository port or API/UI change. | ✅ Done (this commit) | Round-trip, seed-replacement, and mutation-persistence unit tests in `src/adapter/persist`; restart smoke test verified restore. Next hardening ideas: save-on-interval backstop, snapshot validity guard for corrupt files (already logged + fall back to seeds). |
| 3. Real tutor gateway integration | Replace stub `LearningAgentGateway` / `FormativeCheckGateway` behind the ports with a configurable provider adapter; keep validation layer (`service.TutorAgent`) as the contract gate. | ⏳ Planned | No usecase/controller change needed — ports already primary. |
| 4. Identity & access | Real session identity replacing demo role headers/defaults; per-mile ownership checks already in usecases. | ⏳ Planned | UI stays unchanged (role headers map to identity later). |
| 5. Production persistence swap | Replace file snapshot with a real database adapter satisfying the same repository ports (SQLite/Postgres via `database/sql` when a driver is permissible). | ⏳ Planned | Decision deferred until driver approval; snapshot schema is the migration seed. |
| 6. Hardening & observability | Structured logging/error taxonomy, load tests, doc reconciliation (HTML route naming in `docs/api.md`), CI additions. | ⏳ Planned | |

## Acceptance checkpoints
- Every phase: `go test ./...` green; UI surfaces and `/api/v1/*` endpoints unchanged or extended without breaking contract.
- Persistence guarantees from Phase 2: crash-safe writes (atomic rename), version-guarded restore, seed fallback on missing/corrupt snapshot.
- `docs/project-problem-statement.md` is never modified.
