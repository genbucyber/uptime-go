package models

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"uptime-go/internal/helper"
	"uptime-go/internal/incident"

	"gorm.io/gorm"
)

type Monitor struct {
	ID                       string           `json:"-" gorm:"primaryKey"`
	URL                      string           `json:"url" gorm:"unique"`
	Enabled                  bool             `json:"-"`
	Interval                 time.Duration    `json:"-"`
	ResponseTimeThreshold    time.Duration    `json:"-"`
	CertificateMonitoring    bool             `json:"-"`
	CertificateExpiredBefore *time.Duration   `json:"-"`
	IsUp                     *bool            `json:"is_up"`
	StatusCode               *int             `json:"status_code"`
	ResponseTime             *int64           `json:"response_time"`
	CertificateExpiredDate   *time.Time       `json:"certificate_expired_date"`
	LastUp                   *time.Time       `json:"last_up"`
	LastDown                 *time.Time       `json:"last_down"`
	CreatedAt                time.Time        `json:"-"`
	UpdatedAt                time.Time        `json:"last_check"`
	Histories                []MonitorHistory `json:"histories,omitempty" gorm:"foreignKey:MonitorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Incidents                []Incident       `json:"-" gorm:"foreignKey:MonitorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type MonitorHistory struct {
	ID           string    `json:"-" gorm:"primaryKey"`
	MonitorID    string    `json:"-"`
	IsUp         bool      `json:"is_up" gorm:"index"`
	StatusCode   int       `json:"-"`
	ResponseTime int64     `json:"response_time"` // in milliseconds
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	Monitor      Monitor   `json:"-" gorm:"foreignKey:MonitorID"`
}

type Incident struct {
	ID          string        `json:"id" gorm:"primaryKey"`
	MonitorID   string        `json:"monitor_id"`
	IncidentID  uint64        `json:"-"`
	Type        incident.Type `json:"type" gorm:"index"`
	Description string        `json:"description"`
	CreatedAt   time.Time     `json:"created_at"`
	SolvedAt    *time.Time    `json:"solved_at" gorm:"index"`
	Monitor     Monitor       `gorm:"foreignKey:MonitorID"`
}

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *MonitorHistory) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID = helper.GenerateRandomID()

	return nil
}

func (r Response) Print() {
	data, err := json.Marshal(r)

	if err != nil {
		log.Printf("error serializing response: %v", err)
		return
	}

	fmt.Println(string(data))
}
