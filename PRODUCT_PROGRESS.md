# CoLearn product work tracker

This file records the user-facing work requested for the authenticated student and teacher areas. Work is delivered in small slices; each slice is marked complete only after its relevant tests or route checks pass.

## Route and navigation audit

- [x] Inventory registered HTML and API routes against links currently rendered in templates.
- [x] Identify malformed student concept links: `/student/workspace&amp;concept_id=...` omitted the query delimiter and could open the wrong concept.
- [ ] Add route-level coverage for every navigation destination and form action; report any intentionally retired legacy destinations.

## 1. Shared account menu and profile entry point

- [x] Replace duplicate “Profile” and “Account” links with one keyboard-accessible user menu on student and teacher pages.
- [x] Show the signed-in person’s name and initials; explain tutor mode separately from identity.
- [x] Provide “View profile” and “Sign out” actions in the menu.
- [x] Verify menu markup on student and teacher pages and verify a concept link opens its requested concept.
- [x] Fix student concept links and verify they open the requested concept.
- [ ] Browser-check hover, keyboard, and narrow-screen behavior (automated route tests currently cover rendered links and actions).

## 2. Full profile and account settings

- [x] Persist and edit student program, branch, semester, and academic year; persist and edit teacher department.
- [x] Restrict profile updates to the signed-in account and refresh the current token after a name change.
- [x] Extend profile details with current course enrollments and teacher/course information.
- [x] Display enrolled courses with teacher information and course-specific details.
- [x] Add credential changes with validation and safe session revocation.
- [x] Verify profile field persistence and name refresh in the authenticated integration flow.

## 3. Course discovery and enrollment

- [x] Support student-to-course enrollment records with one request per course (many-to-many structure).
- [x] Add catalog/details pages with teacher identity and course description.
- [x] Add student enrollment requests and teacher approve/deny actions.
- [x] Enforce approval before opening a course learning session or using its learning APIs.
- [x] Verify pending access is rejected, teacher decisions update state, and approved courses appear in the student profile.
- [x] Add course selection to the student learning dashboard for learners enrolled in more than one course.

## 4. Teacher course and curriculum studio

- [x] Add teacher-owned course creation and editing.
- [x] Add curriculum hierarchy: modules → chapters → lessons → lesson content.
- [x] Support manual creation and later editing of each curriculum level and content item.
- [x] Add clear empty states and exercise course studio routes and forms in integration tests.

## 5. Agent-assisted authoring

- [x] Add a bounded curriculum generation action for modules, chapters, lessons, and lesson content.
- [x] Keep generated content as an editable draft; require teacher review before publishing.
- [x] Record generation actions and make the demo provider capability explicit in the UI.
- [x] Verify deterministic demo-provider output and draft state in integration tests.
- [ ] Connect a production model provider and add provider failure handling.

## Current implementation slice

- Status: requested account, enrollment, course studio, and curriculum-authoring foundations are implemented. Follow-up gaps are listed below.
- Completed: shared account menu, removal of duplicate profile/account navigation, clearer tutor-style label, repaired concept links, persisted study profile fields, authenticated profile editing, multi-course catalog/details, teacher review of enrollment requests, and approval-gated learning access. Teachers can create/edit courses and manually create/edit modules, chapters, lessons, and content. The curriculum demo agent creates unpublished drafts and records generation actions.
- Ownership: legacy learning material publish/policy actions now verify that the signed-in teacher owns the referenced course; bootstrap fails closed if the ownership dependency is unavailable.
- Verification: `GOCACHE=/tmp/colearning-go-cache go test ./...`, `GOCACHE=/tmp/colearning-go-cache go vet ./...`, and `git diff --check` pass. Integration coverage exercises account menus, profile persistence, course requests and decisions, pending access rejection, curriculum authoring/edit/publish, generated draft state, and course selection.
- Remaining: browser-level hover, keyboard, and narrow-screen checks; an actual production model provider and its failure handling; complete automated audit of every rendered link and form destination.

## Follow-up: teacher material and synchronized course views

- [x] Accept blank module/chapter descriptions and lesson summaries when those fields are optional; invalid curriculum forms return to the editor with an inline error.
- [x] Count approved enrollments from the enrollment table in live course reads, so student and teacher course cards show current learner totals.
- [x] Make an unqualified student Learning space link open the first approved course; send students with no approved course to the catalog.
- [x] Show all of a student's course enrollments on Overview with status, teacher identity, and direct course actions.
- [x] Add teacher academic title and department profile fields and show them in student course discovery/details.
- [x] Add course switching to teacher overview and content studio, scoped to courses owned by the signed-in teacher.
- [x] Keep authenticated student and teacher views out of browser caches and keep top navigation available at narrow widths.
- [x] Integration coverage verifies optional chapter descriptions, workspace navigation, approval-derived learner counts, title persistence, and multi-course Overview.
- Verification: `GOCACHE=/tmp/colearning-go-cache go test ./...`, `GOCACHE=/tmp/colearning-go-cache go vet ./...`, and `git diff --check` pass.
- Remaining: visual browser review is still needed for responsive layout and tab highlighting; teacher directory is represented through teacher details on each course card, not a separate directory page.

## Follow-up: live rendering and policy-aware student controls

- [x] Apply `Cache-Control: no-store` centrally to rendered HTML so every student, teacher, and profile page reflects the latest repository state after navigation.
- [x] Rebuild the refreshed student conversation view from its active session's course, avoiding a default-course read model in multi-course sessions.
- [x] Show only tutor modes allowed by the current course policy and recover to an allowed mode if a teacher changed policy while a session was open.
- [x] Cover rendered-page cache headers and tutor-mode policy selection with automated tests.
- Verification: `GOCACHE=/tmp/colearning-go-cache go test ./...`, `GOCACHE=/tmp/colearning-go-cache go vet ./...`, and `git diff --check` pass.
- [x] Add saved-node removal to Curriculum Studio. Removed modules, chapters, lessons, and content are archived, removed from the live course, and excluded from tutor context while learning history remains readable.
- [x] Add account password changes with current-password verification, confirmation, and account-wide token revocation.
- Verification: integration flow changes a password, confirms the new credential works, and confirms the old signed session is rejected. It also archives a lesson, checks it disappears from the live curriculum and tutor context, and confirms the existing session remains readable.
- Next usability gaps: browser-level interaction checks and production tutor/check providers.

## Follow-up: student dashboard hierarchy and conversation sides

- [x] Keep one primary overview action for resuming the current conversation or starting the featured lesson; retain individual learning-path links for students choosing a different concept.
- [x] Make course cards lead to course details rather than adding more duplicate learning-space buttons.
- [x] Show a course selector only when at least two courses are approved, and apply the selection as soon as it changes. This removes the misleading no-op-looking `Open` button when pending requests were counted as selectable courses.
- [x] Keep the workspace route tied to the selected course or active session, and remove the redundant Learning space item from student top navigation.
- [x] Place tutor replies on the left and student messages on the right.
- Verification: `GOCACHE=/tmp/colearning-go-cache go test ./...`, `GOCACHE=/tmp/colearning-go-cache go vet ./...`, and `git diff --check` pass. Integration and template tests cover the course selector and message role markup.

## Follow-up: teacher dashboard duplication and application usability audit

- [x] Remove teacher dashboard buttons that duplicate the shared top navigation.
- [x] Make teacher course selectors apply immediately and consistently, retaining a no-JavaScript fallback.
- [x] Render inline course form errors while preserving submitted values.
- [x] Keep lesson goals and source provenance in the student Learning space brief instead of duplicating them on Overview.
- [x] Clarify that the dashboard time metric measures elapsed session span, not active study minutes.
- [x] Document verified fixes, remaining product limitations, and a prioritized follow-up plan in `docs/usability-audit.md`.
- Verification: `GOCACHE=/tmp/colearning-go-cache go test ./...`, `GOCACHE=/tmp/colearning-go-cache go vet ./...`, and `git diff --check`.
