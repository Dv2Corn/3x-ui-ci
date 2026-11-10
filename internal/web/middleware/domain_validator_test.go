package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Host names are case-insensitive and may end with the root's dot, so a domain
// saved as "Panel.Example.com" must not lock everyone out with 403.
func TestDomainValidatorMiddleware(t *testing.T) {
	cases := []struct {
		name, domain, host string
		want               int
	}{
		{"exact", "panel.example.com", "panel.example.com", http.StatusOK},
		{"exact with port", "panel.example.com", "panel.example.com:2053", http.StatusOK},
		{"configured upper case", "Panel.Example.com", "panel.example.com:2053", http.StatusOK},
		{"configured trailing dot", "panel.example.com.", "panel.example.com", http.StatusOK},
		{"configured with spaces", " panel.example.com ", "panel.example.com", http.StatusOK},
		{"request upper case and dot", "panel.example.com", "PANEL.example.com.:2053", http.StatusOK},
		{"configured IDN", "bücher.example", "xn--bcher-kva.example", http.StatusOK},
		{"IPv6 without port", "2001:db8::1", "[2001:db8::1]", http.StatusOK},
		{"blank domain checks nothing", "   ", "other.example.com", http.StatusOK},
		{"other host", "panel.example.com", "other.example.com", http.StatusForbidden},
		{"subdomain of configured", "example.com", "panel.example.com", http.StatusForbidden},
	}
	gin.SetMode(gin.TestMode)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(DomainValidatorMiddleware(tc.domain))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tc.host
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("domain %q, Host %q: status %d, want %d", tc.domain, tc.host, rec.Code, tc.want)
			}
		})
	}
}
