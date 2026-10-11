package acp

import (
	"context"
	"github.com/chenhg5/cc-connect/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQueenCardGrokModelsParsing(t *testing.T) {
	got := parseGrokModels("You are logged in.\nDefault model: grok-4.6\nAvailable models:\n - grok-4.7\n * grok-4.6 (default)\n - grok-4.7\n - other\n")
	if len(got) != 2 || got[0].Name != "grok-4.7" || got[1].Name != "grok-4.6" {
		t.Fatal(got)
	}
}
func TestQueenCardOnlyGrokHasSwitchers(t *testing.T) {
	for _, cmd := range []string{"other", "/bin/grok", "grok.exe"} {
		a := wrapIfGrok(&Agent{cmd: cmd})
		_, models := a.(core.ModelSwitcher)
		_, efforts := a.(core.ReasoningEffortSwitcher)
		if models != isGrokCLI(cmd) || efforts != isGrokCLI(cmd) {
			t.Fatal(cmd)
		}
	}
}
func TestQueenCardGrokDiscoveryCacheAndTimeout(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "grok")
	count := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nprintf x >> '" + count + "'\nprintf 'Available models:\\n  - grok-4.7\\n  * grok-4.6 (default)\\n'\n"
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	a := &grokACPAgent{Agent: &Agent{cmd: bin, workDir: dir}}
	got := a.AvailableModels(context.Background())
	if len(got) != 2 {
		t.Fatal(got)
	}
	got[0].Name = "mutated"
	if got = a.AvailableModels(context.Background()); got[0].Name != "grok-4.7" {
		t.Fatal(got)
	}
	b, err := os.ReadFile(count)
	if err != nil || string(b) != "x" {
		t.Fatal(string(b), err)
	}
	a.modelsAt = time.Time{}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexec sleep 10\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if len(a.AvailableModels(ctx)) != 2 || time.Since(start) > time.Second {
		t.Fatal("timeout/stale cache")
	}
}
func TestQueenCardGrokSwitchLaunchAndWorkspace(t *testing.T) {
	a := &grokACPAgent{Agent: &Agent{cmd: "grok", cliExtraArgs: []string{"agent", "-m", "old"}, args: []string{"--effort=low", "stdio"}}}
	a.SetModel("grok-4.7")
	a.SetReasoningEffort("HIGH")
	a.SetModel("grok-4.6")
	want := "agent --model grok-4.6 --reasoning-effort high stdio"
	if got := strings.Join(a.args, " "); got != want {
		t.Fatal(got, want)
	}
	if a.GetModel() != "grok-4.6" || a.GetReasoningEffort() != "high" {
		t.Fatal("getters")
	}
	a.SetReasoningEffort("invalid")
	if a.GetReasoningEffort() != "high" {
		t.Fatal("invalid changed effort")
	}
	opts := a.WorkspaceAgentOptions()
	if opts["model"] != "grok-4.6" || opts["reasoning_effort"] != "high" {
		t.Fatal(opts)
	}
	if strings.Join(a.AvailableReasoningEfforts(), ",") != "low,medium,high,xhigh" {
		t.Fatal("efforts")
	}
	a.SetReasoningEffort("")
	if strings.Contains(strings.Join(a.args, " "), "reasoning-effort") {
		t.Fatal(a.args)
	}
}

// Use a local ACP peer to verify two successive subprocesses receive the new
// flags, without invoking Grok, authentication, or any external service.
func TestQueenCardGrokNextACPProcessUsesSwitchedSettings(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "grok")
	record := filepath.Join(dir, "argv")
	script := `#!/usr/bin/python3
import sys,json
with open(sys.argv[sys.argv.index('--record')+1],'a') as f: f.write(' '.join(sys.argv[1:])+'\n')
for line in sys.stdin:
 p=json.loads(line)
 if 'id' not in p: continue
 result={}
 if p['method']=='initialize': result={'protocolVersion':1,'agentCapabilities':{}}
 if p['method']=='session/new': result={'sessionId':'local-test'}
 print(json.dumps({'jsonrpc':'2.0','id':p['id'],'result':result}),flush=True)
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	a := &grokACPAgent{Agent: &Agent{cmd: bin, workDir: dir, args: []string{"agent", "stdio", "--record", record}}}
	for _, tc := range []struct{ model, effort string }{{"grok-4.6", "low"}, {"grok-4.7", "xhigh"}} {
		a.SetModel(tc.model)
		a.SetReasoningEffort(tc.effort)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		s, err := a.StartSession(ctx, "")
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		cancel()
	}
	b, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "--model grok-4.6") || !strings.Contains(lines[1], "--model grok-4.7") || !strings.Contains(lines[1], "--reasoning-effort xhigh") {
		t.Fatal(string(b))
	}
}
func TestQueenCardGrokConfiguredLaunchFlagsReported(t *testing.T) {
	a := wrapIfGrok(&Agent{cmd: "grok", cliExtraArgs: []string{"agent", "--model=grok-4.7"}, args: []string{"--effort", "high", "stdio"}}).(*grokACPAgent)
	if a.GetModel() != "grok-4.7" || a.GetReasoningEffort() != "high" {
		t.Fatal("configured flags not reported")
	}
}

func TestQueenCardGrokActualModelsOutput(t *testing.T) {
	const output = "You are logged in with grok.com.\n\nDefault model: grok-4.6\n\nAvailable models:\n  - grok-4.7\n  - grok-4.7-build-fast\n  * grok-4.6 (default)\n  - grok-4.5\n"
	got := parseGrokModels(output)
	if len(got) != 4 || got[2].Name != "grok-4.6" || got[3].Name != "grok-4.5" {
		t.Fatal(got)
	}
}
func TestQueenCardGrokFlagBoundaries(t *testing.T) {
	a := wrapIfGrok(&Agent{cmd: "grok", args: []string{"--model", "--effort", "HIGH", "stdio"}}).(*grokACPAgent)
	if a.GetModel() != "" || a.GetReasoningEffort() != "high" {
		t.Fatal(a.model, a.effort)
	}
	a.SetModel("grok-4.6")
	if !strings.Contains(strings.Join(a.args, " "), "stdio") {
		t.Fatal(a.args)
	}
}
func TestQueenCardGrokDiscoveryDoesNotBlockSettings(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "grok")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexec sleep 10\n"), 0700); err != nil {
		t.Fatal(err)
	}
	a := &grokACPAgent{Agent: &Agent{cmd: bin}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { a.AvailableModels(ctx); close(done) }()
	time.Sleep(30 * time.Millisecond)
	settings := make(chan struct{})
	go func() { a.SetModel("grok-4.6"); close(settings) }()
	select {
	case <-settings:
	case <-time.After(200 * time.Millisecond):
		t.Error("discovery held settings lock")
	}
	cancel()
	<-done
}
