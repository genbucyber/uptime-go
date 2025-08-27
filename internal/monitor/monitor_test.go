package monitor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"uptime-go/internal/incident"
	"uptime-go/internal/models"
	"uptime-go/internal/net"
	"uptime-go/internal/net/database"

	"github.com/stretchr/testify/assert"
)

func TestMonitorHandleWebsiteDown(t *testing.T) {
	testCases := []struct {
		name                 string
		monitor              models.Monitor
		checkResult          net.CheckResults
		err                  error
		setup                func(db *database.Database, monitor *models.Monitor)
		expectedResult       bool
		expectedIncidentType incident.Type
	}{
		{
			name:                 "new timeout incident",
			monitor:              models.Monitor{},
			checkResult:          net.CheckResults{},
			err:                  os.ErrDeadlineExceeded,
			expectedResult:       true,
			expectedIncidentType: incident.Timeout,
		},
		{
			name:        "incident already exists",
			monitor:     models.Monitor{},
			checkResult: net.CheckResults{},
			err:         errors.New(string(incident.UnexpectedStatusCode)),
			setup: func(db *database.Database, monitor *models.Monitor) {
				monitor.Incidents = []models.Incident{{Type: incident.UnexpectedStatusCode}}
				db.DB.Create(monitor)
			},
			expectedResult:       false,
			expectedIncidentType: incident.UnexpectedStatusCode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := database.InitializeTestDatabase()
			uptimeMonitor, _ := NewUptimeMonitor(db, nil)

			if tc.setup != nil {
				tc.setup(db, &tc.monitor)
			}

			result, incidentType := uptimeMonitor.handleWebsiteDown(&tc.monitor, &tc.checkResult, tc.err)
			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedIncidentType, incidentType)
		})
	}
}

func TestMonitorResolveIncidents(t *testing.T) {
	testCases := []struct {
		name           string
		monitor        models.Monitor
		incidentType   incident.Type
		setup          func(db *database.Database, monitor *models.Monitor)
		expectedResult bool
	}{
		{
			name:         "can be solve",
			monitor:      models.Monitor{},
			incidentType: incident.Timeout,
			setup: func(db *database.Database, monitor *models.Monitor) {
				monitor.Incidents = []models.Incident{{Type: incident.Timeout}}
				db.DB.Create(monitor)
			},
			expectedResult: true,
		},
		{
			name:         "nothing to solve",
			monitor:      models.Monitor{},
			incidentType: incident.Timeout,
			setup: func(db *database.Database, monitor *models.Monitor) {
				now := time.Now()
				monitor.Incidents = []models.Incident{{Type: incident.Timeout, SolvedAt: &now}}
				db.DB.Create(monitor)
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := database.InitializeTestDatabase()
			uptimeMonitor, _ := NewUptimeMonitor(db, nil)

			if tc.setup != nil {
				tc.setup(db, &tc.monitor)
			}

			result := uptimeMonitor.resolveIncidents(&tc.monitor, tc.incidentType)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestHandleSSL(t *testing.T) {
	expiredDuration := time.Hour * 24 * 30 // 30 days
	now := time.Now()
	expiredDate := now.Add(-time.Hour)
	expiringSoonDate := now.Add(time.Hour * 24 * 15)
	farFutureDate := now.Add(99999999 * time.Minute)

	testCases := []struct {
		name           string
		monitor        models.Monitor
		checkResult    net.CheckResults
		setup          func(db *database.Database, monitor *models.Monitor)
		expectedResult bool
		expectedDesc   string
	}{
		{
			name:        "create almost expired incident",
			monitor:     models.Monitor{CertificateExpiredBefore: &expiredDuration, URL: "https://example.com"},
			checkResult: net.CheckResults{SSLExpiredDate: &expiringSoonDate},
			setup: func(db *database.Database, monitor *models.Monitor) {
				db.DB.Create(monitor)
			},
			expectedResult: true,
			expectedDesc:   "Certificate almost expired",
		},
		{
			name: "solve incident",
			monitor: models.Monitor{
				CertificateExpiredBefore: &expiredDuration,
			},
			checkResult: net.CheckResults{SSLExpiredDate: &farFutureDate},
			setup: func(db *database.Database, monitor *models.Monitor) {
				monitor.Incidents = []models.Incident{{Type: incident.SSLExpired}}
				db.DB.Create(monitor)
			},
			expectedResult: true,
		},
		{
			name: "update almost expired to expired",
			monitor: models.Monitor{
				CertificateExpiredBefore: &expiredDuration,
			},
			checkResult: net.CheckResults{SSLExpiredDate: &expiredDate},
			setup: func(db *database.Database, monitor *models.Monitor) {
				monitor.Incidents = []models.Incident{{Description: "Certificate almost expired", Type: incident.SSLExpired}}
				db.DB.Create(monitor)
			},
			expectedResult: true,
			expectedDesc:   "Certificate expired",
		},
		{
			name:           "should do nothing if ssl expired date is nil",
			monitor:        models.Monitor{},
			checkResult:    net.CheckResults{SSLExpiredDate: nil},
			expectedResult: false,
		},
		{
			name:        "should create new expired incident if cert is expired and no incident exists",
			monitor:     models.Monitor{URL: "https://example.com"},
			checkResult: net.CheckResults{SSLExpiredDate: &expiredDate},
			setup: func(db *database.Database, monitor *models.Monitor) {
				db.DB.Create(monitor)
			},
			expectedResult: true,
			expectedDesc:   "Certificate expired",
		},
		{
			name:        "should do nothing if cert is expired and expired incident already exists",
			monitor:     models.Monitor{},
			checkResult: net.CheckResults{SSLExpiredDate: &expiredDate},
			setup: func(db *database.Database, monitor *models.Monitor) {
				monitor.Incidents = []models.Incident{{Description: "Certificate expired", Type: incident.SSLExpired}}
				db.DB.Create(monitor)
			},
			expectedResult: false,
		},
		{
			name: "should do nothing if cert is expiring soon and incident already exists",
			monitor: models.Monitor{
				CertificateExpiredBefore: &expiredDuration,
			},
			checkResult: net.CheckResults{SSLExpiredDate: &expiringSoonDate},
			setup: func(db *database.Database, monitor *models.Monitor) {
				monitor.Incidents = []models.Incident{{Description: "Certificate almost expired", Type: incident.SSLExpired}}
				db.DB.Create(monitor)
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := database.InitializeTestDatabase()
			uptimeMonitor, _ := NewUptimeMonitor(db, nil)

			if tc.setup != nil {
				tc.setup(db, &tc.monitor)
			}

			result := uptimeMonitor.handleSSL(&tc.monitor, &tc.checkResult)
			assert.Equal(t, tc.expectedResult, result)

			if tc.expectedDesc != "" {
				lastIncident := uptimeMonitor.db.GetLastIncident(tc.monitor.URL, incident.SSLExpired)
				assert.Equal(t, tc.expectedDesc, lastIncident.Description)
			}
		})
	}
}

func TestCheckWebsite(t *testing.T) {
	t.Run("website is up", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		db, _ := database.InitializeTestDatabase()
		uptimeMonitor, _ := NewUptimeMonitor(db, nil)
		monitor := &models.Monitor{
			URL:                   server.URL,
			Interval:              1 * time.Minute,
			ResponseTimeThreshold: 5 * time.Second,
		}
		db.DB.Create(monitor)

		uptimeMonitor.checkWebsite(monitor)

		db.DB.First(monitor)
		assert.True(t, *monitor.IsUp)
		assert.Equal(t, http.StatusOK, *monitor.StatusCode)
	})

	t.Run("website is down", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		db, _ := database.InitializeTestDatabase()
		uptimeMonitor, _ := NewUptimeMonitor(db, nil)
		monitor := &models.Monitor{
			URL:                   server.URL,
			Interval:              1 * time.Minute,
			ResponseTimeThreshold: 5 * time.Second,
		}
		db.DB.Create(monitor)

		uptimeMonitor.checkWebsite(monitor)

		db.DB.First(monitor)
		assert.False(t, *monitor.IsUp)
		assert.Equal(t, http.StatusInternalServerError, *monitor.StatusCode)

		lastIncident := uptimeMonitor.db.GetLastIncident(monitor.URL, incident.UnexpectedStatusCode)
		assert.True(t, lastIncident.IsExists())
		assert.Equal(t, "Unexpected status code: 500", lastIncident.Description)
	})
}
