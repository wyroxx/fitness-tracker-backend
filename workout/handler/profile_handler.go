package handler

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/VibeTeam/fitness-tracker-backend/shared/middleware"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
	"github.com/gin-gonic/gin"
)

type ProfileCoach interface {
	Coach(context.Context, string, string) (string, error)
}
type ProfileHandler struct {
	repo  repository.TrainingRepository
	coach ProfileCoach
}

func NewProfileHandler(repo repository.TrainingRepository, coach ProfileCoach) *ProfileHandler {
	return &ProfileHandler{repo: repo, coach: coach}
}
func (h *ProfileHandler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	g := r.Group("/profile", auth)
	g.GET("/overview", h.overview)
	g.GET("/strength-progress", h.strengthProgress)
	g.POST("/coach/messages", h.coachMessage)
}

type muscleBalance struct {
	MuscleGroupID uint    `json:"muscle_group_id"`
	Name          string  `json:"name"`
	Sets          int     `json:"sets"`
	SharePercent  float64 `json:"share_percent"`
}
type overviewResponse struct {
	From          time.Time       `json:"from"`
	To            time.Time       `json:"to"`
	Workouts      int             `json:"workouts"`
	Sets          int             `json:"sets"`
	VolumeKg      float64         `json:"volume_kg"`
	MuscleBalance []muscleBalance `json:"muscle_balance"`
}

// A set with no positive repetition count is not a completed strength set.
func completedSet(s models.WorkoutSet) bool { return s.Reps != nil && *s.Reps > 0 }
func rounded(v float64) float64             { return math.Round(v*100) / 100 }
func summarize(trainings []*models.Training, from, to time.Time) overviewResponse {
	result := overviewResponse{From: from, To: to, Workouts: len(trainings), MuscleBalance: []muscleBalance{}}
	groups := map[uint]*muscleBalance{}
	for _, t := range trainings {
		for _, session := range t.Sessions {
			id, name := uint(0), "Unknown"
			if session.WorkoutType != nil {
				id = session.WorkoutType.MuscleGroupID
				if session.WorkoutType.MuscleGroup != nil {
					name = session.WorkoutType.MuscleGroup.Name
				}
			}
			for _, set := range session.Sets {
				if !completedSet(set) {
					continue
				}
				result.Sets++
				if set.Weight != nil && *set.Weight > 0 {
					result.VolumeKg += float64(*set.Weight) * float64(*set.Reps)
				}
				if groups[id] == nil {
					groups[id] = &muscleBalance{MuscleGroupID: id, Name: name}
				}
				groups[id].Sets++
			}
		}
	}
	for _, group := range groups {
		group.SharePercent = rounded(100 * float64(group.Sets) / float64(result.Sets))
		result.MuscleBalance = append(result.MuscleBalance, *group)
	}
	sort.Slice(result.MuscleBalance, func(i, j int) bool {
		return result.MuscleBalance[i].MuscleGroupID < result.MuscleBalance[j].MuscleGroupID
	})
	return result
}
func (h *ProfileHandler) load(c *gin.Context, from, to time.Time) ([]*models.Training, bool) {
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(401, errorResponse{"missing user"})
		return nil, false
	}
	data, err := h.repo.ListByUserBetween(c.Request.Context(), uid, from, to)
	if err != nil {
		c.JSON(500, errorResponse{"could not load training history"})
		return nil, false
	}
	return data, true
}

// @Summary Profile overview for the last seven days
// @Description UTC rolling interval [from,to). Sets require positive reps; volume is sum(weight*reps) in kg. Muscle balance is each primary group's share of counted sets, not a target score.
// @Tags profile
// @Security BearerAuth
// @Produce json
// @Success 200 {object} overviewResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /profile/overview [get]
func (h *ProfileHandler) overview(c *gin.Context) {
	to := time.Now().UTC()
	from := to.AddDate(0, 0, -7)
	data, ok := h.load(c, from, to)
	if !ok {
		return
	}
	c.JSON(200, summarize(data, from, to))
}

type progressExercise struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}
type progressPoint struct {
	TrainingID     uint      `json:"training_id"`
	Date           time.Time `json:"date"`
	Estimated1RMKg float64   `json:"estimated_1rm_kg"`
}
type progressResponse struct {
	From          time.Time          `json:"from"`
	To            time.Time          `json:"to"`
	WorkoutTypeID *uint              `json:"workout_type_id"`
	Exercises     []progressExercise `json:"exercises"`
	Points        []progressPoint    `json:"points"`
	ChangePercent *float64           `json:"change_percent"`
}

func estimated1RM(set models.WorkoutSet) float64 {
	if set.Weight == nil || *set.Weight <= 0 || !completedSet(set) || *set.Reps > 10 {
		return 0
	}
	if *set.Reps == 1 {
		return float64(*set.Weight)
	}
	return float64(*set.Weight) * (1 + float64(*set.Reps)/30)
}

// @Summary Estimated strength progress
// @Description Dates are UTC YYYY-MM-DD, inclusive calendar dates; default last 90 days, maximum 366 days. Epley for 2-10 reps; 1 rep uses actual weight. Best set per training. Change compares first and last points; null with fewer than two points. Exercises lists those with eligible sets in this period. Without an ID, selects the lowest available ID.
// @Tags profile
// @Security BearerAuth
// @Produce json
// @Param workout_type_id query int false "Exercise ID"
// @Param from query string false "First date (YYYY-MM-DD)"
// @Param to query string false "Last date (YYYY-MM-DD)"
// @Success 200 {object} progressResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /profile/strength-progress [get]
func (h *ProfileHandler) strengthProgress(c *gin.Context) {
	now := time.Now().UTC()
	to := now
	from := now.AddDate(0, 0, -90)
	if raw, exists := c.GetQuery("to"); exists {
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			c.JSON(400, errorResponse{"invalid to date"})
			return
		}
		to = date.AddDate(0, 0, 1)
		if to.After(now) {
			to = now
		}
		from = to.AddDate(0, 0, -90)
	}
	if raw, exists := c.GetQuery("from"); exists {
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			c.JSON(400, errorResponse{"invalid from date"})
			return
		}
		from = date
	}
	if !from.Before(to) || to.Sub(from) > 366*24*time.Hour {
		c.JSON(400, errorResponse{"date range must be positive and at most 366 days"})
		return
	}
	var selected *uint
	if raw, exists := c.GetQuery("workout_type_id"); exists {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || id == 0 {
			c.JSON(400, errorResponse{"invalid workout_type_id"})
			return
		}
		value := uint(id)
		selected = &value
	}
	data, ok := h.load(c, from, to)
	if !ok {
		return
	}
	result := progressResponse{From: from, To: to, WorkoutTypeID: selected, Exercises: []progressExercise{}, Points: []progressPoint{}}
	exercises := map[uint]string{}
	for _, t := range data {
		for _, session := range t.Sessions {
			for _, set := range session.Sets {
				if estimated1RM(set) > 0 {
					name := ""
					if session.WorkoutType != nil {
						name = session.WorkoutType.Name
					}
					exercises[session.WorkoutTypeID] = name
				}
			}
		}
	}
	for id, name := range exercises {
		result.Exercises = append(result.Exercises, progressExercise{id, name})
	}
	sort.Slice(result.Exercises, func(i, j int) bool { return result.Exercises[i].ID < result.Exercises[j].ID })
	if selected == nil && len(result.Exercises) > 0 {
		id := result.Exercises[0].ID
		selected = &id
		result.WorkoutTypeID = selected
	}
	if selected != nil {
		for _, t := range data {
			best := float64(0)
			for _, session := range t.Sessions {
				if session.WorkoutTypeID == *selected {
					for _, set := range session.Sets {
						best = math.Max(best, estimated1RM(set))
					}
				}
			}
			if best > 0 {
				result.Points = append(result.Points, progressPoint{t.ID, t.StartedAt, rounded(best)})
			}
		}
	}
	if len(result.Points) > 1 {
		first := result.Points[0].Estimated1RMKg
		last := result.Points[len(result.Points)-1].Estimated1RMKg
		change := rounded((last/first - 1) * 100)
		result.ChangePercent = &change
	}
	c.JSON(200, result)
}

type coachRequest struct {
	Question string `json:"question" binding:"max=2000"`
}
type coachResponse struct {
	Answer string `json:"answer"`
}

// @Summary Ask the AI coach without a conversation
// @Description Each request is independent. Send {} for an overview or a question up to 2000 characters. Context contains only the authenticated user's seven-day summary. No messages are stored.
// @Tags profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body coachRequest true "Optional question"
// @Success 200 {object} coachResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 413 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Failure 503 {object} errorResponse
// @Router /profile/coach/messages [post]
func (h *ProfileHandler) coachMessage(c *gin.Context) {
	if _, ok := middleware.UserID(c); !ok {
		c.JSON(401, errorResponse{"missing user"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var req coachRequest
	if err := bindWorkoutJSON(c, &req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(413, errorResponse{"request body is too large"})
			return
		}
		c.JSON(400, errorResponse{"expected an object with an optional question of at most 2000 characters"})
		return
	}
	to := time.Now().UTC()
	from := to.AddDate(0, 0, -7)
	data, ok := h.load(c, from, to)
	if !ok {
		return
	}
	if h.coach == nil {
		c.JSON(503, errorResponse{"AI coach is not configured"})
		return
	}
	contextJSON, _ := json.Marshal(summarize(data, from, to))
	answer, err := h.coach.Coach(c.Request.Context(), string(contextJSON), strings.TrimSpace(req.Question))
	if err != nil {
		c.JSON(502, errorResponse{"AI coach is temporarily unavailable"})
		return
	}
	c.JSON(200, coachResponse{Answer: answer})
}
