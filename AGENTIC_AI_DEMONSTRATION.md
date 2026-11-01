# CoLearn agent workflow

## Live tutor workflow

Every student tutor turn passes through `agentic.Gateway` and `agentic.TutorAgent` before reaching the configured learning gateway. The tutor plans around the selected mode, course policy, objective, progress and recent turns. It executes three read-only context tools over the authorized request data:

- `read_course_context`
- `read_student_progress`
- `read_published_materials`

The materials tool filters out unpublished material and external sources when course policy disallows them. The service validates the provider response, bounds confidence changes and filters returned source citations against published course material before the use case stores the tutor turn.

## Formative collaboration

The `FormativeChecker` gateway runs only when a student explicitly requests an understanding check. Its output is formative feedback, not a grade. The learning use case then writes evidence and updates concept progress after a tutor response or check. Those writes are application workflow responsibilities rather than separate autonomous agents.

## Execution records

Tutor and check executions are recorded in SQLite's `agent_logs` table with session ID, plan/result data, selected tools and timestamp. `GET /api/agentic/tutor-plan` describes the configured workflow; it does not return a fabricated per-turn plan. Internal execution reasoning is not exposed to student or teacher pages.

## Provider boundary and current limit

The current tutor and formative-check providers are deterministic demo adapters. The orchestration, policy filtering, response validation and execution logging run in the app, but this is not a production language-model connection. A real provider must be added behind the existing repository gateway interfaces with request deadlines, provider timeouts, rate limits and privacy controls.
