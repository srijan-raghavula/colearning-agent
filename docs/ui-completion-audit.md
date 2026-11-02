# UI completion audit

## Scope

This audit covers the browser application served from `/app`, which is the
primary product surface. The server-rendered pages under `/student` and
`/teacher` remain compatibility routes and are not the source of the
standalone application shell.

## Findings and fixes

| Area | Finding | Resolution |
|---|---|---|
| Landing page | Signed-out navigation and hero repeated the same sign-in and account-creation actions. | The shell no longer renders guest actions; the landing page owns exactly two entry actions: **Create your workspace** and **Sign in**. |
| Role navigation | Teacher sessions displayed student links and a teacher link that all led to overlapping or missing screens. | Navigation is role-specific. Teachers now have **Class overview** and **Course setup**; students retain **Overview** and **Learning space**. |
| Teacher dashboard | The SPA only rendered read-only metrics, while its navigation implied more complete pages. | The dashboard now has a single coherent overview plus an in-page Course setup section. |
| Teacher materials | The existing publish-material API was not reachable from the SPA. | Course setup includes a validated publish form and refreshes the dashboard after success. |
| Tutor policy | The existing policy API was not reachable from the SPA. | Course setup includes a policy form wired to the versioned API with inline errors. |
| Multiple relationships | Dashboard APIs returned only the currently inferred course, so a learner with several teachers or a teacher with several courses had no reliable way to switch context. | Student and teacher dashboard responses now include available relationships, and both SPA dashboards provide a context picker that reloads the selected course. |
| Course outline affordances | Student module/chapter headings were static visual bars and lesson rows were not actionable. | Modules and chapters are now keyboard-accessible collapsible sections; each lesson links directly into its learning space. |
| Student dashboard actions | The recommended lesson and every learning-path row launched the same session flow. | The recommended lesson remains the single primary launch; the learning path is now a progress summary rather than a second button set. |
| Learning space navigation | The global Overview link and a second page-header Overview button repeated the same destination. | Removed the page-header duplicate and kept the persistent global navigation. |
| Teacher setup with no concepts | Material/policy forms appeared before a course outline existed, leaving disabled or purposeless controls. | Setup now explains the prerequisite and links directly to the existing Curriculum Studio. |
| Empty teacher state | A teacher without a course received a broken-looking zero-value dashboard. | The UI now explains that a course is required before setup can continue. |
| Feedback states | Teacher mutations had no local loading or error presentation. | Submit buttons disable during requests and errors render beside the relevant form; successful changes show a transient confirmation. |

## Remaining product boundary

Course creation and curriculum authoring still use the compatibility teacher
routes because the versioned SPA API does not expose those operations yet. They
should be added as explicit versioned APIs before introducing more dashboard
navigation. Until then, the SPA does not advertise those unfinished screens as
if they were complete.

## Verification

Run:

```bash
GOCACHE=/tmp/colearning-go-cache go test ./...
GOCACHE=/tmp/colearning-go-cache go vet ./...
git diff --check
```
