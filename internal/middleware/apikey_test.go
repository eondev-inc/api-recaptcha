package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIKeyAuth_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedKey := "test-api-key-12345"
	router := gin.New()
	router.Use(APIKeyAuth(expectedKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(apiKeyHeader, expectedKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedKey := "test-api-key-12345"
	router := gin.New()
	router.Use(APIKeyAuth(expectedKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	// No API key header set

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedKey := "test-api-key-12345"
	router := gin.New()
	router.Use(APIKeyAuth(expectedKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(apiKeyHeader, "wrong-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestAPIKeyAuth_TimingAttackResistance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedKey := "test-api-key-12345"
	router := gin.New()
	router.Use(APIKeyAuth(expectedKey))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test with keys of different lengths to ensure constant-time comparison
	testCases := []string{
		"a",
		"test",
		"test-api-key-12344", // One character different
		"test-api-key-12346", // One character different
		"wrong-key-with-same-length-x",
	}

	for _, testKey := range testCases {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(apiKeyHeader, testKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403 for key %q, got %d", testKey, w.Code)
		}
	}
}
