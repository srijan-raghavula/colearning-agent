# CoLearn UI Architecture

The UI is a thin presentation layer over the learning use cases. The primary browser surface is the standalone application in `src/frontend`; the server-rendered templates remain compatibility views while the product surface migrates.

## Shell

All primary product pages share:

- a role-specific top navigation bar
- a clear student/teacher role context
- a responsive page container
- shared spacing, color, typography, and button tokens
- JSON-backed browser rendering with explicit loading, empty, and error states
- signed-in account identity with role-aware student and teacher routes
- a profile page with the current account name, email, role, and ID
- a shared file-explorer course shell with role-specific content/actions
- a persisted light/dark theme preference
- an HttpOnly session token reused by every authenticated profile/API request

The primary stylesheet and behavior live at `src/frontend/assets/app.css` and
`src/frontend/assets/app.js`. The older server-rendered stylesheet remains only
for compatibility routes.

## Workspace onboarding

Signup is a staged behavior, not a single large form:

1. choose student or teacher
2. provide name and email
3. provide a password
4. provide role-specific details in two-field pages

Student details are academic program/branch followed by semester/academic year.
Teacher details are department/teaching title. The API validates the complete
role-specific profile before creating the account and sets the session token
immediately.

## Student surfaces

### Dashboard (`/student`)

The first screen answers four questions:

1. What should I continue?
2. How confident am I?
3. What concept comes next?
4. Which lesson should I continue?

Primary regions:

- continue-learning hero
- learning metrics
- ordered concept path
- recent conversation activity
- course enrollment summary
- teacher identity

### Learning space (`/student/workspace`)

The workspace is intentionally transcript-first.

- concept and learning objective
- persistent tutor/student transcript
- source chips on tutor messages
- configurable tutor mode selector
- explicit empty state before the first turn
- turn composer with a normal HTML form fallback
- optional understanding-check action and formative feedback panel
- keyboard-accessible Conversation and Sources & goals tabs
- optional learning brief with the current lesson goal, tutor mode, source provenance, and next action

The UI does not expose raw prompt payloads, evaluator scores, or internal confidence calculations as grades. `ConceptProgress.Confidence` is labeled as a learning signal.

## Teacher surfaces

All teacher pages use the same server-rendered navigation for Class overview, My courses, Content studio, and Enrollment studio. Page-specific actions should not duplicate those destinations unless they provide a distinct task, such as creating a course.

### Class dashboard (`/teacher`)

The teacher view emphasizes teaching opportunities rather than gradebook administration.

- active learners
- concepts published
- average concept confidence
- support queue
- recent learning evidence
- content health
- current tutor policy

The SPA teacher dashboard intentionally keeps overview and course setup in one
workspace. Course setup publishes teacher materials and updates tutor policy
through the versioned API instead of linking to unfinished duplicate pages.

### Content studio (`/teacher/content`)

The content studio is the teacher control plane for the first slice.

- published material list
- source kind and provenance
- version labels
- external references
- publish material form
- default tutor mode
- allowed tutor modes
- external-source policy

### Courses and enrollment (`/teacher/courses`, `/teacher/enrollments`)

- teacher-owned course creation and editing
- curriculum editing for each course
- student enrollment request review

## Required states

Every future UI state should be represented in the API/read model and then rendered consistently:

- loading
- empty course/context
- no active session
- no turns yet
- tutor provider unavailable
- source not ready
- external sources disabled
- understanding check unavailable
- understanding check failed
- validation error
- authorization error

## API/UI boundary

Templates consume use-case read models. The JSON API maps those results through explicit versioned DTOs in `src/controller/api_dto.go`, so internal domain changes do not silently alter the v1 response shape. Business calculations such as concept aggregation, session ownership, progress updates, and policy validation stay outside templates.

When a read model changes, update the API documentation and UI view model together rather than embedding calculations in HTML.

## Accessibility and responsive baseline

- semantic headings and form labels
- keyboard-focusable controls
- visible error banners
- no color-only status communication
- responsive grids collapse to one column on small screens
- source and role context remains visible in the page shell
- reduced visual density is preferred over hiding learning state
