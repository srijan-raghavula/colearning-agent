# Project Architecture

## Architectural Style

The application follows an n-layered architecture with all Go packages under `/src`, preserving clear dependency direction and SOLID boundaries.

Dependency direction:

`routes -> controller -> usecase -> repository interfaces`

`usecase -> service`

`adapter -> repository interfaces`

`domain` remains dependency-free from outer layers.

## Repository Layout

```text
src/
  cmd/api/                     # application entrypoint
  adapter/                     # concrete integrations (in-memory/db/ai providers)
  bootstrap/                   # dependency wiring and application composition
  config/                      # runtime settings
  controller/                  # HTTP request orchestration and IO mapping
  domain/                      # entities, value objects, and business rules
  repository/                  # abstraction interfaces for data access
  routes/                      # route registration and middleware
  service/                     # evaluation and feedback services
  templates/                   # server-rendered HTML + HTMX partial templates
  usecase/                     # application workflows (student/faculty logic)
  view/                        # rendering helpers
planning/                      # feature plans and user stories
docs/                          # project-level documentation
```

## Layer Responsibilities

- **Domain**: Core entities (`Submission`, `Evaluation`, dashboard summaries) and rule-centric state.
- **Repository**: Segregated interfaces for reads/writes to submissions and evaluations.
- **Service**: Mastery computation, topic normalization, and feedback generation.
- **Usecase**: Student and faculty workflows (submission lifecycle, dashboards, analytics aggregation).
- **Adapter**: Infrastructure implementations; currently in-memory repositories.
- **Controller**: HTTP handlers that validate inputs and orchestrate use cases.
- **Routes**: Endpoint registration and role gating middleware.
- **View/Templates**: Server-side HTML rendering with HTMX-based partial refresh for both portals.
- **Bootstrap/Config**: Environment/config loading and dependency assembly.

## Role-Specific Interfaces

- **Student portal**: submission upload, status tracking, marks, feedback timeline.
- **Faculty portal**: completion monitoring, missing/overdue alerts, subject/topic mastery summaries.

Both flows are exposed through HTTP handlers with role-aware access checks.

## HTMX + Template Strategy

- Full-page endpoints render initial Student and Faculty pages.
- HTMX partial endpoints refresh submissions, summaries, and missing-work alerts.
- Templates are kept modular by role and endpoint intent.
