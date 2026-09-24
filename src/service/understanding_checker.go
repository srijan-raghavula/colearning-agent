package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

type UnderstandingCheckService struct {
	gateway repository.FormativeCheckGateway
}

func NewUnderstandingCheckService(gateway repository.FormativeCheckGateway) *UnderstandingCheckService {
	return &UnderstandingCheckService{gateway: gateway}
}

func (s *UnderstandingCheckService) Check(ctx context.Context, req repository.FormativeCheckRequest) (repository.FormativeCheckResponse, error) {
	if s == nil || s.gateway == nil {
		return repository.FormativeCheckResponse{}, errors.New("understanding check gateway is not configured")
	}
	if strings.TrimSpace(req.Answer) == "" {
		return repository.FormativeCheckResponse{}, errors.New("an answer or explanation is required")
	}
	response, err := s.gateway.Check(ctx, req)
	if err != nil {
		return repository.FormativeCheckResponse{}, fmt.Errorf("understanding check failed: %w", err)
	}
	response.Summary = strings.TrimSpace(response.Summary)
	response.NextAction = strings.TrimSpace(response.NextAction)
	if response.Summary == "" {
		return repository.FormativeCheckResponse{}, errors.New("understanding check summary is empty")
	}
	if response.NextAction == "" {
		response.NextAction = "Continue practicing this concept"
	}
	if response.ConfidenceDelta < -10 {
		response.ConfidenceDelta = -10
	}
	if response.ConfidenceDelta > 10 {
		response.ConfidenceDelta = 10
	}
	return response, nil
}
