# Usability audit and follow-up plan

This audit covers the rendered student and teacher pages, their navigation and form destinations, the corresponding registered HTML routes, and the existing integration/UI tests. It records verified issues and explicit product boundaries; it is not a claim that every browser/device combination has been visually reviewed.

## Fixed in this pass

- **Teacher overview repeated navigation actions.** The overview showed extra buttons for Content Studio, Enrollment Studio, My courses, source context, and tutor policy even though those destinations are in the top navigation. Removed the duplicates; the dashboard now keeps its course selector as the only page-level control.
- **Course selectors behaved differently.** Content Studio required an “Open” button while the overview changed course immediately. Both now apply a selection on change, with a no-JavaScript submit fallback.
- **Course form validation lost user context.** Invalid title/subject input returned plain text. Validation and save errors now render in the course form and keep entered values; course list and edit pages display their notices/errors.
- **Student overview repeated the learning brief.** Lesson goals and source provenance are most useful beside the active lesson and conversation. Removed the duplicated goal/context cards from Overview; “Sources & goals” in Learning space remains the opt-in location for the current lesson’s goal, tutor mode, and sources.
- **Elapsed time looked like active study time.** Renamed the dashboard metric to “Session span” and clarified that it measures elapsed time between first and most recent activity, not active minutes.
- **Navigation drift and non-responsive source tab.** Teacher pages share one server-rendered navigation partial, including My courses on Enrollment Studio. The student Sources & goals control is an accessible in-page tab, rather than an anchor to a panel already visible beside the conversation.

## Verified coverage

- Teacher HTML routes: overview, course list/create/edit, curriculum, content studio, and enrollment review are registered and linked from the teacher shell.
- Student HTML routes: overview, course catalog/details/enrollment, session workspace, conversation turns, and understanding checks are registered and linked from the student flow.
- Integration tests exercise student enrollment gating, teacher course selection, course creation, curriculum authoring, and live read-after-write views.
- Added checks ensure the teacher overview links each studio destination only once, the content course selector has no inert “Open” button, and invalid course input is preserved with an inline error.

## Prioritized follow-up plan

### P1 — Browser interaction coverage

- Add browser-level checks for keyboard tab navigation, account menus, responsive navigation, course selectors, and the dynamic curriculum editor.
- Review narrow screens and screen-reader behavior in a real browser; current Go tests verify rendered HTML and server behavior, not layout or assistive-technology output.

### P1 — Production learning providers

- Replace the deterministic demo tutor, understanding checker, and curriculum authoring gateway with configurable providers.
- Specify timeouts, retries, unavailable-provider UI, rate limits, and safe fallback behavior before enabling a remote provider.

### P1 — Source lifecycle and safety

- Teacher material publishing currently creates versioned context; the UI does not yet provide a complete edit/archive lifecycle for those materials.
- External references are currently labeled metadata and links, not fetched/indexed source content. Before adding ingestion, validate URL schemes and resolved addresses, block private/local destinations, cap redirects and payload size, and treat retrieved text as untrusted context.

### P2 — Error and route completeness

- Continue replacing any remaining plain-text failures from HTML form submissions with page-level messages that preserve entered values where safe.
- Add systematic tests that extract every template link/form destination and compare it with registered routes, including dynamic path parameters and intentionally legacy endpoints.
- Decide whether the legacy submission/assessment pages remain supported; they are separate from the current conversation-first student product and are intentionally absent from primary navigation.

## Verification

Run `GOCACHE=/tmp/colearning-go-cache go test ./...`, `GOCACHE=/tmp/colearning-go-cache go vet ./...`, and `git diff --check` after each UI or route change. Browser review remains a separate planned check because this repository currently has no browser automation harness.
