# colearning-agent

Go scaffolding for a human-AI co-learning agent with an n-layered architecture and HTMX-enabled dual-role portal.

## Project documentation

- Problem statement: `/docs/project-problem-statement.md`
- Architecture & Mermaid Diagrams: `/docs/project-architecture.md`

## Project structure

```text
src/
  cmd/api/                     # entrypoint only
  adapter/                     # concrete integrations (memory/db/ai/etc.)
  bootstrap/                   # dependency wiring and startup composition
  config/                      # runtime configuration
  controller/                  # HTTP request orchestration
  domain/                      # core entities and business rules
  repository/                  # abstraction interfaces for persistence reads/writes
  routes/                      # route registration + middleware
  service/                     # evaluation and feedback services
  templates/                   # server-rendered HTML templates and HTMX partials
  usecase/                     # application workflows
  view/                        # template rendering helpers
planning/                      # lightweight non-code context for agentic workflows
docs/                          # project-level architecture and problem statement
```

## Quickstart

### Local Development

```bash
go test ./...
go run ./src/cmd/api
```

### Docker & Containerization

Build and run using Docker:

```bash
# Build Docker image
docker build -t colearning-agent:latest .

# Run Docker container
docker run -p 8080:8080 --name colearning-agent colearning-agent:latest
```

Or run using Docker Compose:

```bash
# Start containerized application
docker compose up -d

# Check service status and health
docker compose ps

# View logs
docker compose logs -f

# Stop service
docker compose down
```

## Portal routes

- `GET /healthz`
- `GET /student?role=student&student_id=S-001`
- `GET /student/submissions?role=student&student_id=S-001` (HTMX partial)
- `POST /student/submit?role=student`
- `GET /faculty?role=faculty`
- `GET /faculty/summary?role=faculty` (HTMX partial)
- `GET /faculty/missing?role=faculty` (HTMX partial)

## Planning context folder

Use `/planning` for compact product context (user stories + feature plans) that can
be consumed by humans and AI agents without introducing heavy process overhead.

