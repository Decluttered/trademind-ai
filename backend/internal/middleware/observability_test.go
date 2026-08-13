package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsGuardEnforcesTrustedProxyAndCIDRAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		wantStatus int
	}{
		{name: "public direct IP denied", remoteAddr: "203.0.113.20:9000", wantStatus: http.StatusForbidden},
		{name: "spoofed loopback from untrusted peer denied", remoteAddr: "203.0.113.20:9000", forwarded: "127.0.0.1", wantStatus: http.StatusForbidden},
		{name: "loopback allowed", remoteAddr: "127.0.0.1:9000", wantStatus: http.StatusOK},
		{name: "monitoring CIDR through trusted proxy allowed", remoteAddr: "127.0.0.1:9000", forwarded: "10.42.7.8", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			if err := r.SetTrustedProxies([]string{"127.0.0.1/32", "::1/128"}); err != nil {
				t.Fatal(err)
			}
			r.GET("/internal/metrics", MetricsGuard(true, []string{"127.0.0.1/32", "::1/128", "10.42.0.0/16"}), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/internal/metrics", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tt.wantStatus)
			}
		})
	}
}

func TestMetricsGuardFailsClosedForInvalidAllowlist(t *testing.T) {
	r := gin.New()
	r.GET("/internal/metrics", MetricsGuard(true, []string{"not-a-cidr"}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/internal/metrics", nil)
	req.RemoteAddr = "127.0.0.1:9000"
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusForbidden)
	}
}
