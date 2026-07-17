package api

import (
	"io"
	"net/http"
	"time"
	"uptime-go/internal/configuration"
	"uptime-go/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type ReportQueryParams struct {
	WithStat bool `form:"with_stat"`
}

type HistoryReportQueryParams struct {
	URL   string `form:"url"`
	Limit int    `form:"limit"`
	From  string `form:"from"`
	To    string `form:"to"`
}

type MonitorDailyUptimeStats struct {
	Date             string  `json:"date"`
	UptimePercentage float64 `json:"uptime_percentage"`
	TotalChecks      int     `json:"total_checks"`
}

type MonitorWithDailyUptimeStats struct {
	models.Monitor
	DailyStats []MonitorDailyUptimeStats `json:"stats,omitempty"`
}

func (s *Server) HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "uptime-go",
	})
}

func (s *Server) UpdateConfigHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Failed to read request body", "error": err.Error()})
		return
	}

	if err := configuration.UpdateConfig(s.configPath, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update configuration"})
		log.Err(err).Msg("Error updating configuration")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully. Please restart the application to apply changes."})
}

func (s *Server) GetMonitoringReport(c *gin.Context) {
	url := c.Query("url")
	if url != ""{
		monitor, err := s.db.GetMonitorHistories(url, 100, time.Time{}, time.Time{})
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to retrieve monitor history",
				"error": err.Error(),
			})
			return
		}

		if monitor == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message" : "Record not found for the given url",
			})
			return
		}
		c.JSON(http.StatusOK, monitor)
		return
	}

	var params ReportQueryParams

	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid query parameters",
		})
		return
	}

	var urls []string
	for _, m := range configuration.Config.Monitor {
		urls = append(urls, m.URL)
	}

	monitors, err := s.db.GetMonitors(urls)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to retrieve monitors",
			"error":   err.Error(),
		})
		return
	}

	if params.WithStat {
		var monitorStats []MonitorWithDailyUptimeStats

		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		ninetyDaysAgo := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).AddDate(0, 0, -89)

		monitorURLs := make([]string, 0, len(monitors))
		for _, m := range monitors {
			monitorURLs = append(monitorURLs, m.URL)
		}

		histories, err := s.db.GetHistoryWithMonitor(monitorURLs, ninetyDaysAgo, today)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to retrieve monitor histories for stats calculation",
			})
			return
		}

		for _, monitor := range monitors {
			histories, _ := histories[monitor.URL]
			dailyStats := calculateUptimeStats(histories, ninetyDaysAgo, today)

			monitorStats = append(monitorStats, MonitorWithDailyUptimeStats{
				Monitor:    monitor,
				DailyStats: dailyStats,
			})
		}
		c.JSON(http.StatusOK, monitorStats)
		return
	}

	c.JSON(http.StatusOK, monitors)
}

func (s *Server) GetMonitoringHistoryReport(c *gin.Context) {
	var params HistoryReportQueryParams

	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid query parameters",
		})
		return
	}

	if params.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "URL parameter is required",
		})
		return
	}

	if params.Limit == 0 {
		params.Limit = 1000
	}

	var from, to time.Time

	if params.From != "" && params.To != "" {
		var err error

		from, err = time.Parse("2006-01-02", params.From)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'from' date format. Please use YYYY-MM-DD."})
			return
		}

		to, err = time.Parse("2006-01-02", params.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'to' date format. Please use YYYY-MM-DD."})
			return
		}

		to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	monitor, err := s.db.GetMonitorHistories(params.URL, params.Limit, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to retrieve monitor history",
		})
		return
	}

	if monitor == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Record not found for the given URL",
		})
		return
	}

	c.JSON(http.StatusOK, monitor)
}

func calculateUptimeStats(histories []models.MonitorHistory, from, to time.Time) []MonitorDailyUptimeStats {
	dailyStatsMap := make(map[string]struct {
		UpCount    int
		TotalCount int
	})

	for d := from; !d.After(to); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		dailyStatsMap[dateStr] = struct {
			UpCount    int
			TotalCount int
		}{UpCount: 0, TotalCount: 0}
	}

	for _, history := range histories {
		dateStr := history.CreatedAt.Format("2006-01-02")
		stats := dailyStatsMap[dateStr]
		stats.TotalCount++
		if history.IsUp {
			stats.UpCount++
		}
		dailyStatsMap[dateStr] = stats
	}

	var result []MonitorDailyUptimeStats
	for d := from; !d.After(to); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		stats := dailyStatsMap[dateStr]
		uptimePercentage := 0.0
		if stats.TotalCount > 0 {
			uptimePercentage = (float64(stats.UpCount) / float64(stats.TotalCount)) * 100
		}

		result = append(result, MonitorDailyUptimeStats{
			Date:             dateStr,
			UptimePercentage: uptimePercentage,
			TotalChecks:      stats.TotalCount,
		})
	}

	// If no histories at all, return nil to omit "stats" field in JSON response
	if len(histories) == 0 {
		return nil
	}

	return result
}
