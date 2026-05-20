package gormrepository

import (
	"context"

	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

type gormWorkoutSetRepository struct {
	db *gorm.DB
}

// NewWorkoutSetRepository returns a GORM-backed WorkoutSet repository.
func NewWorkoutSetRepository(db *gorm.DB) repository.WorkoutSetRepository {
	return &gormWorkoutSetRepository{db: db}
}

func (r *gormWorkoutSetRepository) Create(ctx context.Context, set *models.WorkoutSet) error {
	return r.db.WithContext(ctx).Create(set).Error
}

func (r *gormWorkoutSetRepository) ListBySession(ctx context.Context, sessionID uint) ([]*models.WorkoutSet, error) {
	var sets []*models.WorkoutSet
	err := r.db.WithContext(ctx).
		Where("workout_session_id = ?", sessionID).
		Order("set_number ASC").
		Order("id ASC").
		Find(&sets).Error
	return sets, err
}

func (r *gormWorkoutSetRepository) ReplaceForSession(ctx context.Context, sessionID uint, sets []models.WorkoutSet) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workout_session_id = ?", sessionID).Delete(&models.WorkoutSet{}).Error; err != nil {
			return err
		}
		for i := range sets {
			sets[i].ID = 0
			sets[i].WorkoutSessionID = sessionID
			if sets[i].SetNumber == 0 {
				sets[i].SetNumber = i + 1
			}
		}
		if len(sets) == 0 {
			return nil
		}
		return tx.Create(&sets).Error
	})
}

func (r *gormWorkoutSetRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.WorkoutSet{}, id).Error
}
