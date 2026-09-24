# CoLearn API Contract

This document describes the first versioned JSON contract for the co-learning vertical slice. The HTML portal and the JSON API use the same use cases.

## Conventions

- Base path: `/api/v1`
- Content type: `application/json`
- IDs are opaque strings.
- Timestamps are RFC 3339 UTC values.
- Semantic validation errors currently use `400` in the first slice; a later error taxonomy can distinguish `422` without changing the success shapes.
- Errors use the envelope `{ "error": "message" }` in this first slice.
- The current demo accepts `role=student` or `role=teacher` in the query string, or the equivalent `X-Role` header. Production authentication must replace this.
- JSON responses are mapped through explicit DTOs in `src/controller/api_dto.go`; domain refactors do not automatically alter the v1 response shape.

The client-supplied `student_id` is a demo convenience only. The controller resolves the learner from the request identity seam and does not trust a body ownership field. A production identity provider must derive the learner from the authenticated session.

## Student dashboard

```http
GET /api/v1/student/dashboard?role=student&student_id=S-001
```

Returns a `StudentLearningDashboard` containing:

- course and tutor policy
- ordered concept cards
- active session
- recent turns
- published materials and source metadata
- current objective
- learner statistics

## Start or resume a session

```http
POST /api/v1/student/sessions?role=student&student_id=S-001
Content-Type: application/json
```

```json
{
  "course_id": "course-algebra-foundations",
  "concept_id": "linear-equations",
  "mode": "guided"
}
```

`student_id`, `course_id`, `concept_id`, and `mode` are optional for the demo. The use case resolves the current course and active session when they are omitted.

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
GET /api/v1/student/sessions/session-s001-linear?role=student&student_id=S-001
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

The server must verify that the authenticated learner owns the session. The demo uses the request identity seam until authentication is implemented.

## Add a learning turn

```http
POST /api/v1/student/sessions/session-s001-linear/turns?role=student&student_id=S-001
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
GET /api/v1/teacher/dashboard?role=teacher
```

Returns a `TeacherLearningDashboard` containing:

- course and tutor policy
- class concept confidence cards
- source/material health
- support-needed learners
- recent learning evidence

## Publish learning material

```http
POST /api/v1/teacher/materials?role=teacher
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

## Update tutor policy

```http
PUT /api/v1/teacher/courses/course-algebra-foundations/tutor-policy?role=teacher
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
POST /api/v1/student/sessions/session-s001-linear/understanding-checks?role=student&student_id=S-001
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

## Security requirements before production

1. Derive actor ID and role from an authenticated session.
2. Enforce course/teacher ownership for content and policy writes.
3. Enforce session participation for every student read and turn.
4. Validate HTTP(S) source URLs and protect against SSRF, private-address access, excessive redirects, and oversized responses.
5. Treat retrieved source text as untrusted data that cannot override tutor instructions.
6. Add request size limits, rate limits, structured audit events, and provider timeouts.
7. Never accept model-generated URLs as trusted source references without server-side resolution.
