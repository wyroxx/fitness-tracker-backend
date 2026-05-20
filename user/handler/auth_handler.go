package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/user/use_case"
)

// AuthHandler exposes authentication-related HTTP endpoints (login, refresh, logout).
type AuthHandler struct {
	svc *use_case.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *use_case.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterRoutes wires the auth endpoints. The login and refresh endpoints are public.
// The logout endpoint requires the provided auth middleware to ensure the caller is authenticated.
func (h *AuthHandler) RegisterRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc) {
	authGroup := r.Group("/auth")

	// Public endpoints
	authGroup.POST("/login", h.login)
	authGroup.POST("/refresh", h.refresh)

	// Protected endpoint
	authGroupWithAuth := r.Group("/auth")
	authGroupWithAuth.Use(authMiddleware)
	authGroupWithAuth.POST("/logout", h.logout)
}

// --- request/response DTOs ---

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Login
// @Summary      User login
// @Description  Authenticates user credentials and returns JWT pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      loginRequest   true  "Credentials"
// @Success      200      {object}  tokenResponse
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	access, refresh, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokenResponse{AccessToken: access, RefreshToken: refresh})
}

// Refresh tokens
// @Summary      Refresh JWT tokens
// @Description  Exchanges a refresh token for a new JWT pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      refreshRequest  true  "Refresh token"
// @Success      200      {object}  tokenResponse
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	access, refresh, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokenResponse{AccessToken: access, RefreshToken: refresh})
}

// Logout
// @Summary      Logout (client-side token discard)
// @Tags         auth
// @Security     BearerAuth
// @Success      204  {string}  string  "No Content"
// @Router       /auth/logout [post]
func (h *AuthHandler) logout(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
