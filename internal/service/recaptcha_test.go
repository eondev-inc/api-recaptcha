package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "api-recaptcha/internal/errors"
)

// assessRequestBody mirrors the payload the service sends upstream, so a test can assert what
// actually left the process.
type assessRequestBody struct {
	Event struct {
		Token          string `json:"token"`
		SiteKey        string `json:"siteKey"`
		ExpectedAction string `json:"expectedAction"`
	} `json:"event"`
}

func successAssessmentBody() string {
	return `{
		"tokenProperties": {
			"valid": true,
			"action": "login",
			"createTime": "2026-01-01T00:00:00Z"
		},
		"riskAnalysis": {
			"score": 0.9,
			"reasons": ["AUTOMATION"]
		}
	}`
}

// Go's http.Client follows redirects and copies custom headers to the new host; only
// Authorization, Cookie and Proxy-Authorization are stripped. X-goog-api-key is none of those, so
// allowing a redirect hands the Google API key to whatever host the response points at.
func TestAssess_DoesNotForwardAPIKeyOnRedirect(t *testing.T) {
	const apiKey = "google-api-key-must-not-leak"

	var leakTargetKey string
	var leakTargetCalls int
	leakTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leakTargetCalls++
		leakTargetKey = r.Header.Get("X-goog-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, successAssessmentBody())
	}))
	defer leakTarget.Close()

	var directKey string
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		directKey = r.Header.Get("X-goog-api-key")
		http.Redirect(w, r, leakTarget.URL, http.StatusFound)
	}))
	defer redirector.Close()

	svc := NewRecaptchaService(apiKey, "site-key", redirector.URL)

	_, err := svc.Assess(context.Background(), "visitor-token", "login")

	if directKey != apiKey {
		t.Errorf("the API key belongs in the request to Google, got %q", directKey)
	}
	if leakTargetCalls != 0 {
		t.Errorf("a redirect target must never be contacted, it received %d request(s)", leakTargetCalls)
	}
	if leakTargetKey != "" {
		t.Errorf("the Google API key leaked to the redirect target: %q", leakTargetKey)
	}
	if err == nil {
		t.Fatal("a 302 is not a successful assessment and must surface as an error")
	}
}

// The happy path had no coverage at all: nothing pinned the field mapping from the Google
// response into AssessmentResult.
func TestAssess_MapsAssessmentResponse(t *testing.T) {
	var received assessRequestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Errorf("unmarshal request body %q: %v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, successAssessmentBody())
	}))
	defer server.Close()

	svc := NewRecaptchaService("api-key", "site-key", server.URL)

	result, err := svc.Assess(context.Background(), "visitor-token", "  login  ")
	if err != nil {
		t.Fatalf("expected a successful assessment, got %v", err)
	}

	if !result.Valid {
		t.Error("expected the assessment to be valid")
	}
	if result.Score != 0.9 {
		t.Errorf("expected score 0.9, got %v", result.Score)
	}
	if len(result.Reasons) != 1 || result.Reasons[0] != "AUTOMATION" {
		t.Errorf("expected the risk reasons to be mapped, got %v", result.Reasons)
	}
	if received.Event.Token != "visitor-token" || received.Event.SiteKey != "site-key" {
		t.Errorf("unexpected upstream event: %+v", received.Event)
	}
	if received.Event.ExpectedAction != "login" {
		t.Errorf("the action must be trimmed before it is sent, got %q", received.Event.ExpectedAction)
	}
}

// The upstream error body is Google's, but it can echo request detail; only the user-safe message
// may reach the caller.
func TestAssess_DoesNotLeakUpstreamErrorToTheCaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"upstream-internal-detail"}}`)
	}))
	defer server.Close()

	svc := NewRecaptchaService("api-key", "site-key", server.URL)

	_, err := svc.Assess(context.Background(), "visitor-token", "login")
	if err == nil {
		t.Fatal("expected an error for a 400 from Google")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected an *AppError, got %T", err)
	}
	if appErr.Code != apperrors.ErrCodeRecaptchaFailed {
		t.Errorf("expected code %s, got %s", apperrors.ErrCodeRecaptchaFailed, appErr.Code)
	}
	if strings.Contains(appErr.UserMessage(), "upstream-internal-detail") {
		t.Errorf("the upstream body leaked into the user message: %q", appErr.UserMessage())
	}
}

// Validation must happen before any network call, so a malformed request never reaches Google.
func TestAssess_ValidatesInputBeforeCallingGoogle(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, successAssessmentBody())
	}))
	defer server.Close()

	svc := NewRecaptchaService("api-key", "site-key", server.URL)

	cases := []struct {
		name  string
		token string
		want  string
	}{
		{name: "empty token", token: "   ", want: "token is required"},
		{name: "token too long", token: strings.Repeat("t", 2001), want: "token too long"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false

			_, err := svc.Assess(context.Background(), tc.token, "login")
			if err == nil {
				t.Fatalf("expected %q to be rejected", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error %q, got %q", tc.want, err.Error())
			}
			if called {
				t.Error("an invalid request must not reach Google")
			}
		})
	}

	called = false
	if _, err := svc.Assess(context.Background(), "visitor-token", strings.Repeat("a", 101)); err == nil {
		t.Error("expected an over-long action to be rejected")
	}
	if called {
		t.Error("an invalid action must not reach Google")
	}
}
