package monitor

import (
	"fmt"
	"os"
	"sync"
	"time"

	"uptime-go/internal/helper"
	"uptime-go/internal/incident"
	"uptime-go/internal/models"
	"uptime-go/internal/net"
	"uptime-go/internal/net/database"

	"github.com/rs/zerolog/log"
)

// UptimeMonitor represents a service that periodically checks website uptime
type UptimeMonitor struct {
	configs  []*models.Monitor
	db       *database.Database
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewUptimeMonitor(db *database.Database, configs []*models.Monitor) (*UptimeMonitor, error) {
	return &UptimeMonitor{
		configs:  configs,
		db:       db,
		stopChan: make(chan struct{}),
	}, nil
}

func (m *UptimeMonitor) Start() {
	log.Info().Msgf("Starting uptime monitoring for %d websites", len(m.configs))

	// Start a goroutine for each website to monitor
	for _, cfg := range m.configs {
		if !cfg.Enabled {
			log.Info().Msgf("%s - skipped because disabled", cfg.URL)
			continue
		}

		m.wg.Add(1)
		go m.monitorWebsite(cfg)
	}
}

// Shutdown gracefully stops all monitoring goroutines.
func (m *UptimeMonitor) Shutdown() {
	log.Info().Msg("Shutting down uptime monitoring...")
	close(m.stopChan)
	m.wg.Wait()
	log.Info().Msg("Uptime monitoring stopped")
}

func (m *UptimeMonitor) monitorWebsite(cfg *models.Monitor) {
	defer m.wg.Done()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// Perform initial check immediately
	m.checkWebsite(cfg)

	for {
		select {
		case <-ticker.C:
			m.checkWebsite(cfg)
		case <-m.stopChan:
			return
		}
	}
}

func (m *UptimeMonitor) checkWebsite(monitor *models.Monitor) {
	nc := &net.NetworkConfig{
		URL:             monitor.URL,
		RefreshInterval: monitor.Interval,
		Timeout:         monitor.ResponseTimeThreshold,
		SkipSSL:         !monitor.CertificateMonitoring,
	}

	result, err := nc.CheckWebsite()
	if err != nil {
		log.Error().Err(err).Msgf("Error checking %s", monitor.URL)
		// Create a failed check result
		result = &net.CheckResults{
			URL:          monitor.URL,
			LastCheck:    time.Now(),
			ResponseTime: 0,
			IsUp:         false,
			StatusCode:   0,
			ErrorMessage: err.Error(),
		}
	}

	statusText := "UP"
	now := time.Now()
	if result.IsUp {
		if monitor.LastUp == nil {
			monitor.LastUp = &now
		}

		m.resolveIncidents(monitor, incident.UnexpectedStatusCode)
		m.resolveIncidents(monitor, incident.Timeout)
		if monitor.CertificateMonitoring {
			m.handleSSL(monitor, result)
		}
	} else {
		statusText = "DOWN"
		m.handleWebsiteDown(monitor, result, err)
	}

	responseTime := result.ResponseTime.Milliseconds()
	monitor.UpdatedAt = result.LastCheck
	monitor.IsUp = &result.IsUp
	monitor.StatusCode = &result.StatusCode
	monitor.ResponseTime = &responseTime
	monitor.CertificateExpiredDate = result.SSLExpiredDate
	monitor.Histories = []models.MonitorHistory{
		{
			IsUp:         result.IsUp,
			StatusCode:   result.StatusCode,
			ResponseTime: responseTime,
		},
	}

	log.Info().Msgf("%s - %s - Response time: %v - Status: %d",
		monitor.URL, statusText, result.ResponseTime, result.StatusCode)

	if err := m.db.Upsert(monitor); err != nil {
		log.Error().Err(err).Msg("Failed to save result to database")
	}
}

func (m *UptimeMonitor) handleWebsiteDown(monitor *models.Monitor, result *net.CheckResults, err error) (bool, incident.Type) {
	// return true if new incident created; else false and incident type

	var description string
	incidentType := incident.UnexpectedStatusCode

	if err != nil {
		description = err.Error()
		if os.IsTimeout(err) {
			incidentType = incident.Timeout
			result.ResponseTime = monitor.ResponseTimeThreshold
		}
	} else {
		description = fmt.Sprintf("Unexpected status code: %d", result.StatusCode)
	}

	lastIncident := m.db.GetLastIncident(monitor.URL, incidentType)
	if lastIncident.IsExists() {
		return false, incidentType // Incident already recorded
	}

	inc := &models.Incident{
		ID:          helper.GenerateRandomID(),
		MonitorID:   monitor.ID,
		Type:        incidentType,
		Description: description,
		Monitor:     *monitor,
	}

	if id, err := net.NotifyIncident(inc, incident.HIGH, nil); err == nil {
		inc.IncidentID = id
	}

	now := time.Now()
	monitor.LastDown = &now
	m.db.DB.Create(inc)
	log.Warn().Msgf(
		"%s - New Incident detected! - Type: %s",
		monitor.URL, inc.Type,
	)

	return true, incidentType
}

func (m *UptimeMonitor) resolveIncidents(monitor *models.Monitor, incidentType incident.Type) bool {
	// return true if incident solved; else false

	now := time.Now()
	lastIncident := m.db.GetLastIncident(monitor.URL, incidentType)
	if lastIncident.IsExists() {
		lastIncident.SolvedAt = &now
		monitor.LastUp = &now
		m.db.Upsert(lastIncident)
		log.Info().Msgf("%s - Incident Solved - Type: %s - Downtime: %s", monitor.URL, incidentType, time.Since(lastIncident.CreatedAt))
		net.UpdateIncidentStatus(lastIncident, incident.Resolved)

		return true
	}

	return false
}

func (m *UptimeMonitor) handleSSL(monitor *models.Monitor, result *net.CheckResults) bool {
	// If SSL expiry date is not available, do nothing.
	if result.SSLExpiredDate == nil {
		return false
	}

	now := time.Now()
	lastIncident := m.db.GetLastIncident(monitor.URL, incident.SSLExpired)

	attr := map[string]any{
		"expired_date": result.SSLExpiredDate,
	}

	// Certificate is expired.
	if time.Until(*result.SSLExpiredDate) <= 0 {

		// If the existing incident is for "almost expired", update it to "expired".
		if lastIncident.IsExists() && lastIncident.Description == "Certificate almost expired" {
			log.Warn().Msgf("%s - Certificate expired - [%s]", monitor.URL, result.SSLExpiredDate)
			lastIncident.Description = "Certificate expired"
			if id, err := net.NotifyIncident(lastIncident, incident.HIGH, attr); err == nil {
				lastIncident.IncidentID = id
			}
			m.db.Upsert(lastIncident)
			return true
		}

		// If there is no incident, create a new "expired" incident.
		if lastIncident.IsNotExists() {
			log.Warn().Msgf("%s - Certificate expired - [%s]", monitor.URL, result.SSLExpiredDate)
			inc := &models.Incident{
				ID:          helper.GenerateRandomID(),
				MonitorID:   monitor.ID,
				Type:        incident.SSLExpired,
				Description: "Certificate expired",
				Monitor:     *monitor,
			}
			if id, err := net.NotifyIncident(inc, incident.HIGH, attr); err == nil {
				inc.IncidentID = id
			}
			m.db.DB.Create(inc)
			return true
		}

		return false // Incident for expired already exists.
	}

	isSSLExpiringSoon := monitor.CertificateExpiredBefore != nil &&
		time.Until(*result.SSLExpiredDate) <= *monitor.CertificateExpiredBefore

	// Certificate is expiring soon.
	if isSSLExpiringSoon {

		// If no incident exists, create a new "almost expired" incident.
		if lastIncident.IsNotExists() {
			log.Warn().Msgf("%s - Please update SSL Certificate - [%s]", monitor.URL, result.SSLExpiredDate)
			inc := &models.Incident{
				ID:          helper.GenerateRandomID(),
				MonitorID:   monitor.ID,
				Type:        incident.SSLExpired,
				Description: "Certificate almost expired",
				Monitor:     *monitor,
			}
			if id, err := net.NotifyIncident(inc, incident.INFO, attr); err == nil {
				inc.IncidentID = id
			}
			m.db.DB.Create(inc)
			return true
		}

		return false // Incident for expiring soon already exists.
	}

	if lastIncident.IsExists() {
		// Manual resolve
		// if lastIncident.IncidentID != 0 {
		// 	net.UpdateIncidentStatus(lastIncident, incident.Resolved)
		// }

		lastIncident.SolvedAt = &now
		m.db.Upsert(lastIncident)
		log.Info().Msgf("%s - SSL Updated", monitor.URL)
		return true
	}

	return false
}
