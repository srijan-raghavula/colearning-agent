# Feature Plan: P8-002 Co-Learning Agent Foundation

## Objective
Create a thin, demonstrable learning loop in which a student can resume a concept session, interact with a configurable tutor grounded in teacher context, and produce formative concept progress.

## Linked Stories
- STU-001
- STU-002
- STU-003
- STU-004
- STU-005
- TEA-001
- TEA-002
- TEA-003
- OPT-001

## Scope
- Teacher-owned course, concept, objective, and material model.
- Versioned source references for teacher and external context.
- Resumable learning sessions and persisted student/tutor turns.
- Configurable tutor modes.
- Provider-neutral learning agent gateway.
- Response grounding and bounded confidence policy.
- Concept progress and next-action read models.
- Student dashboard, learning space, teacher dashboard, and content studio.
- Versioned JSON endpoints documented in `docs/api.md`.

## Out of Scope
- Final grading or GPA logic.
- Mandatory assignment submission.
- Real external web retrieval in the first slice.
- Production authentication and authorization.
- Durable database persistence.
- LMS integrations.
- Full authoring workflow for courses, rubrics, or assignments.

## Acceptance Criteria
- A student can open a dashboard and resume an active concept session.
- A student can send a message using an allowed tutor mode.
- A tutor response includes a response kind, grounded source references when available, and a next action.
- Student and tutor turns are persisted in the session.
- Concept progress receives bounded formative evidence and exposes a next action.
- A teacher can publish a versioned teacher or labeled external material item.
- A teacher can update allowed tutor modes and external-source policy.
- Student and teacher dashboards render from the same use-case read models as the API.
- Existing evaluation tests and legacy routes continue to pass.

## Technical Notes
- Keep `domain`, `repository`, `service`, `usecase`, and `controller` boundaries explicit.
- Keep `Submission` and `Evaluation` in the legacy assessment path.
- Put the new provider contract in `repository.LearningAgentGateway`; do not expand the legacy `AIGateway` with conversational fields.
- Treat source text as untrusted data and validate source IDs against the current context.
- Use deterministic in-memory adapters for tests and the demo.
- Derive HTML and JSON DTOs from stable use-case/domain shapes where practical; avoid duplicating business calculations in templates.
- Replace demo role/student identity before deployment.
