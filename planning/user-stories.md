# User Stories

The primary stories describe a teacher-grounded learning application. Assessment stories are retained as optional formative-check capabilities rather than the default student workflow.

## STU-001 - Resume a Learning Path
- **Role**: Student
- **Need**: Continue learning without reconstructing context from an assignment list.
- **Value**: Build continuity and reduce the friction of returning to a subject.
- **Acceptance**:
  - Student sees the current course, concept, confidence, and next action.
  - Student can start or resume a session.
  - The learning thread is restored after navigating away.

## STU-002 - Learn with a Configurable Tutor
- **Role**: Student
- **Need**: Choose how the AI tutor should help.
- **Value**: Learn in a way that matches the student's current need.
- **Acceptance**:
  - Student can choose guided, Socratic, direct, practice, hint, or diagnostic mode.
  - The tutor response identifies its response kind and next action.
  - The mode is validated against the course tutor policy.

## STU-003 - Ask Questions and Request Hints
- **Role**: Student
- **Need**: Ask about a concept or request a progressive hint.
- **Value**: Resolve misunderstandings without waiting for a formal assessment.
- **Acceptance**:
  - Student can send a message in an active session.
  - Tutor responses are grounded in the available teacher and approved external context.
  - Student and tutor turns are persisted in the session.

## STU-004 - Practice and Check Understanding
- **Role**: Student
- **Need**: Practice a concept or request an optional understanding check.
- **Value**: Convert passive explanation into durable understanding.
- **Acceptance**:
  - Student can receive a practice prompt or formative question.
  - An optional check produces learning evidence rather than a final assignment grade.
  - Concept progress and the next suggested action update after interaction.

## STU-005 - View Concept Progress
- **Role**: Student
- **Need**: See which concepts are secure, in progress, or need support.
- **Value**: Make the learning path understandable without exposing a misleading grade.
- **Acceptance**:
  - Dashboard shows concept-level confidence and status.
  - Progress is labeled as a learning signal, not a final mark.
  - Student can open the next recommended concept.

## TEA-001 - Publish Teacher Learning Context
- **Role**: Teacher
- **Need**: Publish concepts, objectives, and learning material for the tutor.
- **Value**: Keep AI explanations aligned with the course the teacher intends.
- **Acceptance**:
  - Teacher can publish a versioned teacher note for a concept.
  - Material includes source kind and provenance.
  - External references are visibly labeled and can be disabled by policy.

## TEA-002 - Configure Tutor Behavior
- **Role**: Teacher
- **Need**: Select allowed tutor modes and source policy.
- **Value**: Preserve pedagogical control while allowing adaptation.
- **Acceptance**:
  - Teacher can choose the default tutor mode.
  - Teacher can choose which modes students may use.
  - Teacher can decide whether external sources are available.

## TEA-003 - Review Learning Signals
- **Role**: Teacher
- **Need**: See concept-level confidence and learning evidence across learners.
- **Value**: Intervene where support is useful instead of managing a gradebook.
- **Acceptance**:
  - Teacher dashboard shows concept confidence, evidence count, and support-needed learners.
  - Teacher can inspect recent learning activity.
  - Teacher can add guidance or adjust the learning context.

## OPT-001 - Request Formative Understanding Check
- **Role**: Student or teacher
- **Need**: Explicitly request feedback on an explanation or practice attempt.
- **Value**: Use evaluation when it helps without making every interaction an assessment.
- **Acceptance**:
  - The operation is explicit and optional.
  - It returns formative evidence and recommended next steps.
  - Failure does not remove the tutor response or learning thread.

## LEGACY-001 - Assignment Evaluation
- **Role**: Student or faculty
- **Need**: Use the existing submission/evaluation workflow.
- **Value**: Preserve compatibility while the co-learning architecture is validated.
- **Acceptance**:
  - Legacy routes remain available.
  - Legacy data is not mixed into learning sessions or concept progress.
  - A future migration must preserve the separate assessment history.
