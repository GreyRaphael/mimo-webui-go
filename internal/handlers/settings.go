package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mimo-webui/internal/db"
	"mimo-webui/internal/middleware"
)

// GetSettings returns all settings for the authenticated user.
func GetSettings(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetAuthUser(c)
		settings, err := db.GetSettings(c.Request.Context(), database, user.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
			return
		}
		c.JSON(http.StatusOK, settings)
	}
}

// UpdateSettings updates settings for the authenticated user.
func UpdateSettings(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetAuthUser(c)

		var body map[string]string
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		for name, value := range body {
			if err := db.SetSetting(c.Request.Context(), database, user.UserID, name, value); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting: " + name})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
