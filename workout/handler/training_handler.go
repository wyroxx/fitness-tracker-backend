package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/shared/middleware"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type TrainingHandler struct {
	repo repository.TrainingRepository
}

func NewTrainingHandler(repo repository.TrainingRepository) *TrainingHandler {
	return &TrainingHandler{repo: repo}
}

func (h *TrainingHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	g := r.Group("/trainings")
	g.Use(auth)
	{
		g.POST("", h.create)
		g.GET("", h.list)
		g.GET("/:id", h.getByID)
		g.PATCH("/:id", h.update)
		g.DELETE("/:id", h.delete)
	}
}

type trainingRequest struct {
	Title      string                   `json:"title"`
	StartedAt  time.Time                `json:"started_at"`
	FinishedAt *time.Time               `json:"finished_at"`
	Notes      string                   `json:"notes"`
	Sessions   []trainingSessionRequest `json:"sessions"`
}

type trainingSessionRequest struct {
	WorkoutTypeID uint                `json:"workout_type_id" binding:"required"`
	OrderIndex    int                 `json:"order_index"`
	Datetime      time.Time           `json:"datetime"`
	Sets          []workoutSetRequest `json:"sets"`
}

type trainingUpdateRequest struct {
	Title      *string    `json:"title"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Notes      *string    `json:"notes"`
}

type trainingListResponse struct {
	Items      []*models.Training `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// Create training
// @Summary      Create training with nested sessions and sets
// @Tags         trainings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      trainingRequest  true  "Training"
// @Success      201      {object}  models.Training
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /trainings [post]
func (h *TrainingHandler) create(c *gin.Context) {
	var req trainingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	if req.StartedAt.IsZero() {
		req.StartedAt = time.Now()
	}
	if req.Title == "" {
		req.Title = "Workout"
	}

	training := &models.Training{
		UserID:     uid,
		Title:      req.Title,
		StartedAt:  req.StartedAt,
		FinishedAt: req.FinishedAt,
		Notes:      req.Notes,
		Sessions:   make([]models.WorkoutSession, len(req.Sessions)),
	}
	for i, sessionReq := range req.Sessions {
		sessionTime := sessionReq.Datetime
		if sessionTime.IsZero() {
			sessionTime = req.StartedAt
		}
		orderIndex := sessionReq.OrderIndex
		if orderIndex == 0 {
			orderIndex = i + 1
		}
		training.Sessions[i] = models.WorkoutSession{
			UserID:        uid,
			WorkoutTypeID: sessionReq.WorkoutTypeID,
			OrderIndex:    orderIndex,
			Datetime:      sessionTime,
			Sets:          workoutSetModels(sessionReq.Sets),
		}
	}

	if err := h.repo.Create(c.Request.Context(), training); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	loaded, err := h.repo.GetByID(c.Request.Context(), training.ID)
	if err != nil {
		c.JSON(http.StatusCreated, training)
		return
	}
	c.JSON(http.StatusCreated, loaded)
}

// List trainings
// @Summary      List trainings for current user
// @Tags         trainings
// @Security     BearerAuth
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        cursor  query     string  false  "RFC3339 started_at cursor"
// @Success      200     {object} trainingListResponse
// @Failure      400     {object}  errorResponse
// @Failure      401     {object}  errorResponse
// @Failure      500     {object}  errorResponse
// @Router       /trainings [get]
func (h *TrainingHandler) list(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	limit := queryInt(c, "limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var cursor *time.Time
	if raw := c.Query("cursor"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return
		}
		cursor = &parsed
	}
	trainings, err := h.repo.ListByUser(c.Request.Context(), uid, limit+1, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := trainingListResponse{Items: trainings}
	if len(trainings) > limit {
		resp.Items = trainings[:limit]
		resp.NextCursor = resp.Items[len(resp.Items)-1].StartedAt.Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, resp)
}

// Get training
// @Summary      Get training by ID
// @Tags         trainings
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      int  true  "Training ID"
// @Success      200  {object}  models.Training
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /trainings/{id} [get]
func (h *TrainingHandler) getByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	training, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	uid, _ := middleware.UserID(c)
	if training.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, training)
}

// Update training
// @Summary      Update training metadata
// @Tags         trainings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      int                    true  "Training ID"
// @Param        payload  body      trainingUpdateRequest  true  "Update"
// @Success      200      {object}  models.Training
// @Failure      400      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /trainings/{id} [patch]
func (h *TrainingHandler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	training, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	uid, _ := middleware.UserID(c)
	if training.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var req trainingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title != nil {
		training.Title = *req.Title
	}
	if req.StartedAt != nil {
		training.StartedAt = *req.StartedAt
	}
	if req.FinishedAt != nil {
		training.FinishedAt = req.FinishedAt
	}
	if req.Notes != nil {
		training.Notes = *req.Notes
	}
	if err := h.repo.Update(c.Request.Context(), training); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	loaded, err := h.repo.GetByID(c.Request.Context(), training.ID)
	if err != nil {
		c.JSON(http.StatusOK, training)
		return
	}
	c.JSON(http.StatusOK, loaded)
}

// Delete training
// @Summary      Delete training
// @Tags         trainings
// @Security     BearerAuth
// @Param        id   path      int  true  "Training ID"
// @Success      204  {string}  string  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /trainings/{id} [delete]
func (h *TrainingHandler) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	training, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	uid, _ := middleware.UserID(c)
	if training.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func queryInt(c *gin.Context, name string, fallback int) int {
	value, err := strconv.Atoi(c.DefaultQuery(name, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}
