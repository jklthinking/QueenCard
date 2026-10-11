package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

const grokCreditsURL = "https://cli-chat-proxy.grok.com/v1/billing?format=credits"

// grokACPAgent wraps a generic ACP agent whose CLI is Grok Build so
// cc-connect's builtin /usage can query the Grok weekly quota.
type grokACPAgent struct {
	*Agent
	settingsMu sync.Mutex
	model      string
	effort     string
	models     []core.ModelOption
	modelsAt   time.Time
}

var _ core.UsageReporter = (*grokACPAgent)(nil)

func isGrokCLI(cmd string) bool {
	cmd = strings.ReplaceAll(strings.TrimSpace(cmd), `\`, "/")
	base := strings.ToLower(filepath.Base(cmd))
	base = strings.TrimSuffix(base, ".exe")
	return base == "grok"
}

func wrapIfGrok(agent *Agent) core.Agent {
	if agent != nil && isGrokCLI(agent.cmd) {
		g := &grokACPAgent{Agent: agent}
		args := append(append([]string{}, agent.cliExtraArgs...), agent.args...)
		for i, arg := range args {
			key, value, hasValue := strings.Cut(arg, "=")
			if !hasValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
			}
			switch key {
			case "-m", "--model":
				g.model = strings.TrimSpace(value)
			case "--reasoning-effort", "--effort":
				g.effort = normalizeGrokEffort(value)
			}
		}
		return g
	}
	return agent
}

type grokAuthAccount struct {
	AccessToken string
	Email       string
	UserID      string
	TeamID      string
	ExpiresAt   time.Time
}

type grokCreditsResponse struct {
	Config grokCreditsConfig `json:"config"`
}

type grokCreditsConfig struct {
	CurrentPeriod struct {
		Type  string `json:"type"`
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"currentPeriod"`
	CreditUsagePercent   float64 `json:"creditUsagePercent"`
	IsUnifiedBillingUser bool    `json:"isUnifiedBillingUser"`
	ProductUsage         []struct {
		Product      string   `json:"product"`
		UsagePercent *float64 `json:"usagePercent"`
	} `json:"productUsage"`
	PrepaidBalance *struct {
		Val json.Number `json:"val"`
	} `json:"prepaidBalance"`
	BillingPeriodStart string `json:"billingPeriodStart"`
	BillingPeriodEnd   string `json:"billingPeriodEnd"`
}

func (a *grokACPAgent) GetUsage(ctx context.Context) (*core.UsageReport, error) {
	account, err := readGrokAuth(os.ReadFile)
	if err != nil {
		return nil, err
	}
	return fetchGrokUsage(ctx, http.DefaultClient, account)
}

func grokHomeDir() (string, error) {
	if home := strings.TrimSpace(os.Getenv("GROK_HOME")); home != "" {
		return home, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".grok"), nil
}

func readGrokAuth(readFile func(string) ([]byte, error)) (grokAuthAccount, error) {
	home, err := grokHomeDir()
	if err != nil {
		return grokAuthAccount{}, err
	}
	path := filepath.Join(home, "auth.json")
	data, err := readFile(path)
	if err != nil {
		return grokAuthAccount{}, fmt.Errorf("read %s: %w", path, err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return grokAuthAccount{}, fmt.Errorf("parse auth.json: %w", err)
	}

	now := time.Now()
	var best grokAuthAccount
	var bestScore int
	for issuer, blob := range raw {
		var entry struct {
			Key       string `json:"key"`
			Email     string `json:"email"`
			UserID    string `json:"user_id"`
			TeamID    string `json:"team_id"`
			ExpiresAt string `json:"expires_at"`
		}
		if json.Unmarshal(blob, &entry) != nil {
			continue
		}
		if strings.TrimSpace(entry.Key) == "" {
			continue
		}
		acct := grokAuthAccount{
			AccessToken: entry.Key,
			Email:       strings.TrimSpace(entry.Email),
			UserID:      strings.TrimSpace(entry.UserID),
			TeamID:      strings.TrimSpace(entry.TeamID),
		}
		if ts, err := time.Parse(time.RFC3339Nano, entry.ExpiresAt); err == nil {
			acct.ExpiresAt = ts
		} else if ts, err := time.Parse(time.RFC3339, entry.ExpiresAt); err == nil {
			acct.ExpiresAt = ts
		}
		score := 1
		if strings.Contains(strings.ToLower(issuer), "auth.x.ai") {
			score += 2
		}
		if !acct.ExpiresAt.IsZero() && acct.ExpiresAt.After(now) {
			score += 4
		}
		if score > bestScore || (score == bestScore && acct.ExpiresAt.After(best.ExpiresAt)) {
			best = acct
			bestScore = score
		}
	}
	if best.AccessToken == "" {
		return grokAuthAccount{}, fmt.Errorf("auth.json missing Grok OIDC access token")
	}
	return best, nil
}

func fetchGrokUsage(ctx context.Context, client *http.Client, account grokAuthAccount) (*core.UsageReport, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, grokCreditsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cc-connect")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Grok billing endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("Grok billing returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload grokCreditsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Grok billing response: %w", err)
	}
	return mapGrokUsage(payload, account, time.Now()), nil
}

func mapGrokUsage(payload grokCreditsResponse, account grokAuthAccount, now time.Time) *core.UsageReport {
	cfg := payload.Config
	used := cfg.CreditUsagePercent
	for _, product := range cfg.ProductUsage {
		if strings.EqualFold(product.Product, "GrokBuild") && product.UsagePercent != nil {
			used = *product.UsagePercent
			break
		}
	}
	usedPercent := int(math.Round(used))
	if usedPercent < 0 {
		usedPercent = 0
	}
	if usedPercent > 100 {
		usedPercent = 100
	}

	endRaw := firstNonEmpty(cfg.CurrentPeriod.End, cfg.BillingPeriodEnd)
	startRaw := firstNonEmpty(cfg.CurrentPeriod.Start, cfg.BillingPeriodStart)
	endAt := parseGrokTime(endRaw)
	startAt := parseGrokTime(startRaw)

	windowSeconds := 0
	if strings.Contains(strings.ToUpper(cfg.CurrentPeriod.Type), "WEEKLY") {
		windowSeconds = 604800
	} else if !startAt.IsZero() && !endAt.IsZero() && endAt.After(startAt) {
		windowSeconds = int(endAt.Sub(startAt).Seconds())
	}

	resetAfter := 0
	resetAtUnix := int64(0)
	if !endAt.IsZero() {
		resetAtUnix = endAt.Unix()
		if d := int(endAt.Sub(now).Seconds()); d > 0 {
			resetAfter = d
		}
	}

	plan := "GrokBuild"
	if cfg.IsUnifiedBillingUser {
		plan = "SuperGrok"
	}

	report := &core.UsageReport{
		Provider:  "grok",
		AccountID: account.TeamID,
		UserID:    account.UserID,
		Email:     account.Email,
		Plan:      plan,
		Buckets: []core.UsageBucket{{
			Name:         "GrokBuild",
			Allowed:      usedPercent < 100,
			LimitReached: usedPercent >= 100,
			Windows: []core.UsageWindow{{
				Name:              "Weekly",
				UsedPercent:       usedPercent,
				WindowSeconds:     windowSeconds,
				ResetAfterSeconds: resetAfter,
				ResetAtUnix:       resetAtUnix,
			}},
		}},
	}
	if cfg.PrepaidBalance != nil {
		bal := strings.TrimSpace(cfg.PrepaidBalance.Val.String())
		report.Credits = &core.UsageCredits{
			HasCredits: bal != "" && bal != "0",
			Balance:    bal,
		}
	}
	return report
}

func parseGrokTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func normalizeGrokEffort(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "low", "medium", "high", "xhigh":
		return value
	}
	return ""
}
