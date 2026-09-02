# Feature Plan: P8-001 Automated Evaluation Engine

## Objective
Build a service that evaluates student submissions and computes topic/subject mastery percentages.

## Linked Stories
- STU-002
- FAC-002

## Scope
- Input model for submissions and topic scores.
- Mastery percentage calculation service.
- Feedback payload shape for student-facing diagnostics.

## Out of Scope (Initial Slice)
- File content NLP grading.
- LMS integrations.
- Advanced analytics visualizations.

## Acceptance Criteria
- Service computes mastery percentage with bounds handling.
- Evaluation output includes topic scores and feedback slots.
- Feature can be called by faculty and student-facing interfaces.

## Technical Notes
- Place domain models in `internal/domain`.
- Place orchestration/use-case logic in `internal/application`.
- Keep adapters in `internal/infrastructure`.
