package core

import "testing"

func TestQueenCardSelfUpdateRefusesAllTags(t *testing.T) {
	for _, tag := range []string{"v1.5.0", "", "../anything"} {
		if err := SelfUpdate(tag, false); err == nil || err.Error() != SelfUpdateDisabledMessage {
			t.Fatal(err)
		}
	}
}
func TestQueenCardUpgradeCommandsRefuse(t *testing.T) {
	p := &stubPlatformEngine{n: "test"}
	e := NewEngine("test", &stubModelModeAgent{}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "test:u", ReplyCtx: "ctx"}
	e.cmdUpgrade(p, msg, nil)
	e.cmdUpgradeConfirm(p, msg)
	if len(p.sent) != 2 || p.sent[0] != SelfUpdateDisabledMessage || p.sent[1] != SelfUpdateDisabledMessage {
		t.Fatal(p.sent)
	}
}
