package repository

import (
	"context"
	"time"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
)

// TrainingRepository provides operations for the user-facing workout aggregate.
type TrainingRepository interface {
	Create(ctx context.Context, training *models.Training) error
	GetByID(ctx context.Context, id uint) (*models.Training, error)
	Update(ctx context.Context, training *models.Training) error
	Delete(ctx context.Context, id uint) error
	ListByUser(ctx context.Context, userID uint, limit int, cursor *time.Time) ([]*models.Training, error)
	CountByUser(ctx context.Context, userID uint) (int, error)
	CountByUserSince(ctx context.Context, userID uint, since time.Time) (int, error)
}
