package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"api-recaptcha/internal/handler"
	"api-recaptcha/internal/logger"
	"api-recaptcha/internal/middleware"
	"api-recaptcha/internal/service"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Log.Warn(".env file not found, using system environment variables")
	}

	appAPIKey := os.Getenv("APP_API_KEY")
	if appAPIKey == "" {
		logger.Log.Error("APP_API_KEY environment variable is required")
		os.Exit(1)
	}

	googleAPIKey := os.Getenv("GOOGLE_RECAPTCHA_API_KEY")
	if googleAPIKey == "" {
		logger.Log.Error("GOOGLE_RECAPTCHA_API_KEY environment variable is required")
		os.Exit(1)
	}

	siteKey := os.Getenv("GOOGLE_RECAPTCHA_SITE_KEY")
	if siteKey == "" {
		logger.Log.Warn("GOOGLE_RECAPTCHA_SITE_KEY not set, using default (not recommended for production)")
		siteKey = "6LfTUuorAAAAAEYi8wmrchk8zaxcasstljmj-ZZT"
	}

	projectID := os.Getenv("GOOGLE_RECAPTCHA_PROJECT_ID")
	if projectID == "" {
		logger.Log.Error("GOOGLE_RECAPTCHA_PROJECT_ID environment variable is required")
		os.Exit(1)
	}

	recaptchaEndpoint := "https://recaptchaenterprise.googleapis.com/v1/projects/" + projectID + "/assessments"

	recaptchaService := service.NewRecaptchaService(googleAPIKey, siteKey, recaptchaEndpoint)
	verifyHandler := handler.NewVerifyHandler(recaptchaService)

	rateLimiter := middleware.NewRateLimiter()
	defer rateLimiter.Stop()

	router := gin.Default()
	router.Use(middleware.CORS())

	// Health check endpoints (no authentication required)
	router.GET("/health", handler.HealthCheck)
	router.GET("/ready", handler.ReadinessCheck)

	// API endpoints (with rate limiting and authentication)
	api := router.Group("/api/v1")
	api.Use(rateLimiter.RateLimit())
	api.Use(middleware.APIKeyAuth(appAPIKey))
	api.POST("/recaptcha/verify", verifyHandler.Handle)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Log.Info("starting server",
			"port", port,
			"environment", os.Getenv("GIN_MODE"),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutting down server...")

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Log.Info("server stopped gracefully")
}
