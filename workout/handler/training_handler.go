package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/shared/middleware"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type TrainingHandler struct {
	repo         repository.TrainingRepository
	exerciseRepo repository.WorkoutTypeRepository
}

func NewTrainingHandler(repo repository.TrainingRepository, exerciseRepo repository.WorkoutTypeRepository) *TrainingHandler {
	return &TrainingHandler{repo: repo, exerciseRepo: exerciseRepo}
}

func (h *TrainingHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	g := r.Group("/trainings")
	g.Use(auth)
	{
		g.POST("", h.create)
		g.GET("", h.list)
		g.DELETE("/:id", h.delete)
	}
}

// trainingRequest matches CreateWorkoutRequest in the Swift app.
type trainingRequest struct {
	Title     string           `json:"title" binding:"required"`
	StartedAt time.Time        `json:"started_at" binding:"required"`
	Sessions  []workoutRequest `json:"sessions" binding:"required,len=1,dive"`
}

type workoutRequest struct {
	Title     string                   `json:"title" binding:"required"`
	StartedAt time.Time                `json:"started_at" binding:"required"`
	Exercises []workoutExerciseRequest `json:"exercises" binding:"required,dive"`
}

type workoutExerciseRequest struct {
	ID   uint                `json:"id" binding:"required"`
	Name string              `json:"name" binding:"required"`
	Sets []workoutSetRequest `json:"sets" binding:"required,dive"`
}

// Create training
// @Summary      Create training from the Swift CreateWorkoutRequest
// @Description  sessions contains one Workout with title, started_at and exercises, as sent by the current Swift app.
// @Tags         trainings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      trainingRequest  true  "Training"
// @Success      201      {object}  workoutResponse
// @Failure      400      {object}  errorResponse
// @Failure      401      {object}  errorResponse
// @Failure      500      {object}  errorResponse
// @Router       /trainings [post]
func (h *TrainingHandler) create(c *gin.Context) {
	var req trainingRequest
	if err := bindWorkoutJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	workout := req.Sessions[0]
	if req.Title != workout.Title || !req.StartedAt.Equal(workout.StartedAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session title and started_at must match the enclosing training"})
		return
	}
	training := &models.Training{
		UserID: uid, Title: req.Title, StartedAt: req.StartedAt,
		Sessions: make([]models.WorkoutSession, len(workout.Exercises)),
	}
	for i, exercise := range workout.Exercises {
		if _, err := h.exerciseRepo.GetByID(c.Request.Context(), exercise.ID); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"error": "exercise could not be loaded"})
			return
		}
		training.Sessions[i] = models.WorkoutSession{
			UserID: uid, WorkoutTypeID: exercise.ID, OrderIndex: i + 1,
			Datetime: req.StartedAt, Sets: workoutSetModels(exercise.Sets),
		}
	}

	if err := h.repo.Create(c.Request.Context(), training); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	loaded, err := h.repo.GetByID(c.Request.Context(), training.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, workoutFromTraining(loaded))
}

// List trainings
// @Summary      List trainings for current user
// @Tags         trainings
// @Security     BearerAuth
// @Produce      json
// @Description  Returns all current-user workouts as a bare array matching [Workout] in Swift.
// @Success      200     {array} workoutResponse
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
	trainings, err := h.repo.ListByUser(c.Request.Context(), uid, -1, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]workoutResponse, len(trainings))
	for i, training := range trainings {
		resp[i] = workoutFromTraining(training)
	}
	c.JSON(http.StatusOK, resp)
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
