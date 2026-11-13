package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/VibeTeam/fitness-tracker-backend/workout/handler"
	"github.com/VibeTeam/fitness-tracker-backend/workout/models"
	"github.com/VibeTeam/fitness-tracker-backend/workout/repository/gormrepository"
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// testRouter creates an isolated Gin engine backed by an in‑mem SQLite DB and
// registers only the routes needed for these tests.
func testRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)

	// migrate the minimal set of tables we touch
	require.NoError(t, db.AutoMigrate(&models.MuscleGroup{}, &models.WorkoutType{},
		&models.Training{}, &models.WorkoutSession{}, &models.WorkoutSet{}))

	// repositories
	mgRepo := gormrepository.NewMuscleGroupRepository(db)
	wtRepo := gormrepository.NewWorkoutTypeRepository(db)
	trainingRepo := gormrepository.NewTrainingRepository(db)

	// handlers
	mgHandler := handler.NewMuscleGroupHandler(mgRepo)
	wtHandler := handler.NewWorkoutTypeHandler(wtRepo)
	trainingHandler := handler.NewTrainingHandler(trainingRepo)

	// stub auth: inject a fixed authenticated user ID for all requests so that
	// endpoints requiring authorization (e.g., workout-session CRUD) succeed.
	const testUserID uint = 1
	noAuth := func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	}

	r := gin.New()
	mgHandler.RegisterRoutes(r, noAuth)
	wtHandler.RegisterRoutes(r, noAuth)
	trainingHandler.RegisterRoutes(r, noAuth)

	return r, db
}

func asJSON(t *testing.T, v any) *bytes.Buffer {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}
