package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCORSRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.POST("/verify", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func doCORSPreflight(t *testing.T, r http.Handler, origin string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodOptions, "/verify", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	return rec
}

func doCORSRequest(t *testing.T, r http.Handler, origin string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/verify", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	return rec
}

// Unset CORS_ALLOWED_ORIGINS must deny every cross-origin request: the safe default for a
// service that authenticates with a header is "no browser origin is allowed".
func TestCORS_DeniesCrossOriginByDefault(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	rec := doCORSRequest(t, newCORSRouter(), "https://evil.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("unset CORS_ALLOWED_ORIGINS must not allow any origin, got %q", got)
	}
}

// A wildcard entry is not an allowlist and must never be honoured: this API is called by
// servers holding a shared key, so every cross-origin browser origin is an explicit decision.
func TestCORS_WildcardEntryIsIgnored(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	rec := doCORSRequest(t, newCORSRouter(), "https://evil.example.com")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a wildcard entry must not allow any origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("credentials must never be advertised, got %q", got)
	}
}

// Only an exact allowlist match may be echoed back, and the response must declare Origin as a
// cache-key input.
func TestCORS_AllowsListedOriginOnly(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://ops.example.com")

	rec := doCORSRequest(t, newCORSRouter(), "https://app.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected the listed origin to be echoed, got %q", got)
	}
	if got := strings.Join(rec.Header().Values("Vary"), ","); !strings.Contains(got, "Origin") {
		t.Errorf("a reflected origin must add Vary: Origin, got Vary %q", got)
	}

	rec = doCORSRequest(t, newCORSRouter(), "https://evil.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("unlisted origin must not be allowed, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("credentials must never be advertised, got %q", got)
	}
}

// The API authenticates with X-API-Key and uses no cookies, so Allow-Credentials is never
// needed and must never be sent.
func TestCORS_NeverAdvertisesCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	rec := doCORSRequest(t, newCORSRouter(), "https://app.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("advertised credentials %q", got)
	}
}

// Preflight must advertise only the headers and methods this API actually exposes.
func TestCORS_PreflightAdvertisesOnlyRealSurface(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	rec := doCORSPreflight(t, newCORSRouter(), "https://app.example.com")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("expected methods %q, got %q", "GET, POST, OPTIONS", got)
	}
	headers := rec.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(headers, "X-API-Key") || !strings.Contains(headers, "Content-Type") {
		t.Errorf("expected X-API-Key and Content-Type to stay allowed, got %q", headers)
	}
	if strings.Contains(headers, "PUT") || strings.Contains(headers, "X-CSRF-Token") {
		t.Errorf("preflight must not advertise unused surface, got %q", headers)
	}
}

// A denied origin must not be told which methods and headers the API accepts.
func TestCORS_DeniedOriginGetsNoSurface(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	rec := doCORSPreflight(t, newCORSRouter(), "https://evil.example.com")
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("denied origin must not receive Allow-Methods, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "" {
		t.Errorf("denied origin must not receive Allow-Headers, got %q", got)
	}
}
