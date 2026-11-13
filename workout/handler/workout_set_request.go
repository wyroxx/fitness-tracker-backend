package handler

import "github.com/VibeTeam/fitness-tracker-backend/workout/models"

type workoutSetRequest struct {
	SetNumber       int      `json:"set_number"`
	WeightKg        *float64 `json:"weight_kg"`
	Reps            *int     `json:"reps"`
	DurationSeconds *int     `json:"duration_seconds"`
	DistanceMeters  *float64 `json:"distance_meters"`
	Notes           string   `json:"notes"`
}

func workoutSetModels(reqs []workoutSetRequest) []models.WorkoutSet {
	sets := make([]models.WorkoutSet, len(reqs))
	for i, req := range reqs {
		setNumber := req.SetNumber
		if setNumber == 0 {
			setNumber = i + 1
		}
		sets[i] = models.WorkoutSet{
			SetNumber:       setNumber,
			WeightKg:        req.WeightKg,
			Reps:            req.Reps,
			DurationSeconds: req.DurationSeconds,
			DistanceMeters:  req.DistanceMeters,
			Notes:           req.Notes,
		}
	}
	return sets
}
