package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/VibeTeam/fitness-tracker-backend/shared/middleware"
	"github.com/VibeTeam/fitness-tracker-backend/user/models"
	"github.com/VibeTeam/fitness-tracker-backend/user/repository"
)

// UserHandler bundles dependencies for user-related HTTP endpoints.
type UserHandler struct {
	repo repository.UserRepository
}

// New creates a new UserHandler instance.
func New(repo repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// RegisterRoutes attaches user CRUD endpoints to the supplied Gin router.
func (h *UserHandler) RegisterRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc) {
	users := r.Group("/users")
	usersWithAuth := r.Group("/users")

	{
		users.POST("", h.create)
	}

	usersWithAuth.Use(authMiddleware)
	{
		usersWithAuth.GET("/me", h.getMe)
	}
}

type createUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Create user
// @Summary      Register new user
// @Description  Creates a user and returns the stored record
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        payload  body      createUserRequest  true  "User info"
// @Success      201      {object}  models.User
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /users [post]
func (h *UserHandler) create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user := &models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := h.repo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, userResponse(user))
}

// Get current authenticated user
// @Summary      Get current user
// @Description  Returns the authenticated user's information
// @Tags         users
// @Produce      json
// @Success      200  {object}  models.User
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/me [get]
// @Security     BearerAuth
func (h *UserHandler) getMe(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userResponse(user))
}

// helper to shape user JSON response without password hash
func userResponse(u *models.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"created_at": u.CreatedAt,
	}
}
