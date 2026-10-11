package feishu

import (
	"encoding/json"
	"github.com/chenhg5/cc-connect/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueenCardStatesAndPalette(t *testing.T) {
	states := []core.CardStatus{core.CardStatusWorking, core.CardStatusDone, core.CardStatusError, core.CardStatusStopped}
	for _, status := range states {
		for _, buttons := range []bool{false, true} {
			for _, color := range []string{"blue", "green", "violet", "orange", "yellow", "invalid", "red", "carmine"} {
				t.Run(string(status)+"/"+color+"/"+stringBool(buttons), func(t *testing.T) {
					running := status == core.CardStatusWorking
					opts := core.QueenCardOptions{Project: "demo", SessionKey: "test:demo", Language: core.LangChinese, Display: core.QueenCardDisplay{HeaderColor: color, HeaderIcon: "🧭", DisplayName: "Guide-A", CardButtons: buttons}}
					raw := buildQueenCard(status, nil, "正文", running, "🧠 model · 💪 high", opts)
					var card map[string]any
					if err := json.Unmarshal([]byte(raw), &card); err != nil {
						t.Fatal(err)
					}
					header := card["header"].(map[string]any)
					expected := core.NormalizeHeaderColor(color)
					if status == core.CardStatusError {
						expected = "red"
					}
					if header["template"] != expected {
						t.Fatal(header)
					}
					if header["title"].(map[string]any)["content"] != "🧭 Guide-A" {
						t.Fatal(header)
					}
					if card["config"].(map[string]any)["streaming_mode"] != running {
						t.Fatal("streaming state")
					}
					if card["schema"] != "2.0" || !strings.Contains(raw, "main_text") {
						t.Fatal("schema/body")
					}
					elements := card["body"].(map[string]any)["elements"].([]any)
					if elements[0].(map[string]any)["element_id"] != "main_text" {
						t.Fatal("body must lead")
					}
					count := 0
					for _, element := range elements {
						m := element.(map[string]any)
						if m["tag"] != "column_set" {
							continue
						}
						count++
						columns := m["columns"].([]any)
						if len(columns) != 4 {
							t.Fatal("button count")
						}
						for i, column := range columns {
							button := column.(map[string]any)["elements"].([]any)[0].(map[string]any)
							disabled := running
							if i == 3 {
								disabled = !running
								if button["confirm"] == nil {
									t.Fatal("stop confirmation")
								}
							}
							if button["disabled"] != disabled {
								t.Fatal(button)
							}
							behavior := button["behaviors"].([]any)[0].(map[string]any)
							if behavior["value"].(map[string]any)["session_key"] != "test:demo" {
								t.Fatal("callback routing")
							}
						}
					}
					if (count == 1) != buttons {
						t.Fatal("button configuration")
					}
				})
			}
		}
	}
}
func stringBool(value bool) string {
	if value {
		return "buttons"
	}
	return "no_buttons"
}

func queenSelectorFixture() *core.Card {
	return core.NewCard().Title("换模型", "indigo").Markdown("当前：gpt-6.1-sol").Select("选择模型", []core.CardSelectOption{{Text: "gpt-6.1-sol", Value: "act:/model switch 1"}, {Text: "gpt-6-astra", Value: "act:/model switch 2"}}, "act:/model switch 1").Buttons(core.CardButton{Text: "换强度", Value: "nav:/reasoning"}, core.CardButton{Text: "返回", Value: "nav:/current"}).Build()
}
func TestQueenSelectorUsesCardV2(t *testing.T) {
	card := renderCardMap(queenSelectorFixture(), "test:demo")
	if card["schema"] != "2.0" {
		t.Fatal(card)
	}
	elements := card["body"].(map[string]any)["elements"].([]map[string]any)
	selectMap := elements[1]
	if selectMap["tag"] != "select_static" || selectMap["initial_index"] != 0 {
		t.Fatal(selectMap)
	}
	value := selectMap["behaviors"].([]map[string]any)[0]["value"].(map[string]string)
	if value["session_key"] != "test:demo" || value["action"] != "" {
		t.Fatal("selected option must route the action", value)
	}
}
func TestQueenCardPreservesControlsAtSizeLimit(t *testing.T) {
	raw := buildQueenCard(core.CardStatusDone, nil, strings.Repeat("正文", 20000), false, "footer", core.QueenCardOptions{Project: "demo", Display: core.QueenCardDisplay{CardButtons: true}})
	if len(raw) > maxRichCardJSONBytes || !strings.Contains(raw, "nav:/model") || !strings.Contains(raw, "footer") {
		t.Fatal("oversized body lost controls or footer")
	}
}

// Set QUEENCARD_RENDER_DIR only to export locally generated review artifacts.
func TestExportQueenCards(t *testing.T) {
	dir := os.Getenv("QUEENCARD_RENDER_DIR")
	if dir == "" {
		t.Skip("export not requested")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	steps := []core.ToolStep{{Kind: core.ToolStepKindThinking, Name: "Thinking", Summary: "检查卡片布局"}, {Name: "Read", Summary: "读取设计规格"}}
	states := []core.CardStatus{core.CardStatusWorking, core.CardStatusDone, core.CardStatusError, core.CardStatusStopped}
	names := []string{"01-running", "02-done", "03-error", "04-stopped"}
	for i, status := range states {
		body := "QueenCard 本地生成验收示例。"
		footer := "🧠 gpt-6.1-sol · 💪 high · ⌛ 42s\n📝 上下文 ▓▓▓▓░░░░ 48%"
		if i == 0 {
			footer = "🧠 gpt-6.1-sol · 💪 high · ⏳ 进行中"
		}
		if i == 2 {
			body += "\n\n" + core.NewI18n(core.LangChinese).Tf(core.MsgQueenErrorNote, "示例错误")
		}
		if i == 3 {
			body += "\n\n" + core.NewI18n(core.LangChinese).T(core.MsgQueenStopNote)
			footer = strings.Replace(footer, "42s", "12s（停止时）", 1)
		}
		raw := buildQueenCard(status, steps, body, i == 0, footer, core.QueenCardOptions{Project: "queencard-demo", SessionKey: "demo:review", Language: core.LangChinese, Display: core.QueenCardDisplay{HeaderColor: "blue", HeaderIcon: "🧭", DisplayName: "Guide-A", CardButtons: true}})
		writeQueenJSON(t, dir, names[i], []byte(raw))
	}
	raw, err := json.Marshal(renderCardMap(queenSelectorFixture(), "demo:review"))
	if err != nil {
		t.Fatal(err)
	}
	writeQueenJSON(t, dir, "05-switch-model", raw)
}
func writeQueenJSON(t *testing.T, dir, name string, raw []byte) {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), pretty, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestQueenCardOptionalPanelsAndAvatar(t *testing.T) {
	hide := false
	opts := core.QueenCardOptions{Project: "demo", Display: core.QueenCardDisplay{CardShowThinking: &hide, CardShowTools: &hide, CardAvatarImgKey: "review-existing-image-key"}}
	raw := buildQueenCard(core.CardStatusWorking, []core.ToolStep{{Kind: core.ToolStepKindThinking, Summary: "thought"}, {Summary: "tool"}}, "body", true, "", opts)
	if strings.Contains(raw, "collapsible_panel") || !strings.Contains(raw, "review-existing-image-key") {
		t.Fatal(raw)
	}
	opts.Display.CardAvatarImgKey = ""
	if strings.Contains(buildQueenCard(core.CardStatusDone, nil, "body", false, "", opts), "custom_icon") {
		t.Fatal("empty avatar rendered")
	}
}
func TestQueenCardV2CallbackRoutesNavigationAndSelection(t *testing.T) {
	platform, err := New(map[string]any{"app_id": "test_app", "app_secret": "test_fixture", "enable_feishu_card": true})
	if err != nil {
		t.Fatal(err)
	}
	ip := platform.(*interactivePlatform)
	for _, option := range []string{"", "act:/model switch 2"} {
		expected := "nav:/model"
		values := map[string]any{"session_key": "demo:review"}
		if option == "" {
			values["action"] = expected
		} else {
			expected = option
		}
		called := false
		ip.cardNavHandler = func(action, key string) *core.Card {
			called = true
			if action != expected || key != "demo:review" {
				t.Errorf("route %s %s", action, key)
			}
			return queenSelectorFixture()
		}
		event := &callback.CardActionTriggerEvent{Event: &callback.CardActionTriggerRequest{Action: &callback.CallBackAction{Value: values, Option: option}, Operator: &callback.Operator{OpenID: "test_user"}, Context: &callback.Context{OpenChatID: "test_chat", OpenMessageID: "test_message"}}}
		response, err := ip.onCardAction(event)
		if err != nil || response == nil || response.Card == nil || !called {
			t.Fatal(response, err)
		}
	}
}
