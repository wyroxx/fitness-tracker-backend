package handler_test

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/stretchr/testify/require"
)

// SWIFT_APP_PATH points to the FitnessTracker checkout. Compile the real app's
// network layer unchanged, then execute it against isolated backend fixtures.
func TestActualSwiftAppContract(t *testing.T) {
	app := os.Getenv("SWIFT_APP_PATH")
	if app == "" || runtime.GOOS != "darwin" {
		t.Skip("set SWIFT_APP_PATH on macOS to test the actual Swift network layer")
	}
	compiler, err := exec.LookPath("swiftc")
	require.NoError(t, err)
	tmp := t.TempDir()
	binary := filepath.Join(tmp, "swift-contract")
	args := []string{"-parse-as-library", "-module-cache-path", filepath.Join(tmp, "cache"), "-o", binary, "testdata/SwiftContractProbe.swift"}
	for _, source := range []string{
		"Core/APIClient.swift", "Core/APIError.swift",
		"Features/Exercises/Domain/Exercise.swift", "Features/Exercises/Domain/ExerciseRepository.swift", "Features/Exercises/Data/ExerciseRepositoryImpl.swift",
		"Features/MuscleGroups/Domain/MuscleGroup.swift", "Features/MuscleGroups/Domain/MuscleGroupRepository.swift", "Features/MuscleGroups/Data/MuscleGroupRepositoryImpl.swift",
		"Features/Workouts/Domain/Workout.swift", "Features/Workouts/Domain/WorkoutExercise.swift", "Features/Workouts/Domain/WorkoutSet.swift", "Features/Workouts/Domain/WorkoutRepository.swift",
		"Features/Workouts/Data/CreateWorkoutRequest.swift", "Features/Workouts/Data/WorkoutRepositoryImpl.swift",
	} {
		args = append(args, filepath.Join(app, "FitnessTracker", source))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	output, err := exec.CommandContext(ctx, compiler, args...).CombinedOutput()
	require.NoError(t, err, string(output))
	r, db := testRouter(t)
	group := models.MuscleGroup{Name: "Chest"}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Create(&models.WorkoutType{Name: "Bench press", Description: "Barbell press", MuscleGroupID: group.ID}).Error)
	server := httptest.NewServer(r)
	defer server.Close()
	output, err = exec.CommandContext(ctx, binary, server.URL).CombinedOutput()
	require.NoError(t, err, string(output))
	t.Log(string(output))
	var count int64
	require.NoError(t, db.Model(&models.Training{}).Where("user_id = ?", 1).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
