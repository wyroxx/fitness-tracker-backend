package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/VibeTeam/fitness-tracker-backend/llm/suggester"
	"github.com/VibeTeam/fitness-tracker-backend/shared/middleware"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

// SuggestHandler returns AI-based workout suggestions.
type SuggestHandler struct {
	trainingRepo repository.TrainingRepository
	suggester    *suggester.Suggester
}

type suggestionResponse struct {
	Suggestion string `json:"suggestion"`
}

// errorResponse is used for Swagger documentation of error payloads.
type errorResponse struct {
	Error string `json:"error"`
}

func NewSuggestHandler(repo repository.TrainingRepository, sg *suggester.Suggester) *SuggestHandler {
	return &SuggestHandler{trainingRepo: repo, suggester: sg}
}

func (h *SuggestHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	g := r.Group("/suggest-workout")
	g.Use(auth)
	g.GET("", h.suggest)
}

// Suggest workout
// @Summary      Suggest next workout
// @Tags         workout-suggestions
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  suggestionResponse
// @Failure      500  {object}  errorResponse
// @Router       /suggest-workout [get]
func (h *SuggestHandler) suggest(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	// Fetch last 10 user-facing trainings with nested exercise sessions and sets.
	trainings, err := h.trainingRepo.ListByUser(c.Request.Context(), uid, 10, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(trainings) == 0 {
		c.JSON(http.StatusOK, suggestionResponse{Suggestion: "No history yet. Start with a full-body beginner routine."})
		return
	}
	var parts []string
	for idx, training := range trainings {
		var exerciseParts []string
		for _, session := range training.Sessions {
			if session.WorkoutType == nil {
				continue
			}
			exercise := session.WorkoutType.Name
			if len(session.Sets) > 0 {
				exercise += " (" + setSummary(session.Sets) + ")"
			}
			exerciseParts = append(exerciseParts, exercise)
		}
		line := "Training " + strconv.Itoa(idx+1) + ": " + training.Title +
			" on " + training.StartedAt.Format("2006-01-02")
		if len(exerciseParts) > 0 {
			line += " - " + strings.Join(exerciseParts, "; ")
		}
		parts = append(parts, line)
	}
	history := strings.Join(parts, "\n")
	suggestion, err := h.suggester.Suggest(history)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, suggestionResponse{Suggestion: suggestion})
}

func setSummary(sets []models.WorkoutSet) string {
	var setParts []string
	for _, set := range sets {
		var fields []string
		if set.WeightKg != nil {
			fields = append(fields, "weight="+strconv.FormatFloat(*set.WeightKg, 'f', -1, 64)+"kg")
		}
		if set.Reps != nil {
			fields = append(fields, "reps="+strconv.Itoa(*set.Reps))
		}
		if set.DurationSeconds != nil {
			fields = append(fields, "duration="+strconv.Itoa(*set.DurationSeconds)+"s")
		}
		if set.DistanceMeters != nil {
			fields = append(fields, "distance="+strconv.FormatFloat(*set.DistanceMeters, 'f', -1, 64)+"m")
		}
		if len(fields) == 0 {
			setParts = append(setParts, "set "+strconv.Itoa(set.SetNumber))
			continue
		}
		setParts = append(setParts, "set "+strconv.Itoa(set.SetNumber)+": "+strings.Join(fields, ", "))
	}
	return strings.Join(setParts, "; ")
}
