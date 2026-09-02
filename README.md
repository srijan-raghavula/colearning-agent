# colearning-agent

Go scaffolding for a human-AI co-learning agent with an n-layered, modular structure.

## Project structure

```text
cmd/
  api/                      # entrypoint
internal/
  application/              # use-cases and business orchestration
  domain/                   # core entities and rules
  infrastructure/           # external adapters (db, files, services)
  interfaces/               # delivery layer (http, cli, etc.)
planning/                   # lightweight non-code context for agentic workflows
```

## Quickstart

```bash
go test ./...
go run ./cmd/api
```

The API exposes a basic health endpoint at `GET /healthz`.

## Planning context folder

Use `/planning` for compact product context (user stories + feature plans) that can
be consumed by humans and AI agents without introducing heavy process overhead.