package codex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueenCardCodexSixEffortsAndLaunch(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	cache := `{"client_version":"0.159.2","models":[{"slug":"gpt-6.1-sol","visibility":"list","supported_in_api":true,"supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"},{"effort":"max"},{"effort":"ultra"}]}]}`
	if err := os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "models_cache.json"), []byte(cache), 0600); err != nil {
		t.Fatal(err)
	}
	a := &Agent{model: "gpt-6.1-sol", activeIdx: -1}
	if got := strings.Join(a.AvailableReasoningEfforts(), ","); got != "low,medium,high,xhigh,max,ultra" {
		t.Fatal(got)
	}
	for _, effort := range a.AvailableReasoningEfforts() {
		a.SetReasoningEffort(strings.ToUpper(effort))
		if a.GetReasoningEffort() != effort {
			t.Fatal(effort)
		}
		cs, err := newCodexSession(context.Background(), "codex", nil, t.TempDir(), a.GetModel(), a.GetReasoningEffort(), "full-auto", "", "", nil, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		defer cs.Close()
		if !containsSequence(cs.buildExecArgs("hello", nil), []string{"-c", `model_reasoning_effort="` + effort + `"`}) {
			t.Fatal(effort)
		}
	}
	a.SetModel("unknown")
	if len(a.AvailableReasoningEfforts()) != 4 {
		t.Fatal("fallback")
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "models_cache.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if len(a.AvailableReasoningEfforts()) != 4 {
		t.Fatal("malformed fallback")
	}
}
func TestQueenCardCachePrecedesOldAPIAndFiltersHide(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "models_cache.json"), []byte(`{"client_version":"0.159.2","models":[{"slug":"gpt-6.1-sol","visibility":"list","supported_in_api":true},{"slug":"old-hidden","visibility":"hide","supported_in_api":true}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("cache should avoid API")
		w.Write([]byte(`{"data":[{"id":"o3"}]}`))
	}))
	defer srv.Close()
	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	a := &Agent{activeIdx: -1}
	got := a.AvailableModels(context.Background())
	if len(got) != 1 || got[0].Name != "gpt-6.1-sol" {
		t.Fatal(got)
	}
}
func TestQueenCardFallbackCurrentModels(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	a := &Agent{activeIdx: -1}
	got := a.AvailableModels(context.Background())
	if len(got) != 7 || got[0].Name != "gpt-6.1-sol" || got[6].Name != "gpt-5.6-luna" {
		t.Fatal(got)
	}
}
func TestQueenCardDefaultConfiguredModelEfforts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	for name, content := range map[string]string{"config.toml": "model = 'gpt-6.1-sol'\n", "models_cache.json": `{"models":[{"slug":"gpt-6.1-sol","supported_reasoning_levels":[{"effort":"max"},{"effort":"ultra"}]}]}`} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a := &Agent{activeIdx: -1}
	if got := strings.Join(a.AvailableReasoningEfforts(), ","); got != "max,ultra" {
		t.Fatal(got)
	}
}
