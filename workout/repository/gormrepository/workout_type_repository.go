package gormrepository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

// gormWorkoutTypeRepository implements repository.WorkoutTypeRepository using GORM.
type gormWorkoutTypeRepository struct {
	db *gorm.DB
}

// NewWorkoutTypeRepository returns a GORM-backed WorkoutType repository.
func NewWorkoutTypeRepository(db *gorm.DB) repository.WorkoutTypeRepository {
	return &gormWorkoutTypeRepository{db: db}
}

func (r *gormWorkoutTypeRepository) Create(ctx context.Context, wt *models.WorkoutType) error {
	return r.db.WithContext(ctx).Create(wt).Error
}

func (r *gormWorkoutTypeRepository) GetByID(ctx context.Context, id uint) (*models.WorkoutType, error) {
	var wt models.WorkoutType
	err := r.db.WithContext(ctx).Preload("MuscleGroup").First(&wt, id).Error
	if err != nil {
		return nil, err
	}
	return &wt, nil
}

func (r *gormWorkoutTypeRepository) Update(ctx context.Context, wt *models.WorkoutType) error {
	return r.db.WithContext(ctx).Save(wt).Error
}

func (r *gormWorkoutTypeRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.WorkoutType{}, id).Error
}

func (r *gormWorkoutTypeRepository) List(ctx context.Context, limit, offset int) ([]*models.WorkoutType, error) {
	var wts []*models.WorkoutType
	err := r.db.WithContext(ctx).
		Preload("MuscleGroup").
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&wts).Error
	return wts, err
}

func (r *gormWorkoutTypeRepository) ListByMuscleGroup(ctx context.Context, muscleGroupID uint, limit, offset int) ([]*models.WorkoutType, error) {
	var wts []*models.WorkoutType
	err := r.db.WithContext(ctx).
		Where("muscle_group_id = ?", muscleGroupID).
		Preload("MuscleGroup").
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&wts).Error
	return wts, err
}

func (r *gormWorkoutTypeRepository) Search(ctx context.Context, query string, muscleGroupID *uint, limit, offset int) ([]*models.WorkoutType, error) {
	var wts []*models.WorkoutType
	db := r.db.WithContext(ctx).Preload("MuscleGroup").Order("name ASC")
	if muscleGroupID != nil {
		db = db.Where("muscle_group_id = ?", *muscleGroupID)
	}
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		db = db.Where("lower(name) LIKE ?", "%"+strings.ToLower(trimmed)+"%")
	}
	err := db.Limit(limit).Offset(offset).Find(&wts).Error
	return wts, err
}

func (r *gormWorkoutTypeRepository) Count(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.WorkoutType{}).Count(&count).Error
	return int(count), err
}
