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

type WorkoutSessionHandler struct {
	repo         repository.WorkoutSessionRepository
	trainingRepo repository.TrainingRepository
	setRepo      repository.WorkoutSetRepository
}

func NewWorkoutSessionHandler(repo repository.WorkoutSessionRepository, trainingRepo repository.TrainingRepository, setRepo repository.WorkoutSetRepository) *WorkoutSessionHandler {
	return &WorkoutSessionHandler{repo: repo, trainingRepo: trainingRepo, setRepo: setRepo}
}

func (h *WorkoutSessionHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	ws := r.Group("/workout-sessions")
	ws.Use(auth)
	{
		ws.POST("", h.create)
		ws.GET("", h.list)
		ws.GET("/:id", h.getByID)
		ws.DELETE("/:id", h.delete)
		ws.GET("/:id/sets", h.listSets)
		ws.PUT("/:id/sets", h.replaceSets)
	}
}

type workoutSessionRequest struct {
	TrainingID    uint                `json:"training_id" binding:"required"`
	WorkoutTypeID uint                `json:"workout_type_id" binding:"required"`
	OrderIndex    int                 `json:"order_index"`
	Datetime      time.Time           `json:"datetime"`
	Sets          []workoutSetRequest `json:"sets"`
}

type workoutSetRequest struct {
	SetNumber       int      `json:"set_number"`
	WeightKg        *float64 `json:"weight_kg"`
	Reps            *int     `json:"reps"`
	DurationSeconds *int     `json:"duration_seconds"`
	DistanceMeters  *float64 `json:"distance_meters"`
	Notes           string   `json:"notes"`
}

type replaceWorkoutSetsRequest struct {
	Sets []workoutSetRequest `json:"sets"`
}

// create session
// @Summary      Create workout session
// @Tags         workout-sessions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      workoutSessionRequest  true  "Session"
// @Success      201      {object}  models.WorkoutSession
// @Failure      400      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /workout-sessions [post]
func (h *WorkoutSessionHandler) create(c *gin.Context) {
	var req workoutSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	if req.Datetime.IsZero() {
		req.Datetime = time.Now()
	}
	training, err := h.trainingRepo.GetByID(c.Request.Context(), req.TrainingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if training.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	session := &models.WorkoutSession{
		UserID:        uid,
		TrainingID:    req.TrainingID,
		WorkoutTypeID: req.WorkoutTypeID,
		OrderIndex:    req.OrderIndex,
		Datetime:      req.Datetime,
		Sets:          workoutSetModels(req.Sets),
	}
	if err := h.repo.Create(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	loaded, err := h.repo.GetByID(c.Request.Context(), session.ID)
	if err != nil {
		c.JSON(http.StatusCreated, session)
		return
	}
	c.JSON(http.StatusCreated, loaded)
}

// list sessions for user
// @Summary      List workout sessions for user
// @Tags         workout-sessions
// @Security     BearerAuth
// @Produce      json
// @Param        limit   query     int  false  "Limit"
// @Param        offset  query     int  false  "Offset"
// @Param        training_id  query  int  false  "Training ID"
// @Success      200  {array}   models.WorkoutSession
// @Router       /workout-sessions [get]
func (h *WorkoutSessionHandler) list(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	var sessions []*models.WorkoutSession
	var err error
	if trainingIDRaw := c.Query("training_id"); trainingIDRaw != "" {
		trainingID, parseErr := strconv.Atoi(trainingIDRaw)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid training_id"})
			return
		}
		sessions, err = h.repo.ListByTraining(c.Request.Context(), uint(trainingID))
	} else {
		sessions, err = h.repo.ListByUser(c.Request.Context(), uid, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, session := range sessions {
		if session.UserID != uid {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
	}
	c.JSON(http.StatusOK, sessions)
}

// get session
// @Summary      Get workout session by ID
// @Tags         workout-sessions
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      int  true  "WorkoutSession ID"
// @Success      200  {object}  models.WorkoutSession
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /workout-sessions/{id} [get]
func (h *WorkoutSessionHandler) getByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	session, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	uid, _ := middleware.UserID(c)
	if session.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, session)
}

// delete session
// @Summary      Delete workout session
// @Tags         workout-sessions
// @Security     BearerAuth
// @Param        id   path      int  true  "WorkoutSession ID"
// @Success      204  {string}  string  "No Content"
// @Failure      400  {object}  errorResponse
// @Router       /workout-sessions/{id} [delete]
func (h *WorkoutSessionHandler) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	session, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	uid, _ := middleware.UserID(c)
	if session.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// List workout sets
// @Summary      List sets for a workout session
// @Tags         workout-sessions
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      int  true  "WorkoutSession ID"
// @Success      200  {array}   models.WorkoutSet
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /workout-sessions/{id}/sets [get]
func (h *WorkoutSessionHandler) listSets(c *gin.Context) {
	session, ok := h.sessionFromParam(c)
	if !ok {
		return
	}
	sets, err := h.setRepo.ListBySession(c.Request.Context(), session.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sets)
}

// Replace workout sets
// @Summary      Replace all sets for a workout session
// @Tags         workout-sessions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      int                        true  "WorkoutSession ID"
// @Param        payload  body      replaceWorkoutSetsRequest  true  "Sets"
// @Success      200      {object}  models.WorkoutSession
// @Failure      400      {object}  errorResponse
// @Failure      404      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /workout-sessions/{id}/sets [put]
func (h *WorkoutSessionHandler) replaceSets(c *gin.Context) {
	session, ok := h.sessionFromParam(c)
	if !ok {
		return
	}
	var req replaceWorkoutSetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sets := workoutSetModels(req.Sets)
	if err := h.setRepo.ReplaceForSession(c.Request.Context(), session.ID, sets); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	loaded, err := h.repo.GetByID(c.Request.Context(), session.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"sets": sets})
		return
	}
	c.JSON(http.StatusOK, loaded)
}

func (h *WorkoutSessionHandler) sessionFromParam(c *gin.Context) (*models.WorkoutSession, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	session, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return nil, false
	}
	uid, _ := middleware.UserID(c)
	if session.UserID != uid {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return nil, false
	}
	return session, true
}

func workoutSetModels(reqs []workoutSetRequest) []models.WorkoutSet {
	sets := make([]models.WorkoutSet, len(reqs))
	for i, req := range reqs {
		setNumber := req.SetNumber
		if setNumber == 0 {
			setNumber = i + 1
		}
		sets[i] = models.WorkoutSet{
			SetNumber:       setNumber,
			WeightKg:        req.WeightKg,
			Reps:            req.Reps,
			DurationSeconds: req.DurationSeconds,
			DistanceMeters:  req.DistanceMeters,
			Notes:           req.Notes,
		}
	}
	return sets
}
