package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/gorm"
)

// DashboardHandler handles dashboard-related requests
type DashboardHandler struct {
	db          *gorm.DB
	authService service.IAuthService
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(db *gorm.DB, authService service.IAuthService) *DashboardHandler {
	return &DashboardHandler{
		db:          db,
		authService: authService,
	}
}

// RegisterRoutes registers the dashboard routes
func (h *DashboardHandler) RegisterRoutes(router *gin.RouterGroup) {
	dashboard := router.Group("/dashboard")
	{
		dashboard.GET("/stats", h.GetStats)
		dashboard.GET("/favorites/recent", h.GetRecentFavorites)
	}
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	RecipesGenerated int    `json:"recipesGenerated"`
	Favorites        int    `json:"favorites"`
	ThisWeek         int    `json:"thisWeek"`
	PrimaryDiet      string `json:"primaryDiet"`
}

// GetStats returns dashboard statistics for the current user
func (h *DashboardHandler) GetStats(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// TODO: Implement actual statistics calculation
	// For now, return mock data to prevent frontend errors
	stats := DashboardStats{
		RecipesGenerated: 12,
		Favorites:        8,
		ThisWeek:         3,
		PrimaryDiet:      "Mediterranean",
	}

	c.JSON(http.StatusOK, stats)
}

// GetRecentFavorites returns recent favorite recipes for the current user
func (h *DashboardHandler) GetRecentFavorites(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// TODO: Implement actual favorites query
	// For now, return empty array to prevent frontend errors
	c.JSON(http.StatusOK, []interface{}{})
}
