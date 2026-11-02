package memory

import (
	"context"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/agentic"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

// StubCurriculumAuthoringGateway is a deterministic, provider-neutral draft
// generator for local development. It does not call an external language model.
type StubCurriculumAuthoringGateway struct{}

func NewStubCurriculumAuthoringGateway() *StubCurriculumAuthoringGateway {
	return &StubCurriculumAuthoringGateway{}
}

func (*StubCurriculumAuthoringGateway) GenerateCurriculum(_ context.Context, course domain.LearningCourse) (agentic.CurriculumDraft, error) {
	topic := strings.TrimSpace(course.Title)
	if topic == "" {
		topic = strings.TrimSpace(course.Subject)
	}
	description := strings.TrimSpace(course.Description)
	if description == "" {
		description = "Build understanding through clear explanations, examples, and practice."
	}
	return agentic.CurriculumDraft{
		Provider: "deterministic demo authoring adapter",
		Summary:  "Generated a starter outline with editable, unpublished teaching content.",
		Modules: []agentic.AuthoringModuleDraft{
			{
				Title:   "Foundations of " + topic,
				Summary: "Introduce key language, representations, and prerequisite ideas for " + topic + ".",
				Chapters: []agentic.AuthoringChapterDraft{
					{
						Title:   "Core ideas and vocabulary",
						Summary: "Name the main ideas and connect them to the course goal.",
						Lessons: []agentic.AuthoringLessonDraft{{
							Title:    "Identify the central ideas",
							Summary:  "Explain the core ideas in " + topic + " and describe how they relate.",
							Contents: []agentic.AuthoringContentDraft{{Kind: "lesson_note", Title: "Teacher draft: course foundations", Body: description + " Start by defining the important terms, then connect each term to a concrete example."}},
						}},
					},
					{
						Title:   "Represent the ideas",
						Summary: "Move between words, examples, and visual or symbolic forms.",
						Lessons: []agentic.AuthoringLessonDraft{{
							Title:    "Choose a useful representation",
							Summary:  "Select a representation that makes a relationship in " + topic + " easier to explain.",
							Contents: []agentic.AuthoringContentDraft{{Kind: "activity", Title: "Representation activity draft", Body: "Choose one idea from the course description. Represent it in two different ways and explain what each representation makes clear."}},
						}},
					},
				},
			},
			{
				Title:   "Apply and explain " + topic,
				Summary: "Use worked examples and independent reasoning to apply the course ideas.",
				Chapters: []agentic.AuthoringChapterDraft{
					{
						Title:   "Worked examples",
						Summary: "Study a complete example and explain why each step is valid.",
						Lessons: []agentic.AuthoringLessonDraft{{
							Title:    "Explain a worked example",
							Summary:  "Connect the steps of an example to the underlying course ideas.",
							Contents: []agentic.AuthoringContentDraft{{Kind: "example", Title: "Worked example draft", Body: "Write one representative problem for " + topic + ". Show each step, explain the reason for the step, and finish by checking that the result fits the original question."}},
						}},
					},
					{
						Title:   "Independent practice and reflection",
						Summary: "Transfer understanding to a new problem and reflect on the reasoning.",
						Lessons: []agentic.AuthoringLessonDraft{{
							Title:    "Practice and reflect",
							Summary:  "Solve a new problem and justify the method you chose.",
							Contents: []agentic.AuthoringContentDraft{{Kind: "activity", Title: "Practice prompt draft", Body: "Create a new problem about " + topic + " that changes one feature of the worked example. Solve it, explain the choice of method, and check the answer."}},
						}},
					},
				},
			},
		},
	}, nil
}
