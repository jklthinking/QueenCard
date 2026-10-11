package acp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func TestIsGrokCLI(t *testing.T) {
	cases := map[string]bool{
		"grok":              true,
		"Grok":              true,
		"/usr/bin/grok":     true,
		`C:\Tools\grok.exe`: true,
		"true":              false,
		"claude":            false,
		"grok-cli":          false,
	}
	for in, want := range cases {
		if got := isGrokCLI(in); got != want {
			t.Fatalf("isGrokCLI(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestWrapIfGrok_ImplementsUsageReporter(t *testing.T) {
	grok := wrapIfGrok(&Agent{cmd: "grok"})
	if _, ok := grok.(core.UsageReporter); !ok {
		t.Fatal("grok ACP agent should implement UsageReporter")
	}
	other := wrapIfGrok(&Agent{cmd: "true"})
	if _, ok := other.(core.UsageReporter); ok {
		t.Fatal("non-grok ACP agent should not implement UsageReporter")
	}
	if _, ok := other.(*Agent); !ok {
		t.Fatalf("non-grok wrap type = %T, want *Agent", other)
	}
}

func TestMapGrokUsage_WeeklyWindow(t *testing.T) {
	pct := 66.0
	payload := grokCreditsResponse{Config: grokCreditsConfig{
		CreditUsagePercent:   10,
		IsUnifiedBillingUser: true,
	}}
	payload.Config.CurrentPeriod.Type = "USAGE_PERIOD_TYPE_WEEKLY"
	payload.Config.CurrentPeriod.Start = "2026-09-27T13:11:38.705586+00:00"
	payload.Config.CurrentPeriod.End = "2026-10-04T13:11:38.705586+00:00"
	payload.Config.ProductUsage = []struct {
		Product      string   `json:"product"`
		UsagePercent *float64 `json:"usagePercent"`
	}{
		{Product: "GrokBuild", UsagePercent: &pct},
		{Product: "GrokChat"},
	}

	now := time.Date(2026, 10, 4, 13, 1, 0, 0, time.UTC)
	report := mapGrokUsage(payload, grokAuthAccount{
		Email:  "user@example.com",
		UserID: "user-1",
		TeamID: "team-1",
	}, now)

	if report.Provider != "grok" || report.Plan != "SuperGrok" || report.Email != "user@example.com" {
		t.Fatalf("identity = %+v", report)
	}
	if len(report.Buckets) != 1 || len(report.Buckets[0].Windows) != 1 {
		t.Fatalf("buckets = %+v", report.Buckets)
	}
	win := report.Buckets[0].Windows[0]
	if win.UsedPercent != 66 {
		t.Fatalf("used = %d, want 66", win.UsedPercent)
	}
	if win.WindowSeconds != 604800 {
		t.Fatalf("window seconds = %d, want 604800", win.WindowSeconds)
	}
	if win.ResetAfterSeconds < 630 || win.ResetAfterSeconds > 650 {
		t.Fatalf("reset after = %d, want ~638", win.ResetAfterSeconds)
	}
	if report.Buckets[0].LimitReached {
		t.Fatal("66% should still be allowed")
	}
}

func TestFetchGrokUsage_Success(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.URL.String(); got != grokCreditsURL {
				t.Fatalf("url = %q", got)
			}
			if got := req.Header.Get("Authorization"); got != "Bearer token-123" {
				t.Fatalf("authorization = %q", got)
			}
			body := `{
			  "config": {
			    "currentPeriod": {
			      "type": "USAGE_PERIOD_TYPE_WEEKLY",
			      "start": "2026-09-27T13:11:38Z",
			      "end": "2026-10-04T13:11:38Z"
			    },
			    "creditUsagePercent": 66.0,
			    "productUsage": [{"product": "GrokBuild", "usagePercent": 66.0}],
			    "isUnifiedBillingUser": true,
			    "prepaidBalance": {"val": 0}
			  }
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	report, err := fetchGrokUsage(context.Background(), client, grokAuthAccount{
		AccessToken: "token-123",
		Email:       "user@example.com",
	})
	if err != nil {
		t.Fatalf("fetchGrokUsage: %v", err)
	}
	if report.Email != "user@example.com" || report.Plan != "SuperGrok" {
		t.Fatalf("report = %+v", report)
	}
	if report.Buckets[0].Windows[0].UsedPercent != 66 {
		t.Fatalf("used = %d", report.Buckets[0].Windows[0].UsedPercent)
	}
	if report.Credits == nil || report.Credits.HasCredits {
		t.Fatalf("credits = %+v", report.Credits)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
