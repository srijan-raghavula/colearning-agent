package agentic

import (
	"context"
	"fmt"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// TutorAgent plans a grounded turn and executes read-only context tools before
// calling the configured learning gateway.

type AgentPlan struct {
	Steps     []string `json:"steps"`
	Reasoning string   `json:"reasoning"`
}

type ToolCall struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args"`
	Output map[string]any `json:"output,omitempty"`
}

type AgentResult struct {
	AgentName  string         `json:"agent_name"`
	Plan       AgentPlan      `json:"plan"`
	Actions    []string       `json:"actions"`
	Reasoning  string         `json:"reasoning"`
	ToolsUsed  []ToolCall     `json:"tools_used"`
	Results    map[string]any `json:"results"`
	NextAction string         `json:"next_action"`
}

// TutorAgent plans a turn from student state and selects bounded, read-only
// tools over the context already authorized by the application.
type TutorAgent struct {
	Gateway repository.LearningAgentGateway
}

func (a *TutorAgent) PlanResponse(req repository.LearningRequest) AgentPlan {
	plan := AgentPlan{
		Steps: []string{
			"Read current session context",
			"Read concept progress for confidence signal",
			"Select tutor mode from policy",
			"Build grounded request with published materials",
			"Generate response via gateway",
			"Validate response kind and sources",
			"Compute next action from response and progress",
		},
		Reasoning: fmt.Sprintf("Student %s is working on %s in mode %s. Course policy allows %v. Published materials count: %d. Recent turns: %d. The agent will select an explanation aligned with the objective and adapt based on progress.",
			req.StudentID, req.Concept.Title, req.Mode, req.Policy.AllowedModes, len(req.Materials), len(req.RecentTurns)),
	}
	return plan
}

func (a *TutorAgent) ExecuteTurn(ctx context.Context, req repository.LearningRequest) (repository.LearningResponse, AgentResult, error) {
	if a == nil || a.Gateway == nil {
		return repository.LearningResponse{}, AgentResult{}, fmt.Errorf("learning gateway is not configured")
	}
	plan := a.PlanResponse(req)

	// These read-only tools operate only on the authorized, preloaded request
	// context. The agent can filter context but cannot choose arbitrary records.
	grounded := req
	grounded.Materials = publishedMaterials(req.Materials, req.Policy.AllowExternalSources)
	tools := []ToolCall{
		{
			Name:   "read_course_context",
			Args:   map[string]any{"course_id": req.CourseID, "concept_id": req.ConceptID},
			Output: map[string]any{"course_title": req.CourseTitle, "concept_title": req.Concept.Title, "objective": req.Objective.Description},
		},
		{
			Name:   "read_student_progress",
			Args:   map[string]any{"student_id": req.StudentID, "course_id": req.CourseID, "concept_id": req.ConceptID},
			Output: map[string]any{"status": req.Progress.Status, "confidence": req.Progress.Confidence, "next_action": req.Progress.NextAction},
		},
		{
			Name:   "read_published_materials",
			Args:   map[string]any{"course_id": req.CourseID, "concept_id": req.ConceptID},
			Output: map[string]any{"source_ids": sourceIDs(grounded.Materials), "count": len(grounded.Materials)},
		},
	}

	response, err := a.Gateway.Respond(ctx, grounded)
	if err != nil {
		return repository.LearningResponse{}, AgentResult{}, fmt.Errorf("agent gateway failed: %w", err)
	}

	result := AgentResult{
		AgentName: "TutorAgent",
		Plan:      plan,
		Actions:   []string{"planned", "used_context_tools", "called_gateway"},
		Reasoning: plan.Reasoning,
		ToolsUsed: tools,
		Results: map[string]any{
			"mode":             req.Mode,
			"confidence_delta": response.ConfidenceDelta,
			"evidence_kind":    response.Evidence,
			"next_action":      response.NextAction,
		},
		NextAction: response.NextAction,
	}
	return response, result, nil
}

func publishedMaterials(materials []domain.LearningMaterial, allowExternal bool) []domain.LearningMaterial {
	result := make([]domain.LearningMaterial, 0, len(materials))
	for _, material := range materials {
		if !material.Published || (material.Source.IsExternal() && !allowExternal) {
			continue
		}
		result = append(result, material)
	}
	return result
}

func sourceIDs(materials []domain.LearningMaterial) []string {
	ids := make([]string, 0, len(materials))
	for _, material := range materials {
		ids = append(ids, material.Source.ID)
	}
	return ids
}
