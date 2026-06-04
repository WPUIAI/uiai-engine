package credits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

type Service struct {
	cfg    *config.Config
	client *http.Client
}

func New(cfg *config.Config) *Service {
	return &Service{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Cost returns the credit cost for an operation.
func (s *Service) Cost(operation string) float64 {
	if c, ok := s.cfg.Credits.Costs[operation]; ok {
		return c
	}
	return 0
}

// Deduct sends a fire-and-forget credit deduction to WordPress.
// Call this in a goroutine after successful AI calls.
func (s *Service) Deduct(licenseID int, operation, reference string) {
	if licenseID <= 0 {
		return
	}

	url := s.cfg.RESTURL("credits/deduct")
	body, _ := json.Marshal(map[string]any{
		"license_id": licenseID,
		"operation":  operation,
		"reference":  reference,
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[credits] deduct request build failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", s.cfg.WordPress.WebhookSecret)

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[credits] deduct failed for license %d op=%s: %v", licenseID, operation, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[credits] deduct returned %d for license %d op=%s", resp.StatusCode, licenseID, operation)
	}
}

// CheckBalance returns whether a license can afford an operation.
func (s *Service) CheckBalance(licenseID int, operation string) (bool, float64, error) {
	url := s.cfg.RESTURL("credits/balance")
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s?license_id=%d", url, licenseID), nil)
	req.Header.Set("X-Webhook-Secret", s.cfg.WordPress.WebhookSecret)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	var data struct {
		Balance float64 `json:"balance"`
	}
	json.NewDecoder(resp.Body).Decode(&data)

	cost := s.Cost(operation)
	return data.Balance >= cost, data.Balance, nil
}
