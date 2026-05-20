package repository

import (
	"context"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
)

// WorkoutSetRepository provides CRUD and replacement operations for typed sets.
type WorkoutSetRepository interface {
	Create(ctx context.Context, set *models.WorkoutSet) error
	ListBySession(ctx context.Context, sessionID uint) ([]*models.WorkoutSet, error)
	ReplaceForSession(ctx context.Context, sessionID uint, sets []models.WorkoutSet) error
	Delete(ctx context.Context, id uint) error
}
