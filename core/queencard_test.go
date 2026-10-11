package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueenCardStreamingFooterAndUsageFallback(t *testing.T) {
	e := NewEngine("demo", &stubAgent{}, nil, "", LangChinese)
	e.SetReplyFooterEnabled(true)
	e.SetShowContextIndicator(true)
	e.SetShowWorkdirIndicator(false)
	session := &controllableAgentSession{model: "gpt-6.1-sol", reasoningEffort: "high"}
	agent := &stubUsageAgent{report: &UsageReport{Buckets: []UsageBucket{{Windows: []UsageWindow{{Name: "weekly", UsedPercent: 37, WindowSeconds: 7 * 24 * 3600}}}}}}
	streaming := e.composeRichStatusFooter(true, time.Now(), agent, session, "")
	if !strings.Contains(streaming, "⏳ 进行中") || !strings.Contains(streaming, "gpt-6.1-sol") || strings.Contains(streaming, "\n") || strings.Contains(streaming, "耗时") {
		t.Fatal(streaming)
	}
	final := e.composeRichStatusFooter(false, time.Now(), agent, session, "")
	if len(strings.Split(final, "\n")) != 2 || !strings.Contains(final, "63%") {
		t.Fatal("usage fallback", final)
	}
}
func TestQueenCardQuotaHooksAndCache(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"used_percent":63,"label":"额度","resets_at":"2026-10-12T00:00:00+08:00"}`))
	}))
	defer server.Close()
	e := NewEngine("demo", &stubAgent{}, nil, "", LangChinese)
	e.display.QueenCard = QueenCardDisplay{CardShowQuota: true, QuotaURL: server.URL, QuotaCacheSeconds: 300}
	if got := e.queenQuotaText(); got != "💰 额度 63%" {
		t.Fatal(got)
	}
	_ = e.queenQuotaText()
	if calls.Load() != 1 {
		t.Fatal("cache missed")
	}
	e.display.QueenCard.CardShowQuota = false
	if e.queenQuotaText() != "" {
		t.Fatal("disabled hook")
	}
	e.display.QueenCard = QueenCardDisplay{CardShowQuota: true, QuotaCommand: `printf '{"used_percent":21}'`}
	if got := e.queenQuotaText(); got != "💰 本周额度 21%" {
		t.Fatal(got)
	}
	e.display.QueenCard.QuotaCommand = `printf '{"used_percent":101}'`
	if e.queenQuotaText() != "" {
		t.Fatal("invalid percent")
	}
	e.display.QueenCard.QuotaCommand = `printf '{}'`
	if e.queenQuotaText() != "" {
		t.Fatal("missing percent")
	}
}
func TestQueenCardQuotaHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := fetchQueenQuota(ctx, QueenCardDisplay{QuotaURL: server.URL}); err == nil {
		t.Fatal("timeout missing")
	}
}
func TestQueenCardUpdateChecksOfflineAndLocalized(t *testing.T) {
	if _, err := CheckForUpdate("v1.0.0", false); err == nil {
		t.Fatal("update check must reject")
	}
	e := newTestEngine()
	e.i18n = NewI18n(LangChinese)
	if got := e.renderUpgradeCard().RenderText(); !strings.Contains(got, "本版本已禁用自更新") {
		t.Fatal(got)
	}
	for _, lang := range []Language{LangEnglish, LangChinese, LangTraditionalChinese, LangJapanese, LangSpanish} {
		for _, key := range []MsgKey{MsgSelfUpdateDisabled, MsgQueenRunning, MsgQueenDone, MsgQueenError, MsgQueenStopped, MsgQueenStopNote, MsgQueenConfirmStop, MsgQueenQuota} {
			if messages[key][lang] == "" {
				t.Fatal(key, lang)
			}
		}
	}
}
func TestQueenCardStopFinalizesExistingRichCard(t *testing.T) {
	p := &stubCompactProgressPlatform{stubPlatformEngine: stubPlatformEngine{n: "test"}, style: "card", supportPayload: true}
	e := NewEngine("demo", &stubAgent{}, []Platform{p}, "", LangEnglish)
	e.SetDisplayConfig(DisplayCfg{CardMode: "rich", ThinkingMessages: true, ToolMessages: true})
	key := "test:stopped"
	session := e.sessions.GetOrCreateActive(key)
	agentSession := newControllableSession("stopped-session")
	state := &interactiveState{agentSession: agentSession, platform: p, replyCtx: "ctx"}
	agentSession.events <- Event{Type: EventThinking, Content: "kept reasoning"}
	agentSession.events <- Event{Type: EventText, Content: "kept body"}
	done := make(chan struct{})
	go func() {
		e.processInteractiveEvents(state, session, e.sessions, key, "msg", time.Now(), nil, nil, "ctx")
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for len(p.getPreviewStarts()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	// Wait until the text has been processed, then signal stop on the same turn.
	time.Sleep(20 * time.Millisecond)
	state.markStopped()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stop loop did not finish")
	}
	edits := p.getPreviewEdits()
	if len(edits) == 0 {
		t.Fatal("no terminal update")
	}
	last := edits[len(edits)-1]
	for _, want := range []string{"status=stopped", "streaming=false", "kept body", "kept reasoning", "Execution stopped"} {
		if !strings.Contains(last, want) {
			t.Fatal(want, last)
		}
	}
}

func TestQueenCardStoppedFooterUsesCapturedElapsed(t *testing.T) {
	e := NewEngine("demo", &stubAgent{}, nil, "", LangChinese)
	e.SetReplyFooterEnabled(true)
	for _, elapsed := range []time.Duration{99 * time.Millisecond, 100 * time.Millisecond, 59999 * time.Millisecond, time.Minute} {
		got := e.composeRichStatusFooterElapsed(false, elapsed, &stubAgent{}, nil, "", true)
		want := e.i18n.Tf(MsgQueenStoppedAt, e.richStatusText(MsgRichStatusElapsed, formatRichStatusDuration(elapsed)))
		if !strings.Contains(got, want) || strings.Count(got, "（停止时）") != 1 {
			t.Fatalf("elapsed %v: %s", elapsed, got)
		}
	}
}

func TestQueenCardStoppedResolvesMarkdownImages(t *testing.T) {
	p := &stubRichCardResolverPlatform{stubRichCardSilentPlatform: &stubRichCardSilentPlatform{stubPlatformEngine: stubPlatformEngine{n: "test"}}}
	e := NewEngine("demo", &stubAgent{}, []Platform{p}, "", LangEnglish)
	e.SetDisplayConfig(DisplayCfg{CardMode: "rich", ThinkingMessages: true, ToolMessages: true})
	key := "test:stop-image"
	session := e.sessions.GetOrCreateActive(key)
	as := newControllableSession("stop-image")
	state := &interactiveState{agentSession: as, platform: p, replyCtx: "ctx"}
	as.events <- Event{Type: EventText, Content: "![chart](https://example.com/chart.png)"}
	done := make(chan struct{})
	go func() {
		e.processInteractiveEvents(state, session, e.sessions, key, "msg", time.Now(), nil, nil, "ctx")
		close(done)
	}()
	deadline := time.After(time.Second)
	for {
		starts, _, _, _ := p.snapshot()
		if len(starts) > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("no initial card")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	state.markStopped()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stop timed out")
	}
	_, _, updates, _ := p.snapshot()
	if len(updates) == 0 {
		t.Fatal("no terminal card")
	}
	last := updates[len(updates)-1]
	if !strings.Contains(last, "img_v3_chart") || strings.Contains(last, "https://example.com/chart.png") {
		t.Fatal(last)
	}
	modes := p.resolverCallModes()
	if len(modes) == 0 || !modes[len(modes)-1] {
		t.Fatal("stop did not resolve final markdown", modes)
	}
}
