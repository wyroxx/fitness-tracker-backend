package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
		mg.GET("", h.list)
	}
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
