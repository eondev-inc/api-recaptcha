package service

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
)

const (
	maxErrorBodyBytes = 1024
)

// AssessmentResult exposes the relevant information returned by the reCAPTCHA Enterprise API.
type AssessmentResult struct {
	Valid         bool      `json:"valid"`
	Score         float64   `json:"score,omitempty"`
	Action        string    `json:"action,omitempty"`
	InvalidReason string    `json:"invalidReason,omitempty"`
	Reasons       []string  `json:"reasons,omitempty"`
	CreateTime    time.Time `json:"createTime,omitempty"`
}

type assessmentRequest struct {
	Event assessmentEvent `json:"event"`
}

type assessmentEvent struct {
	Token          string `json:"token"`
	SiteKey        string `json:"siteKey"`
	ExpectedAction string `json:"expectedAction,omitempty"`
}

type assessmentResponse struct {
	TokenProperties struct {
		Valid         bool      `json:"valid"`
		Action        string    `json:"action"`
		InvalidReason string    `json:"invalidReason"`
		CreateTime    time.Time `json:"createTime"`
	} `json:"tokenProperties"`
	RiskAnalysis struct {
		Score   float64  `json:"score"`
		Reasons []string `json:"reasons"`
	} `json:"riskAnalysis"`
}

// RecaptchaService coordinates the interaction with the reCAPTCHA Enterprise API.
type RecaptchaService struct {
	client   *http.Client
	apiKey   string
	siteKey  string
	endpoint string
}

// NewRecaptchaService builds a RecaptchaService with sane defaults.
func NewRecaptchaService(apiKey, siteKey, endpoint string) *RecaptchaService {
	return &RecaptchaService{
		client:   &http.Client{Timeout: 10 * time.Second},
		apiKey:   apiKey,
		siteKey:  siteKey,
		endpoint: endpoint,
	}
}

// Assess validates the provided token and returns the assessment outcome.
func (s *RecaptchaService) Assess(ctx context.Context, token, action string) (AssessmentResult, error) {
	if strings.TrimSpace(token) == "" {
		return AssessmentResult{}, errors.New("token is required")
	}

	payload := assessmentRequest{
		Event: assessmentEvent{
			Token:   token,
			SiteKey: s.siteKey,
		},
	}

	if trimmedAction := strings.TrimSpace(action); trimmedAction != "" {
		payload.Event.ExpectedAction = trimmedAction
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AssessmentResult{}, fmt.Errorf("failed to marshal assessment request: %w", err)
	}

	endpoint := fmt.Sprintf("%s?key=%s", s.endpoint, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return AssessmentResult{}, fmt.Errorf("failed to create assessment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return AssessmentResult{}, fmt.Errorf("request to reCAPTCHA Enterprise failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AssessmentResult{}, fmt.Errorf("failed to read assessment response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		trimmed := string(respBody)
		if len(trimmed) > maxErrorBodyBytes {
			trimmed = trimmed[:maxErrorBodyBytes]
		}
		return AssessmentResult{}, fmt.Errorf("reCAPTCHA Enterprise returned status %d: %s", resp.StatusCode, trimmed)
	}

	var assessment assessmentResponse
	if err := json.Unmarshal(respBody, &assessment); err != nil {
		return AssessmentResult{}, fmt.Errorf("failed to decode assessment response: %w", err)
	}

	result := AssessmentResult{
		Valid:         assessment.TokenProperties.Valid,
		Action:        assessment.TokenProperties.Action,
		InvalidReason: assessment.TokenProperties.InvalidReason,
		Score:         assessment.RiskAnalysis.Score,
		Reasons:       assessment.RiskAnalysis.Reasons,
		CreateTime:    assessment.TokenProperties.CreateTime,
	}

	return result, nil
}
