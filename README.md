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
- HTML/HTMX and versioned JSON API surfaces

The application currently uses thread-safe in-memory repositories and a demo role/student identity. These are visible seams for replacement, not production security mechanisms.

## Project structure

```text
src/
  cmd/api/                     # entrypoint only
  adapter/memory/              # in-memory repositories and deterministic gateways
  bootstrap/                   # dependency wiring and demo seed data
  config/                      # runtime configuration
  controller/                  # HTML and JSON request orchestration
  domain/                      # learning and legacy assessment entities
  repository/                  # persistence and provider ports
  routes/                      # route registration and demo role middleware
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
- `http://localhost:8080/student?role=student&student_id=S-001` — student dashboard
- `http://localhost:8080/student/workspace?role=student&student_id=S-001` — learning space
- `http://localhost:8080/teacher?role=teacher` — teacher dashboard
- `http://localhost:8080/teacher/content?role=teacher` — content studio

The demo accepts `role=teacher` or `role=faculty` for teacher routes while the codebase transitions from the legacy faculty vocabulary.

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

Before deployment, replace the demo identity and in-memory adapters, add authenticated actor ownership checks, add a bounded source-ingestion adapter, connect a real model provider behind `LearningAgentGateway`, and expand the optional understanding checker with production rubric/provider versioning. The original requirement document should remain unchanged as the source requirement record.
