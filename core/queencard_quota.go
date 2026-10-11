package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type queenQuotaCache struct {
	sync.Mutex
	at        time.Time
	signature string
	text      string
}

// Hook output is capped to prevent an optional metadata source exhausting memory.
type quotaOutput struct{ data []byte }

func (w *quotaOutput) Write(p []byte) (int, error) {
	if len(w.data)+len(p) > 8192 {
		return 0, fmt.Errorf("quota output exceeds limit")
	}
	w.data = append(w.data, p...)
	return len(p), nil
}
func (e *Engine) queenQuotaText() string {
	cfg := e.display.QueenCard
	if !cfg.CardShowQuota || (cfg.QuotaCommand == "" && cfg.QuotaURL == "") {
		return ""
	}
	cache := &e.queenQuota
	cache.Lock()
	defer cache.Unlock()
	signature := cfg.QuotaCommand + "\x00" + cfg.QuotaURL + "\x00" + cfg.QuotaLabel + "\x00" + string(e.i18n.CurrentLang())
	ttl := cfg.QuotaCacheSeconds
	if ttl <= 0 {
		ttl = 300
	}
	if signature == cache.signature && !cache.at.IsZero() && time.Since(cache.at) < time.Duration(ttl)*time.Second {
		return cache.text
	}
	cache.signature, cache.at, cache.text = signature, time.Now(), ""
	ctx, cancel := context.WithTimeout(e.ctx, 2*time.Second)
	defer cancel()
	data, err := fetchQueenQuota(ctx, cfg)
	if err != nil {
		slog.Warn("QueenCard: quota hook unavailable", "project", e.name)
		return ""
	}
	var result struct {
		UsedPercent *float64 `json:"used_percent"`
		Label       string   `json:"label"`
		ResetsAt    string   `json:"resets_at"`
	}
	if json.Unmarshal(data, &result) != nil || result.UsedPercent == nil || math.IsNaN(*result.UsedPercent) || *result.UsedPercent < 0 || *result.UsedPercent > 100 {
		slog.Warn("QueenCard: invalid quota hook JSON", "project", e.name)
		return ""
	}
	label := result.Label
	if label == "" {
		label = cfg.QuotaLabel
	}
	if label == "" {
		label = e.i18n.T(MsgQueenQuota)
	}
	label = strings.ReplaceAll(strings.ReplaceAll(label, "\n", " "), "\r", " ")
	cache.text = fmt.Sprintf("💰 %s %g%%", label, *result.UsedPercent)
	return cache.text
}
func fetchQueenQuota(ctx context.Context, cfg QueenCardDisplay) ([]byte, error) {
	if cfg.QuotaCommand != "" {
		cmd := exec.CommandContext(ctx, "sh", "-c", cfg.QuotaCommand)
		cmd.WaitDelay = 100 * time.Millisecond
		var output quotaOutput
		cmd.Stdout = &output
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		return output.data, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.QuotaURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quota HTTP status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if len(data) > 8192 {
		return nil, fmt.Errorf("quota output exceeds limit")
	}
	return data, err
}
