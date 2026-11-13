package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VibeTeam/fitness-tracker-backend/workout/handler"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository/gormrepository"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeCoach struct {
	question, history string
	err               error
}

func (f *fakeCoach) Coach(_ context.Context, history, question string) (string, error) {
	f.history = history
	f.question = question
	return "Keep tracking your progress.", f.err
}
func TestProfileEndpoints(t *testing.T) {
	r, db := testRouter(t)
	coach := &fakeCoach{}
	h := handler.NewProfileHandler(gormrepository.NewTrainingRepository(db), coach)
	h.RegisterRoutes(r, func(c *gin.Context) { c.Set("user_id", uint(1)) })
	group := models.MuscleGroup{Name: "Chest"}
	require.NoError(t, db.Create(&group).Error)
	exercise := models.WorkoutType{Name: "Bench", MuscleGroupID: group.ID}
	require.NoError(t, db.Create(&exercise).Error)
	ptr := func(v int) *int { return &v }
	now := time.Now().UTC()
	seed := func(uid uint, at time.Time, weight int) {
		tr := models.Training{UserID: uid, Title: "Training", StartedAt: at, Sessions: []models.WorkoutSession{{UserID: uid, WorkoutTypeID: exercise.ID, Datetime: at, Sets: []models.WorkoutSet{
			{SetNumber: 1, Weight: ptr(weight), Reps: ptr(10)},
			{SetNumber: 2, Weight: ptr(0), Reps: ptr(8)},
			{SetNumber: 3},
		}}}}
		require.NoError(t, db.Create(&tr).Error)
	}
	seed(1, now.AddDate(0, 0, -3), 60)
	seed(1, now.AddDate(0, 0, -1), 75)
	seed(1, now.AddDate(0, 0, -100), 999)
	seed(1, now.AddDate(0, 0, 1), 999)
	seed(2, now.AddDate(0, 0, -1), 999)
	request := func(method, path, body string, code int) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, code, w.Code, w.Body.String())
		var out map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		return out
	}
	overview := request("GET", "/profile/overview", "", 200)
	require.EqualValues(t, 2, overview["workouts"])
	require.EqualValues(t, 4, overview["sets"])
	require.EqualValues(t, 1350, overview["volume_kg"])
	groups := overview["muscle_balance"].([]any)
	require.Len(t, groups, 1)
	require.EqualValues(t, 100, groups[0].(map[string]any)["share_percent"])
	progress := request("GET", "/profile/strength-progress", "", 200)
	points := progress["points"].([]any)
	require.Len(t, points, 2)
	require.EqualValues(t, 80, points[0].(map[string]any)["estimated_1rm_kg"])
	require.EqualValues(t, 100, points[1].(map[string]any)["estimated_1rm_kg"])
	require.EqualValues(t, 25, progress["change_percent"])
	empty := request("GET", "/profile/strength-progress?workout_type_id=999", "", 200)
	require.Empty(t, empty["points"])
	require.Nil(t, empty["change_percent"])
	for _, query := range []string{"from=bad", "to=bad", "workout_type_id=-1", "workout_type_id=0", "from=2020-01-01", "from=2026-01-02&to=2026-01-01"} {
		request("GET", "/profile/strength-progress?"+query, "", 400)
	}
	request("POST", "/profile/coach/messages", `{"question":"How am I doing?"}`, 200)
	require.Equal(t, "How am I doing?", coach.question)
	require.Contains(t, coach.history, `"volume_kg":1350`)
	require.NotContains(t, coach.history, "999")
	request("POST", "/profile/coach/messages", `{}`, 200)
	require.Empty(t, coach.question)
	request("POST", "/profile/coach/messages", `{"conversation_id":"old"}`, 400)
	request("POST", "/profile/coach/messages", `{"question":"`+strings.Repeat("x", 2001)+`"}`, 400)
	coach.err = errors.New("sensitive upstream details")
	failure := request("POST", "/profile/coach/messages", `{}`, 502)
	require.NotContains(t, failure["error"], "sensitive")
}

func TestProfileEmptyAndAuthentication(t *testing.T) {
	_, db := testRouter(t)
	for _, authenticated := range []bool{false, true} {
		r := gin.New()
		h := handler.NewProfileHandler(gormrepository.NewTrainingRepository(db), nil)
		h.RegisterRoutes(r, func(c *gin.Context) {
			if authenticated {
				c.Set("user_id", uint(1))
			}
		})
		for _, path := range []string{"/profile/overview", "/profile/strength-progress", "/profile/coach/messages"} {
			method := "GET"
			if strings.Contains(path, "coach") {
				method = "POST"
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(`{}`)))
			if !authenticated {
				require.Equal(t, 401, w.Code)
				continue
			}
			if method == "POST" {
				require.Equal(t, 503, w.Code)
				continue
			}
			require.Equal(t, 200, w.Code)
			if strings.Contains(path, "strength") {
				require.Contains(t, w.Body.String(), `"points":[]`)
				require.Contains(t, w.Body.String(), `"change_percent":null`)
			} else {
				require.Contains(t, w.Body.String(), `"muscle_balance":[]`)
			}
		}
	}
}
