# colearning-agent

CoLearn is a teacher-grounded human–AI co-learning application. The primary student experience is a configurable tutor conversation grounded in teacher-provided concepts and labeled source material. Automated evaluation remains an optional formative capability and the original assignment routes are retained for compatibility.

## Product documentation

- Original copied requirement: [`docs/project-problem-statement.md`](docs/project-problem-statement.md) — intentionally preserved verbatim
- Target architecture and Mermaid diagrams: [`docs/project-architecture.md`](docs/project-architecture.md)
- Versioned JSON API: [`docs/api.md`](docs/api.md)
- UI architecture: [`docs/ui-architecture.md`](docs/ui-architecture.md)
- User flow: [`planning/userflow-mermaid.md`](planning/userflow-mermaid.md)
- User stories: [`planning/user-stories.md`](planning/user-stories.md)
- Feature plans: [`planning/features/`](planning/features/)

## Current vertical slice

The first slice is intentionally thin but demonstrable:

- one seeded teacher-owned course
- four ordered concepts with objectives and provenance-aware materials
- resumable student learning sessions and tutor turns
- guided, Socratic, direct, practice, hint, and diagnostic modes
- deterministic provider-neutral tutor gateway
- concept progress and next-action read models
- polished student dashboard and learning space
- teacher class dashboard and content studio
- server-rendered HTML and versioned JSON API surfaces

Accounts, course content, learning sessions, turns, and agent execution records use SQLite. Progress and formative-check read models use the durable snapshot store. Signed account identity controls student and teacher route access; the seeded demo accounts remain available for local review.

## Project structure

```text
src/
  cmd/api/                     # entrypoint only
  adapter/memory/              # in-memory repositories and deterministic gateways
  adapter/sqlite/              # durable accounts, content, sessions, and agent records
  agentic/                     # tutor orchestration and formative-check collaboration
  auth/                        # signed identity, password hashing, and account contracts
  bootstrap/                   # dependency wiring and demo seed data
  config/                      # runtime configuration
  controller/                  # HTML and JSON request orchestration
  domain/                      # learning and legacy assessment entities
  repository/                  # persistence and provider ports
  routes/                      # route registration and identity-based role middleware
  service/                     # tutor response policy and legacy evaluator
  templates/                   # server-rendered HTML and shared CSS
  usecase/                     # student/teacher learning workflows
  view/                        # buffered template rendering
planning/                      # product flow, stories, and feature plans
docs/                          # architecture, API, and original requirement
```

## Quickstart

### Local development

```bash
go test ./...
go vet ./...
go run ./src/cmd/api
```

Open:

- `http://localhost:8080/` — product landing page
- `http://localhost:8080/login` — sign in with the demo student or teacher account
- `http://localhost:8080/signup` — create a persistent student or teacher account
- `http://localhost:8080/profile` — view the signed-in account

The demo accounts are `student@demo.edu` / `student123` and `teacher@demo.edu` / `teacher123`. Role access comes from the signed-in account; query parameters do not select a user or role.

Set `COLEARN_JWT_SECRET` to a stable random value of at least 32 bytes before deployment so signed-in sessions survive restarts. Set `COLEARN_COOKIE_SECURE=true` when TLS terminates at a trusted reverse proxy.

Set `COLEARN_DEMO_ACCOUNTS=false` before deployment to disable the seeded demo passwords. The demo account records remain as course references but receive fresh random passwords at startup.

### Docker and containerization

```bash
docker build -t colearning-agent:latest .
docker run -p 8080:8080 --name colearning-agent colearning-agent:latest
```

Or:

```bash
docker compose up -d
docker compose ps
docker compose logs -f
docker compose down
```

## API surface

The stable versioned contract is documented in [`docs/api.md`](docs/api.md).

```text
GET  /healthz
GET  /api/v1/student/dashboard
POST /api/v1/student/sessions
GET  /api/v1/student/sessions/{session_id}
POST /api/v1/student/sessions/{session_id}/turns
POST /api/v1/student/sessions/{session_id}/understanding-checks
GET  /api/v1/teacher/dashboard
POST /api/v1/teacher/materials
PUT  /api/v1/teacher/courses/{course_id}/tutor-policy
```

HTML compatibility routes remain available:

```text
GET  /student/submissions
POST /student/submit
GET  /faculty/summary
GET  /faculty/missing
```

They are not linked from the co-learning product UI.

## Production follow-up

Before deployment, replace the demo accounts, add a bounded source-ingestion adapter, connect a real model provider behind `LearningAgentGateway`, and expand the optional understanding checker with production rubric/provider versioning. Add rate limits, provider timeouts, and privacy controls as part of that integration. Course enrollment and teacher ownership checks are implemented in the current slice. The agent planner and formative-checker run through the learning workflow and write execution records to SQLite; the configured tutor and check providers are deterministic demo adapters. The original requirement document should remain unchanged as the source requirement record.
