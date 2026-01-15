package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apperrors "api-recaptcha/internal/errors"
	"api-recaptcha/internal/logger"
)

const (
	maxErrorBodyBytes = 1024
)

// Assessor defines the interface for reCAPTCHA assessment.
type Assessor interface {
	Assess(ctx context.Context, token, action string) (AssessmentResult, error)
}

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
		return AssessmentResult{}, apperrors.NewValidationError("token is required", nil)
	}

	// Validate token format (basic check)
	if len(token) > 2000 {
		return AssessmentResult{}, apperrors.NewValidationError("token too long", nil)
	}

	// Validate action length
	if len(action) > 100 {
		return AssessmentResult{}, apperrors.NewValidationError("action name too long", nil)
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
		logger.Log.Error("failed to marshal assessment request", "error", err)
		return AssessmentResult{}, apperrors.NewInternalError("failed to prepare request", err)
	}

	// Use API key in header instead of query param for better security
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		logger.Log.Error("failed to create assessment request", "error", err)
		return AssessmentResult{}, apperrors.NewInternalError("failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Log.Error("request to reCAPTCHA Enterprise failed", "error", err)
		return AssessmentResult{}, apperrors.NewRecaptchaError("failed to connect to reCAPTCHA service", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("failed to read assessment response", "error", err)
		return AssessmentResult{}, apperrors.NewRecaptchaError("failed to read reCAPTCHA response", err)
	}

	if resp.StatusCode != http.StatusOK {
		trimmed := string(respBody)
		if len(trimmed) > maxErrorBodyBytes {
			trimmed = trimmed[:maxErrorBodyBytes]
		}
		logger.Log.Error("reCAPTCHA Enterprise returned error",
			"status", resp.StatusCode,
			"body", trimmed,
		)
		return AssessmentResult{}, apperrors.NewRecaptchaError(
			"reCAPTCHA verification failed",
			fmt.Errorf("status %d: %s", resp.StatusCode, trimmed),
		)
	}

	var assessment assessmentResponse
	if err := json.Unmarshal(respBody, &assessment); err != nil {
		logger.Log.Error("failed to decode assessment response", "error", err, "body", string(respBody))
		return AssessmentResult{}, apperrors.NewInternalError("failed to parse reCAPTCHA response", err)
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
