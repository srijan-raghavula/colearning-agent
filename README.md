# colearning-agent

CoLearn is a teacher-grounded human–AI co-learning application. The primary student experience is a configurable tutor conversation grounded in teacher-provided concepts and labeled source material. Automated evaluation remains an optional formative capability and the original assignment routes are retained for compatibility.

## Product documentation

- Original copied requirement: [`docs/project-problem-statement.md`](docs/project-problem-statement.md) — intentionally preserved verbatim
- Target architecture and Mermaid diagrams: [`docs/project-architecture.md`](docs/project-architecture.md)
- Versioned JSON API: [`docs/api.md`](docs/api.md)
- UI architecture: [`docs/ui-architecture.md`](docs/ui-architecture.md)
- UI completion audit and follow-up boundary: [`docs/ui-completion-audit.md`](docs/ui-completion-audit.md)
- User flow: [`planning/userflow-mermaid.md`](planning/userflow-mermaid.md)
- User stories: [`planning/user-stories.md`](planning/user-stories.md)
- Feature plans: [`planning/features/`](planning/features/)

## Application capabilities

The application includes:

- one seeded teacher-owned course
- four ordered concepts with objectives and provenance-aware materials
- resumable student learning sessions and tutor turns
- guided, Socratic, direct, practice, hint, and diagnostic modes
- configurable OpenAI-compatible tutor, formative-check, and curriculum gateways with a deterministic local fallback
- concept progress and next-action read models
- polished student dashboard and learning space
- teacher class dashboard and content studio
- server-rendered HTML and versioned JSON API surfaces

The primary product UI is now a standalone JavaScript frontend. It is served as
an application shell from `src/frontend`, authenticates through the JSON
session endpoints and HttpOnly session token, and renders student and teacher
workspaces from `/api/v1`. Both roles use the same file-explorer layout,
profile surface, course context picker, and persisted light/dark theme; role
customizations are applied inside that shared shell.
The older template controllers remain available internally for compatibility
and migration safety, but are no longer the primary browser route.

Accounts, course content, learning sessions, turns, progress, formative checks, and agent execution records use SQLite. Existing learning snapshot files are imported idempotently when SQLite is missing records. Signed account identity controls student and teacher route access, including course ownership and enrollment checks.

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
  frontend/                    # standalone SPA shell, browser JavaScript, and CSS
  templates/                   # legacy server-rendered views retained for compatibility
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
- `http://localhost:8080/app/student` — standalone JavaScript student app
- `http://localhost:8080/app/teacher` — standalone JavaScript teacher app
- `http://localhost:8080/login` — sign in with the demo student or teacher account
- `http://localhost:8080/signup` — create a persistent student or teacher account
- `http://localhost:8080/profile` — view the signed-in account

The demo accounts are `student@demo.edu` / `student123` and `teacher@demo.edu` / `teacher123`. Role access comes from the signed-in account; query parameters do not select a user or role.

Set `COLEARN_JWT_SECRET` to a stable random value of at least 32 bytes before deployment so signed-in sessions survive restarts. Set `COLEARN_COOKIE_SECURE=true` when TLS terminates at a trusted reverse proxy.

Set `COLEARN_DEMO_ACCOUNTS=false` before deployment to disable the seeded demo passwords. The demo account records remain as course references but receive fresh random passwords at startup.

### AI provider

The application loads `.env` from the project root without replacing environment variables already set by the process. Put the Groq provider key in the canonical `API_KEY=...` variable; the older `COLEARN_AI_API_KEY` name remains a compatibility fallback. A Groq key is routed to Groq’s OpenAI-compatible endpoint with the current `openai/gpt-oss-120b` model default. Other keys use OpenAI’s endpoint and `gpt-4o-mini` by default. Set `COLEARN_AI_PROVIDER`, `COLEARN_AI_BASE_URL`, and `COLEARN_AI_MODEL` to override those defaults for another compatible provider. `COLEARN_AI_TIMEOUT_SECONDS` defaults to 45. Without a key, local startup uses the deterministic stub adapter. Provider failures are returned without exposing provider response bodies to students. Never commit `.env`; Docker Compose receives values from its normal `.env` interpolation. See [`.env.example`](.env.example).

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
GET  /api/v1/auth/profile
PUT  /api/v1/auth/profile
GET  /api/v1/student/dashboard
GET  /api/v1/student/courses/{course_id}/curriculum
POST /api/v1/student/sessions
GET  /api/v1/student/sessions/{session_id}
POST /api/v1/student/sessions/{session_id}/turns
POST /api/v1/student/sessions/{session_id}/understanding-checks
GET  /api/v1/teacher/dashboard
GET  /api/v1/teacher/courses/{course_id}/curriculum
POST /api/v1/teacher/courses
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

Before deployment, disable seeded demo passwords, configure a stable JWT secret and HTTPS cookies, and add operational controls such as rate limits, backups, monitoring, and privacy/retention policies. The app supports configured model-backed tutor, formative-check, and curriculum workflows; without a key it uses deterministic local adapters. Course enrollment and teacher ownership checks are enforced in the application. The original requirement document remains unchanged as the source requirement record.
