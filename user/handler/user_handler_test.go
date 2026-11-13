package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/VibeTeam/fitness-tracker-backend/user/handler"
	"github.com/VibeTeam/fitness-tracker-backend/user/models"
)

/* ----------- in‑memory UserRepository implementation ------------------------ */

type userMemRepo struct {
	mu    sync.Mutex
	store map[uint]*models.User
	next  uint
}

// GetByEmail implements repository.UserRepository.
func (r *userMemRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	panic("unimplemented")
}

func newUserRepo() *userMemRepo {
	return &userMemRepo{store: make(map[uint]*models.User), next: 1}
}

func (r *userMemRepo) Create(_ context.Context, u *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u.ID = r.next
	r.next++
	cp := *u
	r.store[u.ID] = &cp
	return nil
}

func (r *userMemRepo) GetByID(_ context.Context, id uint) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.store[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, errors.New("not found")
}

func (r *userMemRepo) List(_ context.Context, _, _ int) ([]*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*models.User, 0, len(r.store))
	for _, u := range r.store {
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func (r *userMemRepo) Update(_ context.Context, u *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *u
	r.store[u.ID] = &cp
	return nil
}

func (r *userMemRepo) Delete(_ context.Context, id uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

func (r *userMemRepo) Count(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.store), nil
}

/* --------------------------------------------------------------------------- */

func setupRouter() (*gin.Engine, *userMemRepo) {
	repo := newUserRepo()
	h := handler.New(repo)

	r := gin.New()
	h.RegisterRoutes(r, func(c *gin.Context) {})
	return r, repo
}

func TestCreateUser_Success(t *testing.T) {
	r, _ := setupRouter()

	body := map[string]string{
		"name":     "Bob",
		"email":    "bob@e.com",
		"password": "pwd",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(raw))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestRemovedUserRoute_NotFound(t *testing.T) {
	r, _ := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/99", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserRoutes(t *testing.T) {
	r, _ := setupRouter()
	var routes []string
	for _, route := range r.Routes() {
		routes = append(routes, route.Method+" "+route.Path)
	}
	require.ElementsMatch(t, []string{"POST /users", "GET /users/me"}, routes)
}
