package config

import "testing"

func TestQueenCardProjectDisplayOverrides(t *testing.T) {
	color := "violet"
	icon := "🐻"
	name := "Bear"
	buttons := false
	cfg := &Config{Display: DisplayConfig{HeaderColor: &color, HeaderIcon: &icon, DisplayName: &name}}
	proj := &ProjectConfig{Display: &DisplayConfig{CardButtons: &buttons}}
	got := EffectiveQueenCardDisplay(cfg, proj)
	if got.HeaderColor != "violet" || got.HeaderIcon != "🐻" || got.DisplayName != "Bear" || got.CardButtons || got.QuotaCacheSeconds != 300 {
		t.Fatal(got)
	}
	bad := "carmine"
	proj.Display.HeaderColor = &bad
	if EffectiveQueenCardDisplay(cfg, proj).HeaderColor != "default" {
		t.Fatal("reserved color")
	}
	proj.Display = nil
	if !EffectiveQueenCardDisplay(cfg, proj).CardButtons {
		t.Fatal("global fallback")
	}
}
