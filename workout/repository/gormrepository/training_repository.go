package gormrepository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type gormTrainingRepository struct {
	db *gorm.DB
}

// NewTrainingRepository returns a GORM-backed Training repository.
func NewTrainingRepository(db *gorm.DB) repository.TrainingRepository {
	return &gormTrainingRepository{db: db}
}

func (r *gormTrainingRepository) Create(ctx context.Context, training *models.Training) error {
	return r.db.WithContext(ctx).Create(training).Error
}

func (r *gormTrainingRepository) GetByID(ctx context.Context, id uint) (*models.Training, error) {
	var training models.Training
	err := r.withGraph(ctx).First(&training, id).Error
	if err != nil {
		return nil, err
	}
	return &training, nil
}

func (r *gormTrainingRepository) Update(ctx context.Context, training *models.Training) error {
	return r.db.WithContext(ctx).Save(training).Error
}

func (r *gormTrainingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Training{}, id).Error
}

func (r *gormTrainingRepository) ListByUser(ctx context.Context, userID uint, limit int, cursor *time.Time) ([]*models.Training, error) {
	var trainings []*models.Training
	q := r.withGraph(ctx).
		Where("user_id = ?", userID).
		Order("started_at DESC").
		Order("id DESC").
		Limit(limit)
	if cursor != nil {
		q = q.Where("started_at < ?", *cursor)
	}
	err := q.Find(&trainings).Error
	return trainings, err
}

func (r *gormTrainingRepository) CountByUser(ctx context.Context, userID uint) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Training{}).Where("user_id = ?", userID).Count(&count).Error
	return int(count), err
}

func (r *gormTrainingRepository) CountByUserSince(ctx context.Context, userID uint, since time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Training{}).
		Where("user_id = ? AND started_at >= ?", userID, since).
		Count(&count).Error
	return int(count), err
}

func (r *gormTrainingRepository) withGraph(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Preload("Sessions", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC").Order("id ASC")
		}).
		Preload("Sessions.WorkoutType").
		Preload("Sessions.WorkoutType.MuscleGroup").
		Preload("Sessions.Sets", func(db *gorm.DB) *gorm.DB {
			return db.Order("set_number ASC").Order("id ASC")
		})
}
