package gormrepository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository"
)

// newTestDB creates an in‑memory SQLite DB and migrates all workout models.
func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("opening DB: %v", err)
	}

	if err := db.AutoMigrate(
		&models.MuscleGroup{},
		&models.WorkoutType{},
		&models.Training{},
		&models.WorkoutSession{},
		&models.WorkoutSet{},
	); err != nil {
		t.Fatalf("migrating schema: %v", err)
	}

	return db
}

/*
CRUD path for MuscleGroup
*/
func TestMuscleGroupCRUD(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	var mgRepo repository.MuscleGroupRepository = NewMuscleGroupRepository(db)

	// CREATE
	orig := &models.MuscleGroup{Name: "Chest"}
	if err := mgRepo.Create(ctx, orig); err != nil {
		t.Fatalf("create: %v", err)
	}
	if orig.ID == 0 {
		t.Fatalf("create: expected auto‑ID")
	}

	// READ (single)
	got, err := mgRepo.GetByID(ctx, orig.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Name != "Chest" {
		t.Fatalf("get by id: wrong name %q", got.Name)
	}

	// UPDATE
	got.Name = "Upper Chest"
	if err := mgRepo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	// LIST / COUNT
	list, err := mgRepo.List(ctx, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: want 1, got %d (err=%v)", len(list), err)
	}
	cnt, _ := mgRepo.Count(ctx)
	if cnt != 1 {
		t.Fatalf("count: want 1, got %d", cnt)
	}

	// DELETE
	if err := mgRepo.Delete(ctx, got.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if c, _ := mgRepo.Count(ctx); c != 0 {
		t.Fatalf("delete: record still present")
	}
}

/*
Basic path for WorkoutType (requires a MuscleGroup FK)
*/
func TestWorkoutTypeCreateAndGet(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	mgRepo := NewMuscleGroupRepository(db)
	wtRepo := NewWorkoutTypeRepository(db)

	// prerequisite muscle group
	mg := &models.MuscleGroup{Name: "Back"}
	if err := mgRepo.Create(ctx, mg); err != nil {
		t.Fatalf("create mg: %v", err)
	}

	// CREATE workout type
	wt := &models.WorkoutType{
		Name:          "Deadlift",
		MuscleGroupID: mg.ID,
	}
	if err := wtRepo.Create(ctx, wt); err != nil {
		t.Fatalf("create wt: %v", err)
	}

	// READ (preloaded muscle group)
	got, err := wtRepo.GetByID(ctx, wt.ID)
	if err != nil {
		t.Fatalf("get wt: %v", err)
	}
	if got.Name != "Deadlift" || got.MuscleGroup.ID != mg.ID {
		t.Fatalf("unexpected workout type %+v", got)
	}
}

/*
Smoke test for WorkoutSession & WorkoutSet to ensure associations persist.
*/
func TestWorkoutSessionWithSets(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	mgRepo := NewMuscleGroupRepository(db)
	wtRepo := NewWorkoutTypeRepository(db)
	wsRepo := NewWorkoutSessionRepository(db)
	setRepo := NewWorkoutSetRepository(db)
	trainingRepo := NewTrainingRepository(db)

	// setup prerequisite records
	mg := &models.MuscleGroup{Name: "Legs"}
	if err := mgRepo.Create(ctx, mg); err != nil {
		t.Fatalf("create mg: %v", err)
	}
	wt := &models.WorkoutType{Name: "Squat", MuscleGroupID: mg.ID}
	if err := wtRepo.Create(ctx, wt); err != nil {
		t.Fatalf("create wt: %v", err)
	}

	training := &models.Training{
		UserID:    42,
		Title:     "Leg day",
		StartedAt: time.Now(),
	}
	if err := trainingRepo.Create(ctx, training); err != nil {
		t.Fatalf("create training: %v", err)
	}

	// CREATE session
	session := &models.WorkoutSession{
		TrainingID:    training.ID,
		WorkoutTypeID: wt.ID,
		UserID:        42,
		Datetime:      time.Now(),
	}
	if err := wsRepo.Create(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	reps := 12
	weight := 80.0
	set := &models.WorkoutSet{
		WorkoutSessionID: session.ID,
		SetNumber:        1,
		WeightKg:         &weight,
		Reps:             &reps,
	}
	if err := setRepo.Create(ctx, set); err != nil {
		t.Fatalf("create set: %v", err)
	}

	// Round-trip: fetch session preloaded with typed sets.
	stored, err := wsRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(stored.Sets) != 1 || *stored.Sets[0].Reps != 12 {
		t.Fatalf("sets not persisted: %+v", stored.Sets)
	}
}

func TestTrainingNestedCreate(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	mgRepo := NewMuscleGroupRepository(db)
	wtRepo := NewWorkoutTypeRepository(db)
	trainingRepo := NewTrainingRepository(db)

	mg := &models.MuscleGroup{Name: "Chest"}
	if err := mgRepo.Create(ctx, mg); err != nil {
		t.Fatalf("create mg: %v", err)
	}
	wt := &models.WorkoutType{Name: "Bench press", MuscleGroupID: mg.ID}
	if err := wtRepo.Create(ctx, wt); err != nil {
		t.Fatalf("create wt: %v", err)
	}

	reps := 10
	weight := 50.0
	training := &models.Training{
		UserID:    7,
		Title:     "Chest & Triceps",
		StartedAt: time.Now(),
		Sessions: []models.WorkoutSession{
			{
				UserID:        7,
				WorkoutTypeID: wt.ID,
				Datetime:      time.Now(),
				Sets: []models.WorkoutSet{
					{SetNumber: 1, WeightKg: &weight, Reps: &reps},
				},
			},
		},
	}
	if err := trainingRepo.Create(ctx, training); err != nil {
		t.Fatalf("create training: %v", err)
	}

	stored, err := trainingRepo.GetByID(ctx, training.ID)
	if err != nil {
		t.Fatalf("get training: %v", err)
	}
	if len(stored.Sessions) != 1 || len(stored.Sessions[0].Sets) != 1 {
		t.Fatalf("nested associations not persisted: %+v", stored)
	}
}
