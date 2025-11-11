package api

import (
	"io"
	"net/http"
	"uptime-go/internal/configuration"

	"github.com/gin-gonic/gin"
)

type ReportQueryParams struct {
	URL   string `form:"url"`
	Limit int    `form:"limit"`
}

func (s *Server) UpdateConfigHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Failed to read request body", "error": err.Error()})
		return
	}

	if err := configuration.UpdateConfig(s.configPath, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update configuration", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully. Please restart the application to apply changes."})
}

func (s *Server) GetMonitoringReport(c *gin.Context) {
	var queryParams ReportQueryParams

	if err := c.ShouldBindQuery(&queryParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid query parameters", "error": err.Error()})
		return
	}

	if queryParams.Limit == 0 {
		queryParams.Limit = 1000
	}

	if queryParams.URL == "" {
		monitors, err := s.db.GetAllMonitors()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve monitors", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, monitors)
		return
	}

	monitor, err := s.db.GetMonitorWithHistories(queryParams.URL, queryParams.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve monitor details", "error": err.Error()})
		return
	}

	if monitor == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, monitor)
}

func (s *Server) HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "uptime-go",
	})
}
