package codex

import (
	"encoding/json"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var reasoningCache = struct {
	sync.Mutex
	entries map[string]reasoningCacheEntry
}{entries: make(map[string]reasoningCacheEntry)}

type reasoningCacheEntry struct {
	at     time.Time
	levels []string
}

const reasoningCacheTTL = 5 * time.Second

func cachedReasoningEfforts(model string) []string {
	home, _ := resolveCodexHome(nil)
	key := home + "\x00" + model
	reasoningCache.Lock()
	defer reasoningCache.Unlock()
	if entry, ok := reasoningCache.entries[key]; ok && time.Since(entry.at) < reasoningCacheTTL {
		return append([]string(nil), entry.levels...)
	}
	levels := readReasoningEfforts(home, model)
	reasoningCache.entries[key] = reasoningCacheEntry{time.Now(), levels}
	return append([]string(nil), levels...)
}
func readReasoningEfforts(home, model string) []string {
	var cfg struct {
		Model   string `toml:"model"`
		Catalog string `toml:"model_catalog_json"`
	}
	_, _ = toml.DecodeFile(filepath.Join(home, "config.toml"), &cfg)
	if model == "" {
		model = cfg.Model
	}
	paths := []string{}
	if cfg.Catalog != "" {
		path := cfg.Catalog
		if strings.HasPrefix(path, "~/") {
			userHome, _ := os.UserHomeDir()
			path = filepath.Join(userHome, path[2:])
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(home, path)
		}
		paths = append(paths, path)
	}
	paths = append(paths, filepath.Join(home, "models_cache.json"))
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var payload struct {
			Models []struct {
				Slug   string `json:"slug"`
				Levels []struct {
					Effort string `json:"effort"`
				} `json:"supported_reasoning_levels"`
			} `json:"models"`
		}
		if json.Unmarshal(b, &payload) != nil {
			continue
		}
		for _, m := range payload.Models {
			if m.Slug != model {
				continue
			}
			var out []string
			seen := map[string]bool{}
			for _, l := range m.Levels {
				effort := normalizeReasoningEffort(l.Effort)
				if effort != "" && !seen[effort] {
					out = append(out, effort)
					seen[effort] = true
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}
