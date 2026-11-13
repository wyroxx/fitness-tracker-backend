package handler

import (
	"time"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
)

// These DTOs match both mobile clients. Training ID supports Flutter deletion.
type workoutResponse struct {
	ID        uint                      `json:"id"`
	Title     string                    `json:"title"`
	StartedAt string                    `json:"started_at"`
	Exercises []workoutExerciseResponse `json:"exercises"`
}

type workoutExerciseResponse struct {
	ID   uint                 `json:"id"`
	Name string               `json:"name"`
	Sets []workoutSetResponse `json:"sets"`
}

type workoutSetResponse struct {
	SetNumber int  `json:"set_number"`
	Weight    *int `json:"weight,omitempty"`
	Reps      *int `json:"reps,omitempty"`
}

type exerciseResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func workoutFromTraining(training *models.Training) workoutResponse {
	result := workoutResponse{
		ID:        training.ID,
		Title:     training.Title,
		StartedAt: training.StartedAt.Format(time.RFC3339Nano),
		Exercises: make([]workoutExerciseResponse, len(training.Sessions)),
	}
	for i, session := range training.Sessions {
		exercise := workoutExerciseResponse{ID: session.WorkoutTypeID, Sets: make([]workoutSetResponse, len(session.Sets))}
		if session.WorkoutType != nil {
			exercise.Name = session.WorkoutType.Name
		}
		for j, set := range session.Sets {
			exercise.Sets[j] = workoutSetResponse{SetNumber: set.SetNumber, Weight: set.Weight, Reps: set.Reps}
		}
		result.Exercises[i] = exercise
	}
	return result
}

func exerciseFromModel(exercise *models.WorkoutType) exerciseResponse {
	return exerciseResponse{ID: exercise.ID, Name: exercise.Name, Description: exercise.Description}
}

func exercisesFromModels(exercises []*models.WorkoutType) []exerciseResponse {
	result := make([]exerciseResponse, len(exercises))
	for i, exercise := range exercises {
		result[i] = exerciseFromModel(exercise)
	}
	return result
}
