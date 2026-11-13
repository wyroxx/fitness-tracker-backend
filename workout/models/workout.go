package models

import "time"

// MuscleGroup represents a primary muscle group targeted by a workout.
type MuscleGroup struct {
	ID   uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"type:text;not null"`
}

// WorkoutType represents a particular kind of workout (e.g., Bench Press) and the muscle group it trains.
type WorkoutType struct {
	ID            uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name          string `json:"name" gorm:"type:text;not null"`
	Description   string `json:"description" gorm:"type:text"`
	MuscleGroupID uint   `json:"muscle_group_id" gorm:"not null;index"`

	// Associations
	MuscleGroup *MuscleGroup `json:"muscle_group,omitempty" gorm:"foreignKey:MuscleGroupID"`
}

// Training is a user-facing workout card containing one or more exercise sessions.
type Training struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Title     string    `json:"title" gorm:"type:text;not null"`
	StartedAt time.Time `json:"started_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Sessions []WorkoutSession `json:"sessions" gorm:"foreignKey:TrainingID;constraint:OnDelete:CASCADE"`
}

// WorkoutSession is a log entry for a completed workout instance performed by a user.
type WorkoutSession struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TrainingID    uint      `json:"training_id" gorm:"index"`
	WorkoutTypeID uint      `json:"workout_type_id" gorm:"not null;index"`
	UserID        uint      `json:"user_id" gorm:"not null;index"`
	OrderIndex    int       `json:"order_index" gorm:"not null;default:0"`
	Datetime      time.Time `json:"datetime" gorm:"not null"`

	// Associations
	Training    *Training    `json:"training,omitempty" gorm:"foreignKey:TrainingID"`
	WorkoutType *WorkoutType `json:"workout_type,omitempty" gorm:"foreignKey:WorkoutTypeID"`
	Sets        []WorkoutSet `json:"sets" gorm:"foreignKey:WorkoutSessionID;constraint:OnDelete:CASCADE"`
}

// WorkoutSet stores strength-training performance for one set. Weight is in kilograms.
type WorkoutSet struct {
	ID               uint `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkoutSessionID uint `json:"workout_session_id" gorm:"not null;index"`
	SetNumber        int  `json:"set_number" gorm:"not null"`
	Weight           *int `json:"weight,omitempty"`
	Reps             *int `json:"reps,omitempty"`
}
