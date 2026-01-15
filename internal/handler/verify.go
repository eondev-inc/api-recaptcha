package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "api-recaptcha/internal/errors"
	"api-recaptcha/internal/logger"
	"api-recaptcha/internal/service"
)

type verifyRequest struct {
	Token  string `json:"token" binding:"required"`
	Action string `json:"action"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// VerifyHandler processes the verification requests coming from the client.
type VerifyHandler struct {
	recaptcha service.Assessor
}

// NewVerifyHandler wires the dependencies into a VerifyHandler instance.
func NewVerifyHandler(recaptcha service.Assessor) VerifyHandler {
	return VerifyHandler{recaptcha: recaptcha}
}

// Handle receives a token and delegates the validation to the reCAPTCHA Enterprise API.
func (h VerifyHandler) Handle(c *gin.Context) {
	var payload verifyRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		logger.Log.Warn("invalid request body",
			"error", err.Error(),
			"ip", c.ClientIP(),
		)
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "invalid request body",
			Code:  apperrors.ErrCodeInvalidRequest,
		})
		return
	}

	assessment, err := h.recaptcha.Assess(c.Request.Context(), payload.Token, payload.Action)
	if err != nil {
		// Check if it's an AppError
		if appErr, ok := err.(*apperrors.AppError); ok {
			logger.Log.Error("recaptcha verification failed",
				"error", appErr.Internal,
				"message", appErr.Message,
				"code", appErr.Code,
				"ip", c.ClientIP(),
			)
			c.JSON(appErr.HTTPStatus, errorResponse{
				Error: appErr.UserMessage(),
				Code:  appErr.Code,
			})
			return
		}

		// Fallback for unexpected errors
		logger.Log.Error("unexpected error during recaptcha verification",
			"error", err.Error(),
			"ip", c.ClientIP(),
		)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
			Code:  apperrors.ErrCodeInternalError,
		})
		return
	}

	logger.Log.Info("recaptcha verification successful",
		"action", payload.Action,
		"valid", assessment.Valid,
		"score", assessment.Score,
		"ip", c.ClientIP(),
	)

	c.JSON(http.StatusOK, assessment)
}
