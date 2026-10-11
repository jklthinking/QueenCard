package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQueenCardMissingRolloutReturnsImmediately(t *testing.T) {
	cs, err := newCodexSession(context.Background(), "codex", nil, t.TempDir(), "", "", "", "missing", "", []string{"CODEX_HOME=" + t.TempDir()}, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	start := time.Now()
	cs.refreshContextUsageFromRollout()
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("missing file retried")
	}
	if time.Duration(codexContextUsageRetryCount-1)*codexContextUsageRetryDelay > 500*time.Millisecond {
		t.Fatal("retry budget")
	}
}
func TestQueenCardEffortsUseCatalogAndCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("model = \"gpt-6.1-sol\"\nmodel_catalog_json = \"catalog.json\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "catalog.json")
	if err := os.WriteFile(path, []byte(`{"models":[{"slug":"gpt-6.1-sol","supported_reasoning_levels":[{"effort":"max"},{"effort":"ultra"}]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	got := cachedReasoningEfforts("")
	if len(got) != 2 || got[1] != "ultra" {
		t.Fatal(got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	got[0] = "mutated"
	if next := cachedReasoningEfforts(""); len(next) != 2 || next[0] != "max" {
		t.Fatal("cache", next)
	}
}
