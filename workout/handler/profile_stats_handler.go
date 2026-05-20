package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/shared/middleware"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type ProfileStatsHandler struct {
	trainingRepo repository.TrainingRepository
	sessionRepo  repository.WorkoutSessionRepository
}

func NewProfileStatsHandler(trainingRepo repository.TrainingRepository, sessionRepo repository.WorkoutSessionRepository) *ProfileStatsHandler {
	return &ProfileStatsHandler{trainingRepo: trainingRepo, sessionRepo: sessionRepo}
}

func (h *ProfileStatsHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	g := r.Group("/profile")
	g.Use(auth)
	g.GET("/stats", h.getStats)
}

type profileStatsResponse struct {
	TotalWorkouts         int        `json:"total_workouts"`
	TotalExerciseSessions int        `json:"total_exercise_sessions"`
	TrainingsThisWeek     int        `json:"trainings_this_week"`
	LastTrainingAt        *time.Time `json:"last_training_at,omitempty"`
	AIInsight             string     `json:"ai_insight"`
}

// Get profile stats
// @Summary      Get current user's workout stats
// @Tags         profile
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  profileStatsResponse
// @Failure      401  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /profile/stats [get]
func (h *ProfileStatsHandler) getStats(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}

	totalTrainings, err := h.trainingRepo.CountByUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	totalSessions, err := h.sessionRepo.CountByUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	weekAgo := time.Now().AddDate(0, 0, -7)
	trainingsThisWeek, err := h.trainingRepo.CountByUserSince(c.Request.Context(), uid, weekAgo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latest, err := h.trainingRepo.ListByUser(c.Request.Context(), uid, 1, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := profileStatsResponse{
		TotalWorkouts:         totalTrainings,
		TotalExerciseSessions: totalSessions,
		TrainingsThisWeek:     trainingsThisWeek,
		AIInsight:             insightFor(trainingsThisWeek, totalTrainings),
	}
	if len(latest) > 0 {
		resp.LastTrainingAt = &latest[0].StartedAt
	}
	c.JSON(http.StatusOK, resp)
}

func insightFor(trainingsThisWeek, totalTrainings int) string {
	switch {
	case totalTrainings == 0:
		return "Start with a light full-body training and save your first workout."
	case trainingsThisWeek >= 3:
		return "Great consistency this week. Keep tracking sets so progress stays visible."
	case trainingsThisWeek == 0:
		return "No trainings logged in the last 7 days. A short session is enough to restart the rhythm."
	default:
		return "Good start this week. Add one more training to build momentum."
	}
}
