package agentic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

type ExecutionLogger interface {
	AgentLogForSession(context.Context, string, string, string, map[string]any, map[string]any, []map[string]any) error
}

// Gateway runs the planning agent around the configured provider gateway and
// records its plan, tool selection, and result for each tutor turn.
type Gateway struct {
	Agent  *TutorAgent
	Logger ExecutionLogger
}

func (g Gateway) Respond(ctx context.Context, request repository.LearningRequest) (repository.LearningResponse, error) {
	if g.Agent == nil {
		return repository.LearningResponse{}, fmt.Errorf("tutor agent is not configured")
	}
	response, result, err := g.Agent.ExecuteTurn(ctx, request)
	if err != nil {
		return repository.LearningResponse{}, err
	}
	if g.Logger != nil {
		resultJSON, marshalErr := json.Marshal(result)
		if marshalErr == nil {
			var values map[string]any
			if json.Unmarshal(resultJSON, &values) == nil {
				tools := make([]map[string]any, 0, len(result.ToolsUsed))
				for _, tool := range result.ToolsUsed {
					tools = append(tools, map[string]any{"name": tool.Name, "args": tool.Args})
				}
				_ = g.Logger.AgentLogForSession(ctx, request.SessionID, result.AgentName, "execute_tutor_turn", map[string]any{"plan": result.Plan, "reasoning": result.Reasoning}, values, tools)
			}
		}
	}
	return response, nil
}
