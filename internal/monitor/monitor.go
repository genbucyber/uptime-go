package monitor

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"uptime-go/internal/net"
	"uptime-go/internal/net/config"
	"uptime-go/internal/net/database"
)

// UptimeMonitor represents a service that periodically checks website uptime
type UptimeMonitor struct {
	configs  []*config.NetworkConfig
	db       *database.Database
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewUptimeMonitor(configs []*config.NetworkConfig) (*UptimeMonitor, error) {
	// Initialize database
	db, err := database.InitializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &UptimeMonitor{
		configs:  configs,
		db:       &database.Database{DB: db},
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

func (m *UptimeMonitor) monitorWebsite(cfg *config.NetworkConfig) {
	defer m.wg.Done()

	ticker := time.NewTicker(cfg.RefreshInterval)
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

func (m *UptimeMonitor) checkWebsite(cfg *config.NetworkConfig) {
	netConfig := &net.NetworkConfig{
		URL:             cfg.URL,
		Timeout:         cfg.Timeout,
		FollowRedirects: cfg.FollowRedirects,
		SkipSSL:         cfg.SkipSSL,
	}

	result, err := netConfig.CheckWebsite()
	if err != nil {
		log.Printf("Error checking %s: %v", cfg.URL, err)
		// Create a failed check result
		result = &config.CheckResults{
			URL:          cfg.URL,
			LastCheck:    time.Now(),
			ResponseTime: 0,
			IsUp:         false,
			StatusCode:   0,
			ErrorMessage: err.Error(),
		}
	}

	// Log the result
	statusText := "DOWN"
	if result.IsUp {
		statusText = "UP"
	}
	log.Printf("%s - %s - Response time: %v - Status: %d",
		cfg.URL, statusText, result.ResponseTime.Round(time.Millisecond), result.StatusCode)

	// Save result to database
	if err := m.db.SaveResults(result); err != nil {
		log.Printf("Failed to save result to database: %v", err)
	}
}
