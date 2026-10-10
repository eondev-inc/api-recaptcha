package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "api-recaptcha/internal/errors"
	"api-recaptcha/internal/service"
)

// mockAssessor implements the service.Assessor interface for testing
type mockAssessor struct {
	assessFunc func(ctx context.Context, token, action string) (service.AssessmentResult, error)
}

func (m *mockAssessor) Assess(ctx context.Context, token, action string) (service.AssessmentResult, error) {
	if m.assessFunc != nil {
		return m.assessFunc(ctx, token, action)
	}
	return service.AssessmentResult{}, nil
}

func TestVerifyHandler_Handle_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAssessor{
		assessFunc: func(ctx context.Context, token, action string) (service.AssessmentResult, error) {
			return service.AssessmentResult{
				Valid:  true,
				Score:  0.9,
				Action: action,
			}, nil
		},
	}

	handler := NewVerifyHandler(mock)

	router := gin.New()
	router.POST("/verify", handler.Handle)

	reqBody := verifyRequest{
		Token:  "valid-token",
		Action: "login",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result service.AssessmentResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !result.Valid {
		t.Error("expected valid assessment")
	}
	if result.Score != 0.9 {
		t.Errorf("expected score 0.9, got %f", result.Score)
	}
}

func TestVerifyHandler_Handle_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAssessor{}
	handler := NewVerifyHandler(mock)

	router := gin.New()
	router.POST("/verify", handler.Handle)

	// Missing required token field
	reqBody := map[string]string{"action": "login"}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestVerifyHandler_Handle_RecaptchaError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAssessor{
		assessFunc: func(ctx context.Context, token, action string) (service.AssessmentResult, error) {
			return service.AssessmentResult{}, apperrors.NewRecaptchaError("recaptcha failed", nil)
		},
	}

	handler := NewVerifyHandler(mock)

	router := gin.New()
	router.POST("/verify", handler.Handle)

	reqBody := verifyRequest{
		Token:  "invalid-token",
		Action: "login",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", w.Code)
	}

	var errResp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if errResp.Code != apperrors.ErrCodeRecaptchaFailed {
		t.Errorf("expected error code %s, got %s", apperrors.ErrCodeRecaptchaFailed, errResp.Code)
	}
}

// The service validates the token length only after the whole body has been buffered, so nothing
// bounds the body unless the handler does it. An oversized body must be rejected with 413, before
// the service is ever consulted.
func TestVerifyHandler_Handle_RejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	mock := &mockAssessor{
		assessFunc: func(ctx context.Context, token, action string) (service.AssessmentResult, error) {
			called = true
			return service.AssessmentResult{Valid: true}, nil
		},
	}

	handler := NewVerifyHandler(mock)

	router := gin.New()
	router.POST("/verify", handler.Handle)

	oversized, err := json.Marshal(verifyRequest{Token: strings.Repeat("a", 64<<10)})
	if err != nil {
		t.Fatalf("failed to marshal oversized body: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413 for an oversized body, got %d", w.Code)
	}

	var errResp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Code != "PAYLOAD_TOO_LARGE" {
		t.Errorf("expected error code PAYLOAD_TOO_LARGE, got %s", errResp.Code)
	}
	if called {
		t.Error("an oversized body must be rejected before the assessor is called")
	}
}

// A token at the documented maximum must keep working, so the new limit cannot be tight enough
// to reject a legitimate request.
func TestVerifyHandler_Handle_AcceptsBodyWithinTheLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAssessor{
		assessFunc: func(ctx context.Context, token, action string) (service.AssessmentResult, error) {
			return service.AssessmentResult{Valid: true}, nil
		},
	}

	handler := NewVerifyHandler(mock)

	router := gin.New()
	router.POST("/verify", handler.Handle)

	reqBody := verifyRequest{Token: strings.Repeat("t", 2000), Action: strings.Repeat("a", 100)}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("a request at the documented limits must be accepted, got %d", w.Code)
	}
}
