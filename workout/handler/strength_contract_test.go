package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/stretchr/testify/require"
)

func swiftTrainingPayload(exercises string) string {
	workout := fmt.Sprintf(`{"title":"Chest day","started_at":"2026-10-25T12:00:00Z","exercises":%s}`, exercises)
	return fmt.Sprintf(`{"title":"Chest day","started_at":"2026-10-25T12:00:00Z","sessions":[%s]}`, workout)
}

func TestStrengthTrainingContract(t *testing.T) {
	r, db := testRouter(t)
	group := models.MuscleGroup{Name: "Chest"}
	require.NoError(t, db.Create(&group).Error)
	exercise := models.WorkoutType{Name: "Bench press", Description: "Barbell press", MuscleGroupID: group.ID}
	require.NoError(t, db.Create(&exercise).Error)
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, status, w.Code, w.Body.String())
		return w.Body.Bytes()
	}
	require.JSONEq(t, `[]`, string(request(http.MethodGet, "/trainings", "", http.StatusOK)))
	catalog := request(http.MethodGet, fmt.Sprintf("/muscle-groups/%d/workout-types", group.ID), "", http.StatusOK)
	require.JSONEq(t, fmt.Sprintf(`[{"id":%d,"name":"Bench press","description":"Barbell press"}]`, exercise.ID), string(catalog))
	exercises := fmt.Sprintf(`[{"id":%d,"name":"Bench press","sets":[{"set_number":1,"weight":52,"reps":10},{"set_number":2,"weight":0,"reps":8},{"set_number":3,"reps":12},{"set_number":4}]},{"id":%d,"name":"Bench press","sets":[]}]`, exercise.ID, exercise.ID)
	expected := fmt.Sprintf(`{"title":"Chest day","started_at":"2026-10-25T12:00:00Z","exercises":%s}`, exercises)
	created := request(http.MethodPost, "/trainings", swiftTrainingPayload(exercises), http.StatusCreated)
	require.JSONEq(t, expected, string(created))
	require.JSONEq(t, "["+expected+"]", string(request(http.MethodGet, "/trainings", "", http.StatusOK)))
	var training models.Training
	require.NoError(t, db.Preload("Sessions.Sets").First(&training).Error)
	require.Len(t, training.Sessions, 2)
	require.Len(t, training.Sessions[0].Sets, 4)
	require.Equal(t, 52, *training.Sessions[0].Sets[0].Weight)

	// The unpaginated Swift repository must see all history, scoped to its user.
	for i := 0; i < 25; i++ {
		require.NoError(t, db.Create(&models.Training{UserID: 1, Title: "History", StartedAt: time.Now()}).Error)
	}
	require.NoError(t, db.Create(&models.Training{UserID: 2, Title: "Private", StartedAt: time.Now()}).Error)
	var all []map[string]any
	require.NoError(t, json.Unmarshal(request(http.MethodGet, "/trainings", "", http.StatusOK), &all))
	require.Len(t, all, 26)
	for _, item := range all {
		require.Len(t, item, 3)
		require.NotEqual(t, "Private", item["title"])
		require.NotNil(t, item["exercises"])
	}
}

func TestRejectInvalidSwiftPayloads(t *testing.T) {
	for _, tc := range []struct{ name, set, message string }{
		{"old weight", `{"set_number":1,"weight_kg":50}`, "unknown field"},
		{"duration", `{"set_number":1,"duration_seconds":30}`, "unknown field"},
		{"distance", `{"set_number":1,"distance_meters":100}`, "unknown field"},
		{"notes", `{"set_number":1,"notes":"unused"}`, "unknown field"},
		{"fractional weight", `{"set_number":1,"weight":52.5}`, "cannot unmarshal"},
		{"fractional reps", `{"set_number":1,"reps":1.5}`, "cannot unmarshal"},
		{"negative weight", `{"set_number":1,"weight":-1}`, "gte"},
		{"negative reps", `{"set_number":1,"reps":-1}`, "gte"},
		{"negative set number", `{"set_number":-1}`, "gte"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, db := testRouter(t)
			payload := swiftTrainingPayload(fmt.Sprintf(`[{"id":1,"name":"Press","sets":[%s]}]`, tc.set))
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/trainings", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), tc.message)
			var count int64
			require.NoError(t, db.Model(&models.Training{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestSwiftTrainingEnvelopeValidation(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"old backend shape", `{"title":"Chest day","started_at":"2026-10-25T12:00:00Z","sessions":[{"workout_type_id":1,"sets":[]}]}`},
		{"no sessions", `{"title":"Chest day","started_at":"2026-10-25T12:00:00Z","sessions":[]}`},
		{"missing ID", swiftTrainingPayload(`[{"name":"Press","sets":[]}]`)},
		{"unknown ID", swiftTrainingPayload(`[{"id":123,"name":"Missing","sets":[]}]`)},
		{"missing sets", swiftTrainingPayload(`[{"id":1,"name":"Press"}]`)},
		{"mismatched title", strings.Replace(swiftTrainingPayload(`[]`), `"title":"Chest day"`, `"title":"Other"`, 1)},
		{"multiple objects", swiftTrainingPayload(`[]`) + ` {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, db := testRouter(t)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/trainings", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var count int64
			require.NoError(t, db.Model(&models.Training{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}
