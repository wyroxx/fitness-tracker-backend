package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type WorkoutTypeHandler struct {
	repo repository.WorkoutTypeRepository
}

func NewWorkoutTypeHandler(repo repository.WorkoutTypeRepository) *WorkoutTypeHandler {
	return &WorkoutTypeHandler{repo: repo}
}

func (h *WorkoutTypeHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	r.GET("/muscle-groups/:id/workout-types", auth, h.listByMuscleGroup)
}

// List workout types by muscle group
// @Summary      List workout types by muscle group
// @Tags         workout-types
// @Security     BearerAuth
// @Produce      json
// @Param        id      path      int  true   "MuscleGroup ID"
// @Param        limit   query     int  false  "Limit"
// @Param        offset  query     int  false  "Offset"
// @Success      200     {array}  models.WorkoutType
// @Failure      400     {object}  errorResponse
// @Failure      500     {object}  errorResponse
// @Router       /muscle-groups/{id}/workout-types [get]
func (h *WorkoutTypeHandler) listByMuscleGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	limit := queryInt(c, "limit", 100)
	offset := queryInt(c, "offset", 0)
	types, err := h.repo.ListByMuscleGroup(c.Request.Context(), uint(id), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, types)
}

func defaultMetric(value string) string {
	if value == "" {
		return "reps"
	}
	return value
}
