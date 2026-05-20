package repository

import (
	"context"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
)

// WorkoutTypeRepository provides CRUD operations for WorkoutType entities.
type WorkoutTypeRepository interface {
	Create(ctx context.Context, wt *models.WorkoutType) error
	GetByID(ctx context.Context, id uint) (*models.WorkoutType, error)
	Update(ctx context.Context, wt *models.WorkoutType) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, limit, offset int) ([]*models.WorkoutType, error)
	ListByMuscleGroup(ctx context.Context, muscleGroupID uint, limit, offset int) ([]*models.WorkoutType, error)
	Search(ctx context.Context, query string, muscleGroupID *uint, limit, offset int) ([]*models.WorkoutType, error)
	Count(ctx context.Context) (int, error)
}
