package main

import (
	"github.com/chenhg5/cc-connect/core"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestQueenCardExecutableReplacementRefusesWithoutTouchingFiles(t *testing.T) {
	dir := t.TempDir()
	target := dir + "/target"
	src := dir + "/src"
	for _, p := range []string{target, src} {
		if err := os.WriteFile(p, []byte(p), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := replaceExecutable(target, src); err == nil || err.Error() != core.SelfUpdateDisabledMessage {
		t.Fatal(err)
	}
	for _, p := range []string{target, src} {
		b, err := os.ReadFile(p)
		if err != nil || string(b) != p {
			t.Fatal(p, err)
		}
	}
}
func TestQueenCardUpdateCLIExitsWithRefusal(t *testing.T) {
	if os.Getenv("QUEENCARD_UPDATE_TEST_CHILD") == "1" {
		runUpdate()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestQueenCardUpdateCLIExitsWithRefusal$")
	cmd.Env = append(os.Environ(), "QUEENCARD_UPDATE_TEST_CHILD=1", "LC_ALL=C", "LANGUAGE=C")
	b, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(b), core.SelfUpdateDisabledMessage) {
		t.Fatal(string(b), err)
	}
}
