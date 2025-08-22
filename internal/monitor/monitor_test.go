package monitor

import (
	"errors"
	"os"
	"testing"
	"time"
	"uptime-go/internal/incident"
	"uptime-go/internal/models"
	"uptime-go/internal/net"
	"uptime-go/internal/net/database"

	"github.com/stretchr/testify/assert"
)

func TestMonitorHandleWebsiteDown_NewTimeoutIncident(t *testing.T) {
	// can create new incident

	db, _ := database.InitializeTestDatabase()
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)
	monitor := models.Monitor{}
	checkResult := net.CheckResults{}

	result, incidentType := uptimeMonitor.handleWebsiteDown(&monitor, &checkResult, os.ErrDeadlineExceeded)

	assert.True(t, result)
	assert.Equal(t, incidentType, incident.Timeout)
}

func TestMonitorHandleWebsiteDown_IncidentAlreadyExists(t *testing.T) {
	// if incident already exists on database; don't create again

	db, _ := database.InitializeTestDatabase()
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)
	checkResult := net.CheckResults{}
	monitor := models.Monitor{
		Incidents: []models.Incident{
			{Type: incident.UnexpectedStatusCode},
		},
	}

	db.DB.Create(&monitor)

	result, incidentType := uptimeMonitor.handleWebsiteDown(
		&monitor, &checkResult,
		errors.New(string(incident.UnexpectedStatusCode)),
	)

	assert.False(t, result)
	assert.Equal(t, incidentType, incident.UnexpectedStatusCode)
}

func TestMonitorResolveIncidents_CanBeSolve(t *testing.T) {
	db, _ := database.InitializeTestDatabase()
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)
	monitor := models.Monitor{
		Incidents: []models.Incident{
			{Type: incident.Timeout},
		},
	}

	db.DB.Create(&monitor)

	result := uptimeMonitor.resolveIncidents(&monitor, incident.Timeout)
	assert.True(t, result)
}

func TestMonitorResolveIncidents_NothingToSolve(t *testing.T) {
	db, _ := database.InitializeTestDatabase()
	monitor := models.Monitor{}
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)

	db.DB.Create(&monitor)

	result := uptimeMonitor.resolveIncidents(&monitor, incident.SSLExpired)
	assert.False(t, result)
}

func TestMonitorResolveIncidents_NothingToSolve2(t *testing.T) {
	db, _ := database.InitializeTestDatabase()
	now := time.Now()
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)
	monitor := models.Monitor{
		Incidents: []models.Incident{
			{Type: incident.Timeout, SolvedAt: &now},
		},
	}

	db.DB.Create(&monitor)

	result := uptimeMonitor.resolveIncidents(&monitor, incident.Timeout)
	assert.False(t, result)
}

func TestHandleSSL_Create(t *testing.T) {
	db, _ := database.InitializeTestDatabase()
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)
	expiredDuration := time.Duration(31 * 7 * 24 * time.Hour)
	expiredDate := time.Now()

	monitor := models.Monitor{CertificateExpiredBefore: &expiredDuration}
	checkResult := net.CheckResults{SSLExpiredDate: &expiredDate}

	result := uptimeMonitor.handleSSL(&monitor, &checkResult)

	assert.True(t, result)
}

func TestHandleSSL_Solve(t *testing.T) {
	db, _ := database.InitializeTestDatabase()
	uptimeMonitor, _ := NewUptimeMonitor(db, nil)
	expiredDuration := time.Duration(31 * 7 * 24 * time.Hour)
	expiredDate := time.Now().Add(99999999 * time.Minute)

	checkResult := net.CheckResults{SSLExpiredDate: &expiredDate}
	monitor := models.Monitor{
		CertificateExpiredBefore: &expiredDuration,
		Incidents: []models.Incident{
			{Type: incident.SSLExpired},
		},
	}

	db.DB.Create(&monitor)

	result := uptimeMonitor.handleSSL(&monitor, &checkResult)

	assert.True(t, result)
}
