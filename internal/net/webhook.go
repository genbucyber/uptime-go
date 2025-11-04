package net

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"time"
	"uptime-go/internal/configuration"
	"uptime-go/internal/incident"
	"uptime-go/internal/models"
)

type incidentResponse struct {
	Message string `json:"message"`
	Data    struct {
		ID uint64 `json:"incident_id"`
	} `json:"data"`
}

func sendRequest(method string, url string, payload any) (*http.Response, []byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	token := configuration.Config.Main.Auth.Token
	if token == "" {
		log.Printf("[webhook] invalid server token")
		return nil, nil, fmt.Errorf("error creating request for %s: invalid server token", url)
	}

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request payload: %w", err)
		}
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating request for %s: %w", url, err)
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request to %s: %w", url, err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return response, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return response, respBody, nil
}

func NotifyIncident(incident *models.Incident, severity incident.Severity, attributes map[string]any) (uint64, error) {
	if incident.Monitor.IsNotExists() {
		return 0, fmt.Errorf("incident monitor data is not properly initialized")
	}

	ipAddress, err := GetIPAddress()
	if err != nil {
		log.Printf("[webhook] Failed to send incident notification for %s: failed to get server ip address %v", incident.Monitor.URL, err)
		return 0, err
	}

	// Default attributes
	attr := map[string]any{
		"url": incident.Monitor.URL,
	}

	maps.Copy(attr, attributes)

	payload := struct {
		ServerIP   string         `json:"server_ip"`
		Module     string         `json:"module"`
		Severity   string         `json:"severity"`
		Message    string         `json:"message"`
		Event      string         `json:"event"`
		Tags       []string       `json:"tags"`
		Attributes map[string]any `json:"attributes,omitempty"`
	}{
		ServerIP:   ipAddress,
		Module:     "UptimePlugin",
		Severity:   string(severity),
		Message:    incident.Description,
		Event:      "website_down",
		Tags:       []string{"uptime", "monitoring", string(incident.Type)},
		Attributes: attr,
	}

	response, body, err := sendRequest("POST", configuration.GetIncidentCreateURL(), payload)
	if err != nil {
		log.Printf("[webhook] Failed to send incident notification for %s: %v", incident.Monitor.URL, err)
		return 0, err
	}

	if response.StatusCode != http.StatusCreated {
		err := fmt.Errorf("failed to create incident, received status code %d. Body: %s", response.StatusCode, string(body))
		log.Printf("[webhook] %v", err)
		return 0, err
	}

	var result incidentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		err := fmt.Errorf("failed to decode incident response body: %w. Body: %s", err, string(body))
		log.Printf("[webhook] %v", err)
		return 0, err
	}

	log.Printf("[webhook] Successfully created incident for monitor %s - Reason: %s - Incident Master ID: %d", incident.Monitor.URL, incident.Type, result.Data.ID)
	return result.Data.ID, nil
}

func UpdateIncidentStatus(incident *models.Incident, status incident.Status) error {
	if incident.IncidentID == 0 {
		log.Printf("[webhook] Failed to update incident status for %s: incident_id not set", incident.ID)
	}

	payload := struct {
		Status string `json:"status"`
	}{Status: string(status)}

	url := configuration.GetIncidentStatusURL(incident.IncidentID)
	response, body, err := sendRequest("POST", url, payload)
	if err != nil {
		log.Printf("[webhook] Failed to send status update for incident %d: %v", incident.IncidentID, err)
		return err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to update incident status, received status code %d. Body: %s", response.StatusCode, string(body))
		log.Printf("[webhook] %v", err)
		return err
	}

	var result incidentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		err := fmt.Errorf("failed to decode update status response body: %w. Body: %s", err, string(body))
		log.Printf("[webhook] %v", err)
		return err
	}

	log.Printf("[webhook] Successfully updated status for incident %d to '%s'. Message: %s", incident.IncidentID, status, result.Message)
	return nil
}
