package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig defines configuration for rate limiting
type RateLimitConfig struct {
	// Window is the time window for rate limiting
	Window time.Duration
	// Limit is the maximum number of requests allowed in the window
	Limit int
	// Key prefix for Redis keys
	KeyPrefix string
}

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	redis  *redis.Client
	config RateLimitConfig
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(redisClient *redis.Client, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

// RateLimitMiddleware returns a Gin middleware that enforces rate limiting
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			c.Abort()
			return
		}

		userIDStr := fmt.Sprintf("%v", userID)
		allowed, remaining, resetTime, err := rl.IsAllowed(c.Request.Context(), userIDStr)
		if err != nil {
			// Log error but don't fail the request
			c.Header("X-RateLimit-Error", "rate limit check failed")
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.config.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":              "rate limit exceeded",
				"message":            fmt.Sprintf("You have exceeded the rate limit of %d requests per %v", rl.config.Limit, rl.config.Window),
				"rate_limit_remaining": remaining,
				"rate_limit_reset":     resetTime.Unix(),
				"retry_after":          int(time.Until(resetTime).Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IsAllowed checks if a request from the given user is allowed
// Returns: allowed, remaining requests, reset time, error
func (rl *RateLimiter) IsAllowed(ctx context.Context, userID string) (bool, int, time.Time, error) {
	now := time.Now()
	windowStart := now.Truncate(rl.config.Window)
	key := fmt.Sprintf("%s:%s:%d", rl.config.KeyPrefix, userID, windowStart.Unix())

	// Use Redis pipeline for atomic operations
	pipe := rl.redis.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.config.Window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, time.Time{}, err
	}

	count := int(incrCmd.Val())
	remaining := rl.config.Limit - count
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := windowStart.Add(rl.config.Window)
	allowed := count <= rl.config.Limit

	return allowed, remaining, resetTime, nil
}

// CheckOnly checks if a request would be allowed without incrementing the counter
func (rl *RateLimiter) CheckOnly(ctx context.Context, userID string) (bool, int, time.Time, error) {
	now := time.Now()
	windowStart := now.Truncate(rl.config.Window)
	key := fmt.Sprintf("%s:%s:%d", rl.config.KeyPrefix, userID, windowStart.Unix())

	// Only get the current count without incrementing
	count, err := rl.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		// No requests yet in this window
		return true, rl.config.Limit, windowStart.Add(rl.config.Window), nil
	}
	if err != nil {
		return false, 0, time.Time{}, err
	}

	remaining := rl.config.Limit - count
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := windowStart.Add(rl.config.Window)
	allowed := count < rl.config.Limit // Use < instead of <= since we haven't incremented yet

	return allowed, remaining, resetTime, nil
}

// IncrementUsage increments the usage counter for a user
func (rl *RateLimiter) IncrementUsage(ctx context.Context, userID string) error {
	now := time.Now()
	windowStart := now.Truncate(rl.config.Window)
	key := fmt.Sprintf("%s:%s:%d", rl.config.KeyPrefix, userID, windowStart.Unix())

	// Use Redis pipeline for atomic operations
	pipe := rl.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.config.Window)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetRemainingRequests returns the number of remaining requests for a user
func (rl *RateLimiter) GetRemainingRequests(ctx context.Context, userID string) (int, time.Time, error) {
	now := time.Now()
	windowStart := now.Truncate(rl.config.Window)
	key := fmt.Sprintf("%s:%s:%d", rl.config.KeyPrefix, userID, windowStart.Unix())

	count, err := rl.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		// No requests yet in this window
		return rl.config.Limit, windowStart.Add(rl.config.Window), nil
	}
	if err != nil {
		return 0, time.Time{}, err
	}

	remaining := rl.config.Limit - count
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := windowStart.Add(rl.config.Window)
	return remaining, resetTime, nil
}

// RecipeCreationRateLimiter creates a rate limiter for recipe creation (5 per hour for development)
func NewRecipeCreationRateLimiter(redisClient *redis.Client) *RateLimiter {
	return NewRateLimiter(redisClient, RateLimitConfig{
		Window:    time.Hour,
		Limit:     5,
		KeyPrefix: "rate_limit:recipe_creation",
	})
}

// RecipeModificationRateLimiter creates a rate limiter for recipe modifications (10 per recipe per hour)
func NewRecipeModificationRateLimiter(redisClient *redis.Client) *RateLimiter {
	return NewRateLimiter(redisClient, RateLimitConfig{
		Window:    time.Hour,
		Limit:     10,
		KeyPrefix: "rate_limit:recipe_modification",
	})
}

// PerRecipeRateLimitMiddleware creates a middleware for per-recipe rate limiting
func (rl *RateLimiter) PerRecipeRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			c.Abort()
			return
		}

		recipeID := c.Param("id")
		if recipeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
			c.Abort()
			return
		}

		userIDStr := fmt.Sprintf("%v", userID)
		key := fmt.Sprintf("%s:%s", userIDStr, recipeID)
		
		allowed, remaining, resetTime, err := rl.IsAllowed(c.Request.Context(), key)
		if err != nil {
			// Log error but don't fail the request
			c.Header("X-RateLimit-Error", "rate limit check failed")
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.config.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":              "rate limit exceeded",
				"message":            fmt.Sprintf("You have exceeded the rate limit of %d modifications per recipe per %v", rl.config.Limit, rl.config.Window),
				"rate_limit_remaining": remaining,
				"rate_limit_reset":     resetTime.Unix(),
				"retry_after":          int(time.Until(resetTime).Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}