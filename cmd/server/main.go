package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"api-recaptcha/internal/handler"
	"api-recaptcha/internal/middleware"
	"api-recaptcha/internal/service"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	appAPIKey := os.Getenv("APP_API_KEY")
	if appAPIKey == "" {
		log.Fatal("APP_API_KEY environment variable is required")
	}

	googleAPIKey := os.Getenv("GOOGLE_RECAPTCHA_API_KEY")
	if googleAPIKey == "" {
		log.Fatal("GOOGLE_RECAPTCHA_API_KEY environment variable is required")
	}

	siteKey := os.Getenv("GOOGLE_RECAPTCHA_SITE_KEY")
	if siteKey == "" {
		siteKey = "6LfTUuorAAAAAEYi8wmrchk8zaxcasstljmj-ZZT"
	}

	projectID := os.Getenv("GOOGLE_RECAPTCHA_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GOOGLE_RECAPTCHA_PROJECT_ID environment variable is required")
	}

	recaptchaEndpoint := "https://recaptchaenterprise.googleapis.com/v1/projects/" + projectID + "/assessments"

	recaptchaService := service.NewRecaptchaService(googleAPIKey, siteKey, recaptchaEndpoint)
	verifyHandler := handler.NewVerifyHandler(recaptchaService)

	router := gin.Default()
	router.Use(middleware.APIKeyAuth(appAPIKey))
	router.POST("/api/v1/recaptcha/verify", verifyHandler.Handle)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
