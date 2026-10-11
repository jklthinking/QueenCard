package core

import (
	"log/slog"
	"strings"
)

// QueenCardDisplay configures project identity and optional quota metadata.
type QueenCardDisplay struct {
	CardAvatarImgKey  string
	CardShowThinking  *bool
	CardShowTools     *bool
	HeaderColor       string
	HeaderIcon        string
	DisplayName       string
	CardButtons       bool
	CardShowQuota     bool
	QuotaCommand      string
	QuotaURL          string
	QuotaLabel        string
	QuotaCacheSeconds int
}
type QueenCardOptions struct {
	Display    QueenCardDisplay
	Project    string
	SessionKey string
	Language   Language
}

// QueenCardSupporter adds per-turn identity without storing session state on a platform.
type QueenCardSupporter interface {
	BuildQueenCard(status CardStatus, steps []ToolStep, markdown string, streaming bool, footer string, options QueenCardOptions) string
}

func NormalizeHeaderColor(color string) string {
	color = strings.ToLower(strings.TrimSpace(color))
	switch color {
	case "default", "blue", "wathet", "indigo", "turquoise", "green", "yellow", "orange", "violet", "purple", "grey":
		return color
	case "":
		return "default"
	}
	slog.Warn("QueenCard: invalid or reserved header color; using default", "color", color)
	return "default"
}
func (e *Engine) buildTurnRichCard(p RichCardSupporter, sessionKey string, status CardStatus, steps []ToolStep, markdown string, streaming bool, footer string) string {
	if builder, ok := p.(QueenCardSupporter); ok {
		return builder.BuildQueenCard(status, steps, markdown, streaming, footer, QueenCardOptions{e.display.QueenCard, e.name, sessionKey, e.i18n.CurrentLang()})
	}
	return p.BuildRichCard(status, "", steps, markdown, streaming, footer)
}
