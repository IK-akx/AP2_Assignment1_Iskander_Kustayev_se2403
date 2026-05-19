package middleware

import (
	"net/http"
	"strings"
	"time"

	"order/internal/cache"

	"github.com/gin-gonic/gin"
)

// RateLimiterMiddleware creates a Gin middleware for rate limiting
func RateLimiterMiddleware(limiter *cache.RateLimiter, getIdentifier func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get identifier (IP or User ID)
		identifier := getIdentifier(c)
		if identifier == "" {
			c.Next()
			return
		}

		// Check if allowed
		allowed, remaining, resetTime, err := limiter.Allow(identifier)

		if err != nil {
			// On Redis error, log and allow the request (fail open)
			c.Error(err)
			c.Next()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", formatInt(limiter.GetLimit()))
		c.Header("X-RateLimit-Remaining", formatInt(remaining))
		c.Header("X-RateLimit-Reset", formatResetTime(resetTime))

		if !allowed {
			// Rate limit exceeded
			retryAfter := int(resetTime.Sub(time.Now()).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}
			c.Header("Retry-After", formatInt(retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"message":     formatRateLimitMessage(limiter.GetLimit(), limiter.GetWindow()),
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetIdentifierByIP extracts identifier from client IP
func GetIdentifierByIP(c *gin.Context) string {
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = c.Request.RemoteAddr
	}

	// Remove port if present
	if idx := strings.Index(clientIP, ":"); idx != -1 {
		clientIP = clientIP[:idx]
	}

	return "ip:" + clientIP
}

// GetIdentifierByUserID extracts identifier from user ID in request
func GetIdentifierByUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok && userIDStr != "" {
			return "user:" + userIDStr
		}
	}

	return GetIdentifierByIP(c)
}

// GetIdentifierByBoth uses both user ID and IP
func GetIdentifierByBoth(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok && userIDStr != "" {
			return "user:" + userIDStr
		}
	}

	return GetIdentifierByIP(c)
}

// Helper functions
func formatInt(n int) string {
	if n < 0 {
		return "0"
	}
	// Simple conversion for numbers up to 9999
	if n < 10 {
		return string(rune('0' + n))
	}
	return strings.TrimSpace(strings.Join([]string{formatIntHelper(n)}, ""))
}

func formatIntHelper(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}
	return result
}

func formatResetTime(t time.Time) string {
	return t.Format(time.RFC1123)
}

func formatRateLimitMessage(limit int, window time.Duration) string {
	minutes := int(window.Minutes())
	if minutes > 0 {
		return formatRateLimitMessageWithMinutes(limit, minutes)
	}
	seconds := int(window.Seconds())
	return formatRateLimitMessageWithSeconds(limit, seconds)
}

func formatRateLimitMessageWithMinutes(limit, minutes int) string {
	return "You have exceeded the rate limit of " + formatIntHelper(limit) + " requests per " + formatIntHelper(minutes) + " minute(s)"
}

func formatRateLimitMessageWithSeconds(limit, seconds int) string {
	return "You have exceeded the rate limit of " + formatIntHelper(limit) + " requests per " + formatIntHelper(seconds) + " second(s)"
}
