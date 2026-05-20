package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type WorkoutTypeHandler struct {
	repo repository.WorkoutTypeRepository
}

func NewWorkoutTypeHandler(repo repository.WorkoutTypeRepository) *WorkoutTypeHandler {
	return &WorkoutTypeHandler{repo: repo}
}

func (h *WorkoutTypeHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	wt := r.Group("/workout-types")
	wt.Use(auth)
	{
		wt.POST("", h.create)
		wt.GET("", h.list)
		wt.GET("/:id", h.getByID)
		wt.PUT("/:id", h.update)
		wt.DELETE("/:id", h.delete)
	}
	r.GET("/muscle-groups/:id/workout-types", auth, h.listByMuscleGroup)
}

type workoutTypeRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	ImageURL      string `json:"image_url"`
	DefaultMetric string `json:"default_metric"`
	MuscleGroupID uint   `json:"muscle_group_id" binding:"required"`
}

// create workout type
// @Summary      Create workout type
// @Tags         workout-types
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      workoutTypeRequest  true  "Workout type"
// @Success      201      {object}  models.WorkoutType
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /workout-types [post]
func (h *WorkoutTypeHandler) create(c *gin.Context) {
	var req workoutTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wt := &models.WorkoutType{
		Name:          req.Name,
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		DefaultMetric: defaultMetric(req.DefaultMetric),
		MuscleGroupID: req.MuscleGroupID,
	}
	if err := h.repo.Create(c.Request.Context(), wt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Retrieve with association to include muscle group name
	if loaded, err := h.repo.GetByID(c.Request.Context(), wt.ID); err == nil {
		wt = loaded
	}
	c.JSON(http.StatusCreated, wt)
}

// list workout types
// @Summary      List workout types
// @Tags         workout-types
// @Security     BearerAuth
// @Produce      json
// @Param        limit            query     int     false  "Limit"
// @Param        offset           query     int     false  "Offset"
// @Param        query            query     string  false  "Search query"
// @Param        muscle_group_id  query     int     false  "MuscleGroup ID"
// @Success      200  {array}   models.WorkoutType
// @Router       /workout-types [get]
func (h *WorkoutTypeHandler) list(c *gin.Context) {
	limit := queryInt(c, "limit", 100)
	offset := queryInt(c, "offset", 0)
	query := c.Query("query")
	var muscleGroupID *uint
	if raw := c.Query("muscle_group_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid muscle_group_id"})
			return
		}
		value := uint(id)
		muscleGroupID = &value
	}
	types, err := h.repo.Search(c.Request.Context(), query, muscleGroupID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, types)
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

// get workout type
// @Summary      Get workout type by ID
// @Tags         workout-types
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      int  true  "WorkoutType ID"
// @Success      200  {object}  models.WorkoutType
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /workout-types/{id} [get]
func (h *WorkoutTypeHandler) getByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	wt, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wt)
}

// update workout type
// @Summary      Update workout type
// @Tags         workout-types
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      int                 true  "WorkoutType ID"
// @Param        payload  body      workoutTypeRequest  true  "Update"
// @Success      200      {object}  models.WorkoutType
// @Failure      400      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Router       /workout-types/{id} [put]
func (h *WorkoutTypeHandler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	wt, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var req workoutTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wt.Name = req.Name
	wt.Description = req.Description
	wt.ImageURL = req.ImageURL
	wt.DefaultMetric = defaultMetric(req.DefaultMetric)
	wt.MuscleGroupID = req.MuscleGroupID
	if err := h.repo.Update(c.Request.Context(), wt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wt)
}

func defaultMetric(value string) string {
	if value == "" {
		return "reps"
	}
	return value
}

// delete workout type
// @Summary      Delete workout type
// @Tags         workout-types
// @Security     BearerAuth
// @Param        id   path      int  true  "WorkoutType ID"
// @Success      204  {string}  string  "No Content"
// @Failure      400  {object}  errorResponse
// @Router       /workout-types/{id} [delete]
func (h *WorkoutTypeHandler) delete(c *gin.Context) {
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
