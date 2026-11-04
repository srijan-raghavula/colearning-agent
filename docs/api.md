# CoLearn API Contract

This document describes the first versioned JSON contract for the co-learning vertical slice. The HTML portal and the JSON API use the same use cases.

## Conventions

- Base path: `/api/v1`
- Content type: `application/json`
- IDs are opaque strings.
- Timestamps are RFC 3339 UTC values.
- Semantic validation errors currently use `400` in the first slice; a later error taxonomy can distinguish `422` without changing the success shapes.
- Errors use the envelope `{ "error": "message" }` in this first slice.
- Authenticate with `POST /api/auth/login` or `POST /api/auth/signup`. The server sets an HttpOnly session cookie; send that cookie with these requests. Query parameters and `X-Role` do not grant identity.
- JSON responses are mapped through explicit DTOs in `src/controller/api_dto.go`; domain refactors do not automatically alter the v1 response shape.

The controller derives the learner ID and role from the signed session and ignores body/query ownership fields.

## Account endpoints

```http
POST /api/auth/login
POST /api/auth/signup
GET  /profile
POST /profile/password
GET  /logout
GET  /api/v1/auth/profile
PUT  /api/v1/auth/profile
```

Login and signup set the `auth_token` HttpOnly cookie. The browser UI redirects to the workspace for the account role. The profile page shows the account name, email, role, and account ID. Its password-change form verifies the current password, checks the new password confirmation, and revokes previously issued sessions for the account.

Signup requires the shared identity fields (`role`, `name`, `email`, and
`password`) plus the complete role-specific profile. Students provide
`academic_program`, `branch`, `semester`, and `academic_year`; teachers provide
`department` and `teaching_title`. The browser collects these fields in staged
two- or three-field steps, while the API accepts one final JSON request.

The versioned profile endpoint uses one shared user envelope for both roles:

```json
{
  "user": {
    "id": "S-001",
    "name": "Alex Morgan",
    "email": "student@demo.edu",
    "role": "student",
    "auth_version": 1,
    "profile": {
      "academic_program": "BSc Computer Science",
      "branch": "Systems",
      "semester": 3,
      "academic_year": "2026"
    }
  }
}
```

Teacher profiles use the same envelope and replace the role-specific fields
with `department` and `teaching_title`. `PUT` accepts the same `user` profile
fields as a flat JSON object, refreshes the session cookie, and returns the
updated envelope.

## Student dashboard

```http
GET /api/v1/student/dashboard
```

Returns a `StudentLearningDashboard` containing:

- course and tutor policy
- ordered concept cards
- active session
- recent turns
- published materials and source metadata
- current objective
- learner statistics

The browser client can load the selected course notebook separately:

```http
GET /api/v1/student/courses/{course_id}/curriculum
GET /api/v1/teacher/courses/{course_id}/curriculum
```

The response is a nested `course_id` plus `modules`, `chapters`, `lessons`,
and `contents` tree. Student callers must have an approved enrollment and
receive published lesson content only. Teacher callers must own the course
and receive drafts as well as published content.

## Start or resume a session

```http
POST /api/v1/student/sessions
Content-Type: application/json
```

```json
{
  "course_id": "course-algebra-foundations",
  "concept_id": "linear-equations",
  "mode": "guided"
}
```

`course_id`, `concept_id`, and `mode` are optional. The authenticated student ID always comes from the session, never from the request body. The use case resolves the current course and active session when the other fields are omitted.

Successful response: `201 Created`

```json
{
  "id": "session-...",
  "student_id": "S-001",
  "course_id": "course-algebra-foundations",
  "concept_id": "linear-equations",
  "mode": "guided",
  "status": "active",
  "started_at": "2026-09-24T06:17:13Z",
  "last_active_at": "2026-09-24T06:17:13Z"
}
```

The operation is idempotent in the sense that an existing active session for the same student, course, and concept is resumed rather than duplicated.

## Read a session

```http
GET /api/v1/student/sessions/session-s001-linear
```

Response: `200 OK`

```json
{
  "session": { "id": "session-s001-linear", "status": "active" },
  "turns": [
    {
      "role": "tutor",
      "kind": "explanation",
      "body": "...",
      "sources": []
    }
  ]
}
```

The server verifies that the authenticated learner owns the session.

## Add a learning turn

```http
POST /api/v1/student/sessions/session-s001-linear/turns
Content-Type: application/json
```

```json
{
  "mode": "socratic",
  "message": "How do I know my answer is correct?"
}
```

Successful response: `200 OK`

```json
{
  "session": {
    "id": "session-s001-linear",
    "status": "active"
  },
  "turn": {
    "id": "turn-...",
    "session_id": "session-s001-linear",
    "role": "tutor",
    "kind": "question",
    "mode": "socratic",
    "body": "Let’s make the idea visible...",
    "sources": [
      {
        "id": "material-linear-teacher",
        "title": "Teacher note: isolate, then verify",
        "kind": "teacher"
      }
    ]
  },
  "progress": {
    "concept_id": "linear-equations",
    "status": "learning",
    "confidence": 71,
    "evidence_count": 5,
    "next_action": "Answer in your own words"
  },
  "next_action": "Answer in your own words"
}
```

The operation persists both the student message and the tutor response before returning. A provider failure returns an error; the already persisted student turn remains available for retry.

## Teacher dashboard

```http
GET /api/v1/teacher/dashboard
```

Returns a `TeacherLearningDashboard` containing:

- all courses owned by the authenticated teacher in `courses`
- course and tutor policy
- class concept confidence cards
- source/material health
- support-needed learners
- recent learning evidence

Pass `course_id` as a query parameter to load a specific owned course.

The student dashboard also returns enrolled course choices in `courses`,
including teacher identity and enrollment status, so clients can switch among
approved courses without guessing from a single default.

## Course curriculum

```http
GET /api/v1/student/courses/{course_id}/curriculum
GET /api/v1/teacher/courses/{course_id}/curriculum
```

Returns the nested course notebook structure: modules contain chapters,
chapters contain lessons, and lessons contain content items. Student responses
include published content only and require approved enrollment; teacher
responses include drafts and require course ownership. This stable tree is
intended for sidebar clients and future agent/tool integrations.

## Create a teacher course

```http
POST /api/v1/teacher/courses
Content-Type: application/json
```

```json
{
  "title": "Algebra foundations",
  "subject": "Mathematics",
  "description": "Build confidence with equations and functions."
}
```

Creates a course owned by the authenticated teacher and returns the course
object used by the teacher dashboard.

## Publish learning material

```http
POST /api/v1/teacher/materials
Content-Type: application/json
```

```json
{
  "course_id": "course-algebra-foundations",
  "concept_id": "linear-equations",
  "title": "A new teacher note",
  "body": "Use the relationship as your anchor, then verify the result.",
  "source_kind": "teacher",
  "source_ref": ""
}
```

For an external source, provide `source_kind: "external"` and an HTTP(S) `source_ref`. The first slice stores and labels the reference; a production source-ingestion adapter must fetch, normalize, and index the content under the same contract.

Successful response: `201 Created` with the published `LearningMaterial`.

## Upload PDF learning material

```http
POST /api/v1/teacher/materials/pdf
Content-Type: multipart/form-data
```

Form fields are `course_id`, `concept_id`, and optional `title`; upload the
teacher source as `file`. PDF files are limited to 10 MB, text is extracted
locally, and the resulting published material is available to the grounded
tutor with the original filename as its source reference.

## Run the student learning agent

```http
POST /api/v1/student/agent/next
Content-Type: application/json
```

```json
{
  "course_id": "course-algebra-foundations",
  "session_id": "optional-existing-session",
  "mode": "guided"
}
```

The endpoint selects or resumes the student's current learning session,
grounds the tutor in course materials and progress, and returns the next
agent-generated learning action using the same turn and progress contract as
the interactive tutor.

## Update tutor policy

```http
PUT /api/v1/teacher/courses/course-algebra-foundations/tutor-policy
Content-Type: application/json
```

```json
{
  "default_mode": "guided",
  "allowed_modes": ["guided", "socratic", "practice"],
  "allow_external_sources": true,
  "source_policy": "Teacher material leads; external sources are labeled."
}
```

Successful response: `200 OK` with the resulting `TutorPolicy`.

The default mode must be included in `allowed_modes`. An empty `allowed_modes` list is normalized to the full built-in mode set for the demo.

## Optional understanding check

```http
POST /api/v1/student/sessions/session-s001-linear/understanding-checks
```

This operation is explicit and optional. It evaluates the most recent student explanation or practice turn, stores formative evidence, and returns a check result without creating an assignment or final grade. Repeating the request for the same latest turn returns the existing check rather than creating a duplicate.

Successful response: `201 Created`

```json
{
  "id": "check-...",
  "session_id": "session-s001-linear",
  "turn_id": "turn-...",
  "summary": "This is formative feedback on your current explanation, not a final grade.",
  "feedback": ["You have named a useful starting point."],
  "evidence": ["The response names a relevant relationship or approach."],
  "next_action": "Revise the explanation, then ask the tutor for another check"
}
```

The same operation is available as `POST /student/check` for the HTML workspace.

## Future contract seams

The following operations are intentionally not part of the first slice but have clear extension points:

```text
GET  /api/v1/student/progress
GET  /api/v1/teacher/sources
POST /api/v1/teacher/sources
```

The current check is intentionally lightweight. A production implementation can add rubric/provider versioning and more explicit evidence types without changing the learning-session contract.

## Remaining production work

1. Enforce course enrollment and teacher ownership for content and policy writes.
2. Validate HTTP(S) source URLs and protect against SSRF, private-address access, excessive redirects, and oversized responses.
3. Treat retrieved source text as untrusted data that cannot override tutor instructions.
4. Add rate limits and provider timeouts.
5. Never accept model-generated URLs as trusted source references without server-side resolution.

## Model provider configuration

The tutor agent can use any OpenAI-compatible `/chat/completions` endpoint:

```bash
COLEARN_AI_PROVIDER=openai-compatible
COLEARN_AI_API_KEY=...
COLEARN_AI_MODEL=gpt-4o-mini
COLEARN_AI_BASE_URL=https://api.openai.com/v1
COLEARN_AI_TIMEOUT_SECONDS=45
```

The default provider remains `stub` for offline development. The provider response must be JSON matching the tutor contract; application guardrails filter unsupported kinds, sources, unsafe tone, and confidence deltas before persistence.
