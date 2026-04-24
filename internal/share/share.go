package share

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/atotto/clipboard"

	"github.com/rifat977/standup/internal/config"
)

// Copy puts s on the system clipboard.
func Copy(s string) error {
	return clipboard.WriteAll(s)
}

// PostSlack sends s to the configured Slack incoming webhook.
func PostSlack(cfg *config.Config, s string) error {
	if cfg.Slack.WebhookURL == "" {
		return errors.New("slack webhook_url not set")
	}
	body := map[string]string{"text": s}
	if cfg.Slack.Channel != "" {
		body["channel"] = cfg.Slack.Channel
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.Slack.WebhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %s", resp.Status)
	}
	return nil
}
