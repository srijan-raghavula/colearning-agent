package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/agentic"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// OpenAICompatibleGateway talks to providers that implement the
// /chat/completions contract, including OpenAI-compatible self-hosted APIs.
// The provider returns only the typed learning contract; guardrails remain in
// service.TutorAgent.
type OpenAICompatibleGateway struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Format   responseFormat `json:"response_format"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *OpenAICompatibleGateway) Respond(ctx context.Context, req repository.LearningRequest) (repository.LearningResponse, error) {
	var result repository.LearningResponse
	if err := g.completeJSON(ctx, systemPrompt(), marshalLearningRequest(req), &result); err != nil {
		return repository.LearningResponse{}, err
	}
	return result, nil
}

func (g *OpenAICompatibleGateway) Check(ctx context.Context, req repository.FormativeCheckRequest) (repository.FormativeCheckResponse, error) {
	var result repository.FormativeCheckResponse
	payload, _ := json.Marshal(req)
	prompt := `You are a formative learning checker. Return JSON with summary, feedback (array of concise strings), evidence (array of concise strings), next_action, and confidence_delta from -10 to 10. This is not a grade.`
	if err := g.completeJSON(ctx, prompt, string(payload), &result); err != nil {
		return repository.FormativeCheckResponse{}, err
	}
	return result, nil
}

func (g *OpenAICompatibleGateway) GenerateCurriculum(ctx context.Context, course domain.LearningCourse) (agentic.CurriculumDraft, error) {
	var result agentic.CurriculumDraft
	payload, _ := json.Marshal(course)
	prompt := `You are a curriculum authoring assistant. Return a bounded JSON draft with provider, summary, and modules. Each module has title, summary, chapters; each chapter has title, summary, lessons; each lesson has title, summary, contents; each content has kind, title, and body. Produce 1-4 modules, 1-4 chapters per module, 1-4 lessons per chapter, and 1-4 contents per lesson. Keep every text field concise and educational.`
	if err := g.completeJSON(ctx, prompt, string(payload), &result); err != nil {
		return agentic.CurriculumDraft{}, err
	}
	return result, nil
}

func (g *OpenAICompatibleGateway) completeJSON(ctx context.Context, system, user string, target any) error {
	if g == nil || strings.TrimSpace(g.BaseURL) == "" {
		return errors.New("llm base URL is not configured")
	}
	if strings.TrimSpace(g.APIKey) == "" {
		return errors.New("llm API key is not configured")
	}
	if strings.TrimSpace(g.Model) == "" {
		return errors.New("llm model is not configured")
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	body, err := json.Marshal(chatRequest{
		Model:    g.Model,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Format:   responseFormat{Type: "json_object"},
	})
	if err != nil {
		return fmt.Errorf("encode llm request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create llm request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+g.APIKey)
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("call llm provider: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read llm response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("llm provider returned status %d: %s", response.StatusCode, providerError(responseBody))
	}
	var completion chatResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return fmt.Errorf("decode llm response: %w", err)
	}
	if completion.Error != nil {
		return fmt.Errorf("llm provider error: %s", completion.Error.Message)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return errors.New("llm provider returned no completion")
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(strings.TrimSpace(content), "```")
	if err := json.Unmarshal([]byte(content), target); err != nil {
		return fmt.Errorf("decode llm contract: %w", err)
	}
	return nil
}

func systemPrompt() string {
	return `You are a grounded learning tutor. Use only the course context supplied by the application.
Return a single JSON object with exactly these useful fields:
kind (one of explanation, question, hint, practice, summary, escalation),
body (constructive student-facing response),
sources (array of source objects copied from supplied material IDs),
evidence (array of objects with kind, note, confidence_delta),
next_action (short actionable instruction),
confidence_delta (integer from -10 to 10),
escalate (boolean).
Do not invent source IDs. Do not include markdown fences.`
}

func marshalLearningRequest(req repository.LearningRequest) string {
	payload, _ := json.Marshal(struct {
		CourseTitle string                    `json:"course_title"`
		Concept     domain.LearningConcept    `json:"concept"`
		Objective   domain.LearningObjective  `json:"objective"`
		Materials   []domain.LearningMaterial `json:"materials"`
		RecentTurns []domain.LearningTurn     `json:"recent_turns"`
		Progress    domain.ConceptProgress    `json:"progress"`
		Policy      domain.TutorPolicy        `json:"policy"`
		Mode        domain.TutorMode          `json:"mode"`
		Message     string                    `json:"message"`
	}{
		req.CourseTitle, req.Concept, req.Objective, req.Materials, req.RecentTurns,
		req.Progress, req.Policy, req.Mode, req.Message,
	})
	return string(payload)
}

func providerError(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Message != "" {
		return envelope.Error.Message
	}
	message := strings.TrimSpace(string(body))
	if len(message) > 400 {
		return message[:400]
	}
	return message
}
