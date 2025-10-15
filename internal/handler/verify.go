package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"api-recaptcha/internal/service"
)

type verifyRequest struct {
	Token  string `json:"token" binding:"required"`
	Action string `json:"action"`
}

// VerifyHandler processes the verification requests coming from the client.
type VerifyHandler struct {
	recaptcha *service.RecaptchaService
}

// NewVerifyHandler wires the dependencies into a VerifyHandler instance.
func NewVerifyHandler(recaptcha *service.RecaptchaService) VerifyHandler {
	return VerifyHandler{recaptcha: recaptcha}
}

// Handle receives a token and delegates the validation to the reCAPTCHA Enterprise API.
func (h VerifyHandler) Handle(c *gin.Context) {
	var payload verifyRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	assessment, err := h.recaptcha.Assess(c.Request.Context(), payload.Token, payload.Action)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "recaptcha verification failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assessment)
}
