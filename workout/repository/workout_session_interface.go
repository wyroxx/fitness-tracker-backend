package repository

import (
	"context"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
)

// WorkoutSessionRepository provides operations for workout sessions and their sets.
type WorkoutSessionRepository interface {
	Create(ctx context.Context, session *models.WorkoutSession) error
	GetByID(ctx context.Context, id uint) (*models.WorkoutSession, error)
	Update(ctx context.Context, session *models.WorkoutSession) error
	Delete(ctx context.Context, id uint) error

	// ListByUser lists all sessions for a specific user with pagination.
	ListByUser(ctx context.Context, userID uint, limit, offset int) ([]*models.WorkoutSession, error)
	ListByTraining(ctx context.Context, trainingID uint) ([]*models.WorkoutSession, error)
	CountByUser(ctx context.Context, userID uint) (int, error)
}
