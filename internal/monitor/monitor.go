package monitor

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"uptime-go/internal/helper"
	"uptime-go/internal/models"
	"uptime-go/internal/net"
	"uptime-go/internal/net/database"
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		m.Stop()
	}()

	fmt.Println("Starting uptime monitoring for", len(m.configs), "websites")
	fmt.Println("Press Ctrl+C to stop")

	// Start a goroutine for each website to monitor
	for _, cfg := range m.configs {
		if !cfg.Enabled {
			log.Printf("%s - skipped because disabled\n", cfg.URL)
			continue
		}

		m.wg.Add(1)
		go m.monitorWebsite(cfg)
	}

	m.wg.Wait()
}

func (m *UptimeMonitor) Stop() {
	close(m.stopChan)
	m.wg.Wait()
	fmt.Println("Monitoring stopped")
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
		log.Printf("Error checking %s: %v", monitor.URL, err)
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
	if result.IsUp {
		m.resolveIncidents(monitor, models.UnexpectedStatusCode)
		m.resolveIncidents(monitor, models.Timeout)
		m.handleSSL(monitor, result)

		if monitor.LastUp == nil {
			now := time.Now()
			monitor.LastUp = &now
		}
	} else {
		statusText = "DOWN"
		monitor.LastUp = nil
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

	log.Printf("%s - %s - Response time: %v - Status: %d",
		monitor.URL, statusText, result.ResponseTime, result.StatusCode)

	if err := m.db.Upsert(monitor); err != nil {
		log.Printf("Failed to save result to database: %v", err)
	}
}

func (m *UptimeMonitor) handleWebsiteDown(monitor *models.Monitor, result *net.CheckResults, err error) (bool, models.IncidentType) {
	// return true if new incident created; else false and incident type

	var description string
	incidentType := models.UnexpectedStatusCode

	if err != nil {
		description = err.Error()
		if os.IsTimeout(err) {
			incidentType = models.Timeout
			result.ResponseTime = monitor.ResponseTimeThreshold
		}
	} else {
		description = fmt.Sprintf("Unexpected status code: %d", result.StatusCode)
	}

	lastIncident := m.db.GetLastIncident(monitor.URL, incidentType)
	if !lastIncident.CreatedAt.IsZero() {
		return false, incidentType // Incident already recorded
	}

	incident := &models.Incident{
		ID:          helper.GenerateRandomID(),
		MonitorID:   monitor.ID,
		Type:        incidentType,
		Description: description,
	}

	m.db.DB.Create(incident)
	log.Printf(
		"%s - New Incident detected! - Type: %s",
		monitor.URL, incident.Type.String(),
	)

	return true, incidentType
}

func (m *UptimeMonitor) resolveIncidents(monitor *models.Monitor, incidentType models.IncidentType) bool {
	// return true if incident solved; else false

	now := time.Now()
	lastIncident := m.db.GetLastIncident(monitor.URL, incidentType)
	if !lastIncident.CreatedAt.IsZero() {
		lastIncident.SolvedAt = &now
		m.db.Upsert(lastIncident)
		log.Printf("%s - Incident Solved - Type: %s - Downtime: %s\n", monitor.URL, incidentType.String(), time.Since(lastIncident.CreatedAt))
		return true
	}

	return false
}

func (m *UptimeMonitor) handleSSL(monitor *models.Monitor, result *net.CheckResults) bool {
	now := time.Now()
	lastSSLIncident := m.db.GetLastIncident(monitor.URL, models.SSLExpired)

	isSSLExpiringSoon := result.SSLExpiredDate != nil &&
		monitor.CertificateExpiredBefore != nil &&
		time.Until(*result.SSLExpiredDate) <= *monitor.CertificateExpiredBefore

	if isSSLExpiringSoon && lastSSLIncident.CreatedAt.IsZero() {
		log.Printf("%s - Please update SSL Certificate - [%s]", monitor.URL, result.SSLExpiredDate)
		m.db.DB.Create(&models.Incident{
			MonitorID:   monitor.ID,
			Type:        models.SSLExpired,
			Description: fmt.Sprintf("SSL will be expired on %s", result.SSLExpiredDate),
		})
		return true
	} else if !isSSLExpiringSoon && !lastSSLIncident.CreatedAt.IsZero() {
		lastSSLIncident.SolvedAt = &now
		m.db.Upsert(lastSSLIncident)
		log.Printf("%s - SSL Updated\n", monitor.URL)
		return true
	}

	return false
}
