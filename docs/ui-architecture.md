# CoLearn UI Architecture

The UI is a thin presentation layer over the learning use cases. It is polished enough to review as a product surface while keeping the business core small and replaceable.

## Shell

All product pages share:

- a dark top navigation bar
- a clear student/teacher role context
- a responsive page container
- shared spacing, color, typography, and button tokens
- server-rendered HTML with HTMX only for targeted conversation refreshes

The stylesheet lives at `src/templates/assets/app.css` and is served by the application at `/static/assets/app.css`.

## Student surfaces

### Dashboard (`/student`)

The first screen answers four questions:

1. What should I continue?
2. How confident am I?
3. What concept comes next?
4. What context is the tutor using?

Primary regions:

- continue-learning hero
- learning metrics
- ordered concept path
- recent conversation activity
- source/context library
- teacher and tutor-policy summary

### Learning space (`/student/workspace`)

The workspace is intentionally transcript-first.

- concept and learning objective
- persistent tutor/student transcript
- source chips on tutor messages
- configurable tutor mode selector
- explicit empty state before the first turn
- HTMX turn composer
- optional understanding-check action and formative feedback panel
- sticky learning brief with goal, sources, and next action

The UI does not expose raw prompt payloads, evaluator scores, or internal confidence calculations as grades. `ConceptProgress.Confidence` is labeled as a learning signal.

## Teacher surfaces

### Class dashboard (`/teacher`)

The teacher view emphasizes teaching opportunities rather than gradebook administration.

- active learners
- concepts published
- average concept confidence
- support queue
- recent learning evidence
- content health
- current tutor policy

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
