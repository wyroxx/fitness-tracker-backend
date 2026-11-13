package handler

import "github.com/VibeTeam/fitness-tracker-backend/workout/models"

type workoutSetRequest struct {
	SetNumber int  `json:"set_number" binding:"gte=0" example:"1"`
	Weight    *int `json:"weight" binding:"omitempty,gte=0" example:"50"` // External load in kilograms; omit for bodyweight exercises.
	Reps      *int `json:"reps" binding:"omitempty,gte=0" example:"10"`
}

func workoutSetModels(reqs []workoutSetRequest) []models.WorkoutSet {
	sets := make([]models.WorkoutSet, len(reqs))
	for i, req := range reqs {
		setNumber := req.SetNumber
		if setNumber == 0 {
			setNumber = i + 1
		}
		sets[i] = models.WorkoutSet{
			SetNumber: setNumber,
			Weight:    req.Weight,
			Reps:      req.Reps,
		}
	}
	return sets
}
