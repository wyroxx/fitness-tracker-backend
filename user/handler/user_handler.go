package handler

import (
	"net/http"
	"strconv"

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
		users.GET("", h.list)
		users.GET("/:id", h.getByID)
		users.POST("", h.create)
		users.PUT("/:id", h.update)
		users.DELETE("/:id", h.delete)
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

type updateUserRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
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

// List users
// @Summary      List users
// @Tags         users
// @Produce      json
// @Param        limit   query     int  false  "Limit"
// @Param        offset  query     int  false  "Offset"
// @Success      200     {array}   models.User
// @Failure      500     {object}  errorResponse
// @Router       /users [get]
// @Security     BearerAuth
func (h *UserHandler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, err := h.repo.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// map to response slice
	resp := make([]gin.H, len(users))
	for i, u := range users {
		resp[i] = userResponse(u)
	}
	c.JSON(http.StatusOK, resp)
}

// Get user by ID
// @Summary      Get user by ID
// @Tags         users
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  models.User
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [get]
// @Security     BearerAuth
func (h *UserHandler) getByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	user, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, userResponse(user))
}

// Update user
// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id       path      int                 true  "User ID"
// @Param        payload  body      updateUserRequest   true  "Update information"
// @Success      200      {object}  models.User
// @Failure      400      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /users/{id} [put]
// @Security     BearerAuth
func (h *UserHandler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		user.PasswordHash = string(hash)
	}

	if err := h.repo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userResponse(user))
}

// Delete user
// @Summary      Delete user
// @Tags         users
// @Param        id   path      int  true  "User ID"
// @Success      204  {string}  string  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /users/{id} [delete]
// @Security     BearerAuth
func (h *UserHandler) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
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
