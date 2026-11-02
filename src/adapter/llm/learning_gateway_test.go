package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

func TestOpenAICompatibleGatewayMapsJSONCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"kind\":\"question\",\"body\":\"What changes?\",\"sources\":[],\"evidence\":[],\"next_action\":\"Explain it\",\"confidence_delta\":2,\"escalate\":false}"}}]}`))
	}))
	defer server.Close()

	gateway := &OpenAICompatibleGateway{BaseURL: server.URL, APIKey: "test-key", Model: "test-model"}
	response, err := gateway.Respond(context.Background(), repository.LearningRequest{
		Mode:    domain.TutorModeGuided,
		Message: "Help",
	})
	if err != nil {
		t.Fatalf("Respond() error = %v", err)
	}
	if response.Kind != "question" || response.Body != "What changes?" || response.ConfidenceDelta != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestOpenAICompatibleGatewayAcceptsFencedJSONAndReportsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/error/") {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"message":"upstream unavailable"}}`))
			return
		}
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"kind\\\":\\\"hint\\\",\\\"body\\\":\\\"Try again\\\",\\\"next_action\\\":\\\"Try\\\",\\\"confidence_delta\\\":0}\\n```\"}}]}"))
	}))
	defer server.Close()

	gateway := &OpenAICompatibleGateway{BaseURL: server.URL, APIKey: "test-key", Model: "test-model"}
	response, err := gateway.Respond(context.Background(), repository.LearningRequest{})
	if err != nil || response.Kind != "hint" {
		t.Fatalf("fenced response = %+v, error = %v", response, err)
	}

	gateway.BaseURL = server.URL + "/error"
	_, err = gateway.Respond(context.Background(), repository.LearningRequest{})
	if err == nil || !strings.Contains(err.Error(), "status 502") || strings.Contains(err.Error(), "upstream unavailable") {
		t.Fatalf("error = %v", err)
	}
}
