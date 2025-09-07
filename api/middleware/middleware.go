package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// CORS returns a CORS middleware
func CORS() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	})
}

// Logger returns a logger middleware
func Logger() echo.MiddlewareFunc {
	return middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","id":"${id}","remote_ip":"${remote_ip}","host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}","status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}","bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	})
}

// Recover returns a recover middleware
func Recover() echo.MiddlewareFunc {
	return middleware.Recover()
}

// Timeout returns a timeout middleware
func Timeout(timeout time.Duration) echo.MiddlewareFunc {
	return middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: timeout,
	})
}

// RateLimiter returns a rate limiter middleware
func RateLimiter() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStore(100), // 100 requests per second
	})
}

// RequestID returns a request ID middleware
func RequestID() echo.MiddlewareFunc {
	return middleware.RequestID()
}

// HealthCheck is a simple health check middleware
func HealthCheck() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().URL.Path == "/health" {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"status":    "healthy",
					"timestamp": time.Now(),
					"service":   "banking-ledger",
				})
			}
			return next(c)
		}
	}
}
