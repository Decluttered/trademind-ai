package api

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/trademind-ai/trademind/backend/internal/config"
)

func TestRegisterProductionRoutesMountsOwnedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterProductionRoutes(router, &Deps{Config: &config.Config{}})

	want := map[string]bool{
		"GET /api/v1/p10/status":                            false,
		"GET /api/v1/p10/credentials":                       false,
		"POST /api/v1/p10/credentials/offline":              false,
		"POST /api/v1/p10/inventory-read/runs":              false,
		"POST /api/v1/p10/inventory-read/runs/:runId/rerun": false,
		"PUT /api/v1/p10/controls/kill-switches":            false,
		"PUT /api/v1/p10/controls/allowlist":                false,
		"PUT /api/v1/p10/gray":                              false,
		"POST /api/v1/p10/gray/pause":                       false,
		"POST /api/v1/p10/gray/stop":                        false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("production capability route not registered: %s", route)
		}
	}
}
