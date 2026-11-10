// Package middleware provides HTTP middleware functions for the 3x-ui web panel,
// including domain validation utilities.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

// DomainValidatorMiddleware returns a Gin middleware that validates the request domain.
// Both the request host and the configured domain are compared in the form
// common.NormalizeHost gives (no port, case or trailing dot). Requests from
// unauthorized domains are rejected with HTTP 403 Forbidden status.
func DomainValidatorMiddleware(domain string) gin.HandlerFunc {
	domain = common.NormalizeHost(domain)
	return func(c *gin.Context) {
		if domain != "" && common.NormalizeHost(c.Request.Host) != domain {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
