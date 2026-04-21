package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/infrastructure/rest/apiutil"
)

type AuthController struct {
	uc     in.AuthUseCase
	logger *slog.Logger
}

func NewAuthController(uc in.AuthUseCase, logger *slog.Logger) *AuthController {
	return &AuthController{uc: uc, logger: logger}
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

func (ctrl *AuthController) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: err.Error(),
		}})
		return
	}

	result, err := ctrl.uc.Signup(c.Request.Context(), in.SignupCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusCreated, tokenResponse{Token: result.Token, UserID: result.UserID.String()})
}

func (ctrl *AuthController) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: err.Error(),
		}})
		return
	}

	result, err := ctrl.uc.Login(c.Request.Context(), in.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse{Token: result.Token, UserID: result.UserID.String()})
}
