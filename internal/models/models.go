package models

import (
	"time"
	"uptime-go/internal/helper"
	"uptime-go/internal/incident"

	"gorm.io/gorm"
)

type Monitor struct {
	ID                       string           `json:"-" gorm:"primaryKey"`
	URL                      string           `json:"url" gorm:"unique"`
	Enabled                  bool             `json:"enabled"`
	Interval                 time.Duration    `json:"-"`
	ResponseTimeThreshold    time.Duration    `json:"-"`
	CertificateMonitoring    bool             `json:"-"`
	CertificateExpiredBefore *time.Duration   `json:"-"`
	FollowRedirects          bool             `json:"-"`
	IPType                   string           `json:"-"`
	IsUp                     *bool            `json:"is_up"`
	StatusCode               *int             `json:"status_code"`
	ResponseTime             *int64           `json:"response_time"` // in milliseconds
	CertificateExpiredDate   *time.Time       `json:"certificate_expired_date"`
	LastUp                   *time.Time       `json:"last_up"`
	LastDown                 *time.Time       `json:"last_down"`
	CreatedAt                time.Time        `json:"-"`
	UpdatedAt                time.Time        `json:"last_check"`
	Histories                []MonitorHistory `json:"histories,omitempty" gorm:"foreignKey:MonitorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Incidents                []Incident       `json:"-" gorm:"foreignKey:MonitorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// Retry configuration
	MaxRetries    int           `json:"-" gorm:"default:3"`
	RetryInterval time.Duration `json:"-" gorm:"default:60000000000"` // 60s in nanoseconds
	Retries       int           `json:"-" gorm:"default:0"`

	// Granular timeout configuration
	DNSTimeout            time.Duration `json:"-" gorm:"default:5000000000"`  // 5s in nanoseconds
	DialTimeout           time.Duration `json:"-" gorm:"default:10000000000"` // 10s in nanoseconds
	TLSHandshakeTimeout   time.Duration `json:"-" gorm:"default:10000000000"` // 10s in nanoseconds
	ResponseHeaderTimeout time.Duration `json:"-" gorm:"default:20000000000"` // 20s in nanoseconds
}

type MonitorHistory struct {
	ID           string    `json:"-" gorm:"primaryKey"`
	MonitorID    string    `json:"-" gorm:"index"`
	IsUp         bool      `json:"is_up" gorm:"index"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time"` // in milliseconds
	ContentSize	 int64	   `json:content_size`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	Monitor      Monitor   `json:"-" gorm:"foreignKey:MonitorID"`
}

type Incident struct {
	ID          string        `json:"id" gorm:"primaryKey"`
	MonitorID   string        `json:"monitor_id" gorm:"index"`
	IncidentID  uint64        `json:"-"`
	Type        incident.Type `json:"type" gorm:"index"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	SolvedAt    *time.Time    `json:"solved_at" gorm:"index"`
	Monitor     Monitor       `gorm:"foreignKey:MonitorID"`
}

type DailyStat struct{
	Date 		string
	TotalChecks int
	UpChecks 	int
}

func (h *MonitorHistory) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID = helper.GenerateRandomID()

	return nil
}

func (m Monitor) IsExists() bool {
	return !m.CreatedAt.IsZero()
}

func (m Monitor) IsNotExists() bool {
	return m.CreatedAt.IsZero()
}

func (i Incident) IsExists() bool {
	return !i.CreatedAt.IsZero()
}

func (i Incident) IsNotExists() bool {
	return i.CreatedAt.IsZero()
}

func (i Incident) IsSolved() bool {
	return i.SolvedAt != nil
}
