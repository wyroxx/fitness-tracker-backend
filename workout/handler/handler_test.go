package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/handler"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository/gormrepository"
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// testRouter creates an isolated Gin engine backed by an in‑mem SQLite DB and
// registers only the routes needed for these tests.
func testRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)

	// migrate the minimal set of tables we touch
	require.NoError(t, db.AutoMigrate(&models.MuscleGroup{}, &models.WorkoutType{},
		&models.Training{}, &models.WorkoutSession{}, &models.WorkoutSet{}))

	// repositories
	mgRepo := gormrepository.NewMuscleGroupRepository(db)
	wtRepo := gormrepository.NewWorkoutTypeRepository(db)
	trainingRepo := gormrepository.NewTrainingRepository(db)
	wsRepo := gormrepository.NewWorkoutSessionRepository(db)
	setRepo := gormrepository.NewWorkoutSetRepository(db)

	// handlers
	mgHandler := handler.NewMuscleGroupHandler(mgRepo)
	wtHandler := handler.NewWorkoutTypeHandler(wtRepo)
	trainingHandler := handler.NewTrainingHandler(trainingRepo)
	wsHandler := handler.NewWorkoutSessionHandler(wsRepo, trainingRepo, setRepo)

	// stub auth: inject a fixed authenticated user ID for all requests so that
	// endpoints requiring authorization (e.g., workout-session CRUD) succeed.
	const testUserID uint = 1
	noAuth := func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	}

	r := gin.New()
	mgHandler.RegisterRoutes(r, noAuth)
	wtHandler.RegisterRoutes(r, noAuth)
	trainingHandler.RegisterRoutes(r, noAuth)
	wsHandler.RegisterRoutes(r, noAuth)

	return r, db
}

func asJSON(t *testing.T, v any) *bytes.Buffer {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// -----------------------------------------------------------------------------
// Muscle‑group happy‑path CRUD
// -----------------------------------------------------------------------------

func TestMuscleGroupLifecycle(t *testing.T) {
	r, _ := testRouter(t)

	// CREATE
	var mgResp models.MuscleGroup
	{
		reqBody := asJSON(t, map[string]any{"name": "Chest"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/muscle-groups", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &mgResp))
		require.NotZero(t, mgResp.ID)
	}

	// LIST
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/muscle-groups", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var list []models.MuscleGroup
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
		require.Len(t, list, 1)
		require.Equal(t, mgResp.ID, list[0].ID)
	}

	// DELETE
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("/muscle-groups/%d", mgResp.ID), nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	}
}

// -----------------------------------------------------------------------------
// Workout‑type lifecycle (needs a parent muscle‑group)
// -----------------------------------------------------------------------------

func TestWorkoutTypeLifecycle(t *testing.T) {
	r, _ := testRouter(t)

	// prerequisite muscle‑group
	var mg models.MuscleGroup
	{
		reqBody := asJSON(t, map[string]any{"name": "Legs"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/muscle-groups", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &mg))
	}

	// CREATE workout‑type
	var wt models.WorkoutType
	{
		reqBody := asJSON(t, map[string]any{
			"name":            "Squat",
			"muscle_group_id": mg.ID,
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/workout-types", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wt))
		require.Equal(t, mg.ID, wt.MuscleGroupID)
	}

	// DELETE
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("/workout-types/%d", wt.ID), nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	}
}

// -----------------------------------------------------------------------------
// Workout-session with sets proves associations survive round-trip.
// -----------------------------------------------------------------------------

func TestWorkoutSessionWithSets(t *testing.T) {
	r, _ := testRouter(t)

	// create supporting muscle‑group & workout‑type
	var mg models.MuscleGroup
	{
		reqBody := asJSON(t, map[string]any{"name": "Back"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/muscle-groups", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &mg))
	}

	var wt models.WorkoutType
	{
		reqBody := asJSON(t, map[string]any{
			"name":            "Deadlift",
			"muscle_group_id": mg.ID,
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/workout-types", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wt))
	}

	var training models.Training
	{
		reqBody := asJSON(t, map[string]any{
			"title":      "Back day",
			"started_at": time.Now().UTC(),
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/trainings", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &training))
	}

	// create workout‑session
	var ws models.WorkoutSession
	{
		reqBody := asJSON(t, map[string]any{
			"training_id":     training.ID,
			"workout_type_id": wt.ID,
			"datetime":        time.Now().UTC(),
			"sets": []map[string]any{
				{"set_number": 1, "weight_kg": 80, "reps": 8},
			},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/workout-sessions", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ws))
	}

	// fetch the session and ensure typed set is present
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet,
			fmt.Sprintf("/workout-sessions/%d", ws.ID), nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var stored models.WorkoutSession
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stored))
		require.Len(t, stored.Sets, 1)
		require.Equal(t, 1, stored.Sets[0].SetNumber)
	}
}

func TestTrainingNestedCreateAndList(t *testing.T) {
	r, _ := testRouter(t)

	var mg models.MuscleGroup
	{
		reqBody := asJSON(t, map[string]any{"name": "Chest"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/muscle-groups", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &mg))
	}

	var wt models.WorkoutType
	{
		reqBody := asJSON(t, map[string]any{
			"name":            "Bench press",
			"description":     "Classic chest exercise",
			"default_metric":  "reps",
			"muscle_group_id": mg.ID,
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/workout-types", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wt))
	}

	var created models.Training
	{
		reqBody := asJSON(t, map[string]any{
			"title": "Chest & Triceps",
			"sessions": []map[string]any{
				{
					"workout_type_id": wt.ID,
					"sets": []map[string]any{
						{"set_number": 1, "weight_kg": 50, "reps": 12},
						{"set_number": 2, "weight_kg": 55, "reps": 10},
					},
				},
			},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/trainings", reqBody)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
		require.Len(t, created.Sessions, 1)
		require.Len(t, created.Sessions[0].Sets, 2)
	}

	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/trainings?limit=10", nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var list struct {
			Items []models.Training `json:"items"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
		require.Len(t, list.Items, 1)
		require.Equal(t, created.ID, list.Items[0].ID)
	}
}
