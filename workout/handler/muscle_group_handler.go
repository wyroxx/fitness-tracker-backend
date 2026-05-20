package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

// MuscleGroupHandler handles CRUD operations for muscle groups.
type MuscleGroupHandler struct {
	repo repository.MuscleGroupRepository
}

func NewMuscleGroupHandler(repo repository.MuscleGroupRepository) *MuscleGroupHandler {
	return &MuscleGroupHandler{repo: repo}
}

func (h *MuscleGroupHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	mg := r.Group("/muscle-groups")
	mg.Use(auth)
	{
		mg.POST("", h.create)
		mg.GET("", h.list)
		mg.GET("/:id", h.getByID)
		mg.PUT("/:id", h.update)
		mg.DELETE("/:id", h.delete)
	}
}

type muscleGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

// create muscle group
// @Summary      Create muscle group
// @Tags         muscle-groups
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      muscleGroupRequest  true  "Muscle group"
// @Success      201      {object}  models.MuscleGroup
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /muscle-groups [post]
func (h *MuscleGroupHandler) create(c *gin.Context) {
	var req muscleGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mg := &models.MuscleGroup{Name: req.Name}
	if err := h.repo.Create(c.Request.Context(), mg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mg)
}

// list muscle groups
// @Summary      List muscle groups
// @Tags         muscle-groups
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   models.MuscleGroup
// @Router       /muscle-groups [get]
func (h *MuscleGroupHandler) list(c *gin.Context) {
	groups, err := h.repo.List(c.Request.Context(), 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

// get muscle group
// @Summary      Get muscle group by ID
// @Tags         muscle-groups
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      int  true  "MuscleGroup ID"
// @Success      200  {object}  models.MuscleGroup
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /muscle-groups/{id} [get]
func (h *MuscleGroupHandler) getByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	mg, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mg)
}

// update muscle group
// @Summary      Update muscle group
// @Tags         muscle-groups
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      int                true  "MuscleGroup ID"
// @Param        payload  body      muscleGroupRequest true  "Update"
// @Success      200      {object}  models.MuscleGroup
// @Failure      400      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Router       /muscle-groups/{id} [put]
func (h *MuscleGroupHandler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	mg, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var req muscleGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mg.Name = req.Name
	if err := h.repo.Update(c.Request.Context(), mg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mg)
}

// delete muscle group
// @Summary      Delete muscle group
// @Tags         muscle-groups
// @Security     BearerAuth
// @Param        id   path      int  true  "MuscleGroup ID"
// @Success      204  {string}  string  "No Content"
// @Failure      400  {object}  errorResponse
// @Router       /muscle-groups/{id} [delete]
func (h *MuscleGroupHandler) delete(c *gin.Context) {
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
