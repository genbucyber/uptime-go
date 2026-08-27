package api

import (
	"io"
	"math"
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

type MonitorResponse struct {
	URL          string `json:"url"`
	Enabled		 bool	`json:"enabled"`
	IsUp         bool   `json:"is_up"`
	StatusCode   int    `json:"status_code"`
	ResponseTime int64  `json:"response_time"` // milliseconds

	CertificateExpiration    *string    `json:"certificate_expiration,omitempty"`     // RFC3339 format
	CertificateExpiredDate   *time.Time `json:"certificate_expired_date,omitempty"`
	CertificateRemainingDays *int       `json:"certificate_remaining_days,omitempty"`

	LastUp    *time.Time `json:"last_up,omitempty"`
	LastDown  *time.Time `json:"last_down,omitempty"`
	LastCheck time.Time  `json:"last_check"`

	DailyStats []MonitorDailyUptimeStats `json:"stats,omitempty"`
	Histories  []models.MonitorHistory   `json:"histories,omitempty"`
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

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated. Systemd will restart the service to apply changes."})
}

func (s *Server) ReloadConfigHanlder(c *gin.Context){
	if err := configuration.Load(s.configPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to load configuration",
		})
		log.Err(err).Msg("Error loading configuration during reload")
		return
	}

	syncedConfigs, err := configuration.SyncMonitorsWithDB(s.db, configuration.Config.Monitor)
	if err != nil{
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to sync configuration with database",
		})
		log.Err(err).Msg("Error syncing configuration with database during reload")
		return
	}

	if s.monitor != nil {
		s.monitor.Reload(syncedConfigs)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration reloaded successfully",
	})
}

func (s *Server) GetMonitoringReport(c *gin.Context) {
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

	var dailyStatsMap map[string][]models.DailyStat
	var ninetyDaysAgo, today time.Time  

	if params.WithStat {
		now := time.Now().UTC()
		today = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		ninetyDaysAgo = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).AddDate(0, 0, -89)

		monitorURLs := make([]string, 0, len(monitors)) 

		for _, m := range monitors {
			monitorURLs = append(monitorURLs, m.URL)
		}

		dailyStatsMap, err = s.db.GetDailyStats(monitorURLs, ninetyDaysAgo, today)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to retrieve monitor histories for stats calculation", 
			})
			return
		}
	}

	var monitorResponse []MonitorResponse

	for _, m := range monitors {
		var dailyStats []MonitorDailyUptimeStats

		if params.WithStat && dailyStatsMap != nil {
			dailyStats = calculateUptimeStats(dailyStatsMap[m.URL], ninetyDaysAgo, today)
		}

		monitorResponse = append(monitorResponse, buildMonitorResponse(m, dailyStats))
	}

	c.JSON(http.StatusOK, monitorResponse)
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

	c.JSON(http.StatusOK, buildMonitorResponse(*monitor, nil))
}

func calculateUptimeStats(stats []models.DailyStat, from, to time.Time) []MonitorDailyUptimeStats {
	dailyStatsMap := make(map[string]models.DailyStat)

	for _, stat := range stats {
		dailyStatsMap[stat.Date] = stat
	}

	var result []MonitorDailyUptimeStats
	for d := from; !d.After(to); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		stat, exits := dailyStatsMap[dateStr]
		uptimePercentage := 0.0
		totalChecks := 0
		
		if exits && stat.TotalChecks > 0 {
			totalChecks = stat.TotalChecks
			uptimePercentage = (float64(stat.UpChecks) / float64(stat.TotalChecks)) * 100
			uptimePercentage = math.Round(uptimePercentage * 100) / 100
		}

		result = append(result, MonitorDailyUptimeStats{
			Date:             dateStr,
			UptimePercentage: uptimePercentage,
			TotalChecks:      totalChecks,
		})
	}

	if len(stats) == 0 {
		return nil
	}

	return result
}

func buildMonitorResponse(m models.Monitor, dailyStats []MonitorDailyUptimeStats) MonitorResponse {
	statusCodeVal := 0
	if m.StatusCode != nil {
		statusCodeVal = *m.StatusCode
	}

	responseTimeVal := int64(0)
	if m.ResponseTime != nil{
		responseTimeVal = *m.ResponseTime
	}

	isUpVa := false
	if m.IsUp != nil {
		isUpVa = *m.IsUp
	}

	var certExpiration *string                                   
    var certRemainingDays *int

	if m.CertificateExpiredDate != nil {
		formattedStr := m.CertificateExpiredDate.Format(time.RFC3339)           
        certExpiration = &formattedStr                                          
        days := int(time.Until(*m.CertificateExpiredDate).Hours() / 24)         
        certRemainingDays = &days
	}

	return MonitorResponse{
		URL: 						m.URL,
		Enabled: 					m.Enabled,       
		IsUp: 						isUpVa,  
		StatusCode: 				statusCodeVal,
		ResponseTime: 				responseTimeVal,
		CertificateExpiration: 		certExpiration,    
		CertificateExpiredDate: 	m.CertificateExpiredDate,
		CertificateRemainingDays: 	certRemainingDays,
		LastUp:						m.LastUp,
		LastDown: 					m.LastDown,
		LastCheck: 					m.UpdatedAt,
		DailyStats: 				dailyStats,
		Histories: 					m.Histories,
	}
}
