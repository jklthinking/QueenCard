package core

import (
	"strings"
	"testing"
)

type queenCardEffortAgent struct{ stubModelModeAgent }

func (a *queenCardEffortAgent) AvailableReasoningEfforts() []string {
	return []string{"low", "medium", "high", "xhigh", "max", "ultra"}
}
func TestQueenCardSixEffortsCommandCardAndFooter(t *testing.T) {
	for _, effort := range []string{"max", "ultra"} {
		p := &stubPlatformEngine{n: "plain"}
		a := &queenCardEffortAgent{}
		e := NewEngine("test", a, []Platform{p}, "", LangEnglish)
		msg := &Message{SessionKey: "test:u", ReplyCtx: "ctx"}
		s := e.sessions.GetOrCreateActive(msg.SessionKey)
		s.SetAgentSessionID("old", "test")
		e.cmdReasoning(p, msg, []string{effort})
		if a.GetReasoningEffort() != effort || s.GetAgentSessionID() != "" {
			t.Fatal("switch failed", effort)
		}
		if !strings.Contains(p.sent[0], effort) {
			t.Fatal(p.sent)
		}
		card := e.renderReasoningCard()
		if !strings.Contains(card.RenderText(), "ultra") || !strings.Contains(card.RenderText(), "max") {
			t.Fatal(card.RenderText())
		}
		if replyFooterReasoningEffort(nil, a) != effort {
			t.Fatal("footer")
		}
	}
}
