package acp

import (
	"context"
	"github.com/chenhg5/cc-connect/core"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

var _ core.ModelSwitcher = (*grokACPAgent)(nil)
var _ core.ReasoningEffortSwitcher = (*grokACPAgent)(nil)

const grokModelsTTL = 5 * time.Minute
const grokModelsTimeout = 3 * time.Second

func parseGrokModels(output string) []core.ModelOption {
	var result []core.ModelOption
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || (fields[0] != "-" && fields[0] != "*") {
			continue
		}
		name := fields[1]
		if !strings.HasPrefix(name, "grok-") || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, core.ModelOption{Name: name})
	}
	return result
}
func (a *grokACPAgent) AvailableModels(ctx context.Context) []core.ModelOption {
	a.settingsMu.Lock()
	if time.Since(a.modelsAt) < grokModelsTTL && len(a.models) > 0 {
		models := append([]core.ModelOption(nil), a.models...)
		a.settingsMu.Unlock()
		return models
	}
	a.settingsMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, grokModelsTimeout)
	defer cancel()
	a.mu.RLock()
	cmd := a.cmd
	dir := a.workDir
	env := append([]string{}, a.extraEnv...)
	a.mu.RUnlock()
	c := exec.CommandContext(ctx, cmd, "models")
	c.Dir = dir
	c.Env = core.MergeEnv(os.Environ(), env)
	b, err := c.Output()
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	if err != nil {
		slog.Warn("grok: model discovery failed", "error", err)
		return append([]core.ModelOption(nil), a.models...)
	}
	models := parseGrokModels(string(b))
	if len(models) > 0 {
		a.models = models
		a.modelsAt = time.Now()
	}
	return append([]core.ModelOption(nil), a.models...)
}
func (a *grokACPAgent) SetModel(model string) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	a.model = strings.TrimSpace(model)
	a.updateLaunchArgs("--model", a.model)
}
func (a *grokACPAgent) GetModel() string {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.model
}
func (a *grokACPAgent) SetReasoningEffort(effort string) {
	effort = strings.ToLower(strings.TrimSpace(effort))
	if effort != "" && effort != "low" && effort != "medium" && effort != "high" && effort != "xhigh" {
		return
	}
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	a.effort = effort
	a.updateLaunchArgs("--reasoning-effort", effort)
}
func (a *grokACPAgent) GetReasoningEffort() string {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.effort
}
func (a *grokACPAgent) AvailableReasoningEfforts() []string {
	return []string{"low", "medium", "high", "xhigh"}
}

// Agent-level options must precede the stdio subcommand. Strip old aliases
// in both cmd and args to avoid duplicate flags or stale configured values.
func (a *grokACPAgent) updateLaunchArgs(flag, value string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	strip := func(args []string) []string {
		var out []string
		for i := 0; i < len(args); i++ {
			arg := args[i]
			key := strings.SplitN(arg, "=", 2)[0]
			match := key == flag || (flag == "--model" && key == "-m") || (flag == "--reasoning-effort" && key == "--effort")
			if match {
				if key == arg && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
				}
				continue
			}
			out = append(out, arg)
		}
		return out
	}
	a.cliExtraArgs = strip(a.cliExtraArgs)
	a.args = strip(a.args)
	all := append(append([]string{}, a.cliExtraArgs...), a.args...)
	index := 0
	for i, arg := range all {
		if arg == "agent" {
			index = i + 1
			break
		}
	}
	if value != "" {
		all = append(all[:index], append([]string{flag, value}, all[index:]...)...)
	}
	a.cliExtraArgs = nil
	a.args = all
}
func (a *grokACPAgent) WorkspaceAgentOptions() map[string]any {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	opts := a.Agent.WorkspaceAgentOptions()
	opts["model"] = a.model
	opts["reasoning_effort"] = a.effort
	return opts
}
