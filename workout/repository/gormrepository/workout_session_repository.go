package gormrepository

import (
	"context"

	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

// gormWorkoutSessionRepository implements repository.WorkoutSessionRepository using GORM.
type gormWorkoutSessionRepository struct {
	db *gorm.DB
}

// NewWorkoutSessionRepository returns a GORM-backed WorkoutSession repository.
func NewWorkoutSessionRepository(db *gorm.DB) repository.WorkoutSessionRepository {
	return &gormWorkoutSessionRepository{db: db}
}

func (r *gormWorkoutSessionRepository) Create(ctx context.Context, session *models.WorkoutSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *gormWorkoutSessionRepository) GetByID(ctx context.Context, id uint) (*models.WorkoutSession, error) {
	var session models.WorkoutSession
	err := r.db.WithContext(ctx).
		Preload("WorkoutType").
		Preload("WorkoutType.MuscleGroup").
		Preload("Sets", func(db *gorm.DB) *gorm.DB {
			return db.Order("set_number ASC").Order("id ASC")
		}).
		First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *gormWorkoutSessionRepository) Update(ctx context.Context, session *models.WorkoutSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *gormWorkoutSessionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.WorkoutSession{}, id).Error
}

func (r *gormWorkoutSessionRepository) ListByUser(ctx context.Context, userID uint, limit, offset int) ([]*models.WorkoutSession, error) {
	var sessions []*models.WorkoutSession
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("datetime DESC").
		Limit(limit).
		Offset(offset).
		Preload("WorkoutType").
		Preload("WorkoutType.MuscleGroup").
		Preload("Sets", func(db *gorm.DB) *gorm.DB {
			return db.Order("set_number ASC").Order("id ASC")
		}).
		Find(&sessions).Error
	return sessions, err
}

func (r *gormWorkoutSessionRepository) ListByTraining(ctx context.Context, trainingID uint) ([]*models.WorkoutSession, error) {
	var sessions []*models.WorkoutSession
	err := r.db.WithContext(ctx).
		Where("training_id = ?", trainingID).
		Order("order_index ASC").
		Order("id ASC").
		Preload("WorkoutType").
		Preload("WorkoutType.MuscleGroup").
		Preload("Sets", func(db *gorm.DB) *gorm.DB {
			return db.Order("set_number ASC").Order("id ASC")
		}).
		Find(&sessions).Error
	return sessions, err
}

func (r *gormWorkoutSessionRepository) CountByUser(ctx context.Context, userID uint) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.WorkoutSession{}).Where("user_id = ?", userID).Count(&count).Error
	return int(count), err
}
