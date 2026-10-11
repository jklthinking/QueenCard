package feishu

import (
	"github.com/chenhg5/cc-connect/core"
	"strings"
)

func (p *Platform) BuildQueenCard(status core.CardStatus, steps []core.ToolStep, markdown string, streaming bool, footer string, options core.QueenCardOptions) string {
	return buildQueenCard(status, steps, markdown, streaming, footer, options)
}
func queenPanelElements(steps []core.ToolStep, empty string, tr *core.I18n) []map[string]any {
	if len(steps) == 0 {
		return []map[string]any{richPlaceholderElement(empty)}
	}
	var elements []map[string]any
	if len(steps) > 10 {
		elements = append(elements, richPlaceholderElement(tr.Tf(core.MsgQueenHidden, len(steps)-10)))
		steps = steps[len(steps)-10:]
	}
	for _, step := range steps {
		elements = append(elements, richStepElement(step))
	}
	return elements
}
func queenCardButtons(running bool, sessionKey string, tr *core.I18n) map[string]any {
	keys := []core.MsgKey{core.MsgQueenModel, core.MsgQueenEffort, core.MsgQueenNew, core.MsgQueenStop}
	actions := []string{"nav:/model", "nav:/reasoning", "act:/new", "act:/stop"}
	columns := []map[string]any{}
	for i, key := range keys {
		disabled := running
		tip := core.MsgQueenAfterTurn
		kind := "default"
		if i == 3 {
			disabled = !running
			tip = core.MsgQueenNoTask
			if running {
				kind = "danger"
			}
		}
		button := map[string]any{"tag": "button", "text": plainText(tr.T(key)), "type": kind, "width": "fill", "disabled": disabled,
			"behaviors": []map[string]any{{"type": "callback", "value": map[string]string{"action": actions[i], "session_key": sessionKey}}}}
		if disabled {
			button["disabled_tips"] = plainText(tr.T(tip))
		}
		if i == 3 {
			button["confirm"] = map[string]any{"title": plainText(tr.T(core.MsgQueenConfirmStop)), "text": plainText(tr.T(core.MsgQueenConfirmText))}
		}
		columns = append(columns, map[string]any{"tag": "column", "width": "weighted", "weight": 1, "elements": []map[string]any{button}})
	}
	return map[string]any{"tag": "column_set", "flex_mode": "flow", "columns": columns}
}

// renderQueenSelector uses the existing core selector and action protocol in Card 2.0.
func renderQueenSelector(card *core.Card, sessionKey string) (map[string]any, bool) {
	isQueen := false
	for _, element := range card.Elements {
		if selectCard, ok := element.(core.CardSelect); ok && len(selectCard.Options) > 0 {
			value := selectCard.Options[0].Value
			isQueen = strings.HasPrefix(value, "act:/model ") || strings.HasPrefix(value, "act:/reasoning ")
		}
	}
	if !isQueen {
		return nil, false
	}
	var elements []map[string]any
	for _, element := range card.Elements {
		switch value := element.(type) {
		case core.CardMarkdown:
			elements = append(elements, map[string]any{"tag": "markdown", "content": value.Content})
		case core.CardNote:
			elements = append(elements, map[string]any{"tag": "markdown", "content": value.Text, "text_size": "notation"})
		case core.CardSelect:
			options := []map[string]any{}
			selected := -1
			for i, opt := range value.Options {
				options = append(options, map[string]any{"text": plainText(opt.Text), "value": opt.Value})
				if opt.Value == value.InitValue {
					selected = i
				}
			}
			selectMap := map[string]any{"tag": "select_static", "placeholder": plainText(value.Placeholder), "options": options,
				"behaviors": []map[string]any{{"type": "callback", "value": map[string]string{"session_key": sessionKey}}}}
			if selected >= 0 {
				selectMap["initial_index"] = selected
			}
			elements = append(elements, selectMap)
		case core.CardActions:
			columns := []map[string]any{}
			for _, btn := range value.Buttons {
				kind := btn.Type
				if kind == "" {
					kind = "default"
				}
				button := map[string]any{"tag": "button", "text": plainText(btn.Text), "type": kind, "width": "fill",
					"behaviors": []map[string]any{{"type": "callback", "value": map[string]string{"action": btn.Value, "session_key": sessionKey}}}}
				columns = append(columns, map[string]any{"tag": "column", "width": "weighted", "weight": 1, "elements": []map[string]any{button}})
			}
			elements = append(elements, map[string]any{"tag": "column_set", "flex_mode": "flow", "columns": columns})
		}
	}
	return map[string]any{"schema": "2.0", "config": map[string]any{"update_multi": true}, "header": map[string]any{"title": plainText(card.Header.Title), "template": card.Header.Color}, "body": map[string]any{"padding": "12px", "elements": elements}}, true
}
