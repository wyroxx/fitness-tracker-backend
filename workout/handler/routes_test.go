package handler_test

import (
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestClientRoutes(t *testing.T) {
	r, _ := testRouter(t)
	var routes []string
	for _, route := range r.Routes() {
		routes = append(routes, route.Method+" "+route.Path)
	}
	require.ElementsMatch(t, []string{
		"GET /muscle-groups", "GET /muscle-groups/:id/workout-types",
		"POST /trainings", "GET /trainings", "DELETE /trainings/:id",
	}, routes)
	for _, path := range []string{"/muscle-groups", "/muscle-groups/1/workout-types", "/trainings"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 200, w.Code, w.Body.String())
	}
}
