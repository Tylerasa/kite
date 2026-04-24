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
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Pin      string `json:"pin" binding:"required,min=4,max=6"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

// Signup godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      signupRequest  true  "Credentials"
// @Success      201   {object}  tokenResponse
// @Failure      400   {object}  apiutil.ErrorResponse
// @Failure      409   {object}  apiutil.ErrorResponse  "email already registered"
// @Router       /auth/signup [post]
func (ctrl *AuthController) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: apiutil.BindingError(err),
		}})
		return
	}

	result, err := ctrl.uc.Signup(c.Request.Context(), in.SignupCommand{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Pin:      req.Pin,
	})
	if err != nil {
		apiutil.RespondError(c, ctrl.logger, err)
		return
	}

	c.JSON(http.StatusCreated, tokenResponse{Token: result.Token, UserID: result.UserID.String(), Name: result.Name})
}

// Login godoc
// @Summary      Authenticate and receive a JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Credentials"
// @Success      200   {object}  tokenResponse
// @Failure      400   {object}  apiutil.ErrorResponse
// @Failure      401   {object}  apiutil.ErrorResponse  "invalid credentials"
// @Router       /auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
			Code: "validation_error", Message: apiutil.BindingError(err),
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

	c.JSON(http.StatusOK, tokenResponse{Token: result.Token, UserID: result.UserID.String(), Name: result.Name})
}
