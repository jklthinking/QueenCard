# QueenCard

**A Feishu-first fork of cc-connect that makes AI-agent chat cards clear, compact, and pleasant to read.**

[English](./README.md) | [中文](./README.zh-CN.md)

---

## Screenshots

> Placeholders — final screenshots will be added in a later release.
> All screenshots must be redacted (bot tokens, user IDs, chat names, file paths) before publishing.

- `docs/screenshots/card-overview.png` — a full agent reply card with identity header, panels, footer, and buttons *(placeholder)*
- `docs/screenshots/status-footer.png` — close-up of the footer: model · effort · elapsed / context bar / workdir *(placeholder)*
- `docs/screenshots/thinking-separated.png` — thinking kept out of the reply body *(placeholder)*

## Relationship to upstream

QueenCard is a fork of [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect), based on the **v1.5.0** release (commit `17c61062`). cc-connect bridges local AI coding agents (Claude Code, Codex, Gemini CLI, Kimi CLI, and more) to chat platforms such as Feishu/Lark, Telegram, Discord, and Slack. All credit for the bridge architecture, agent integrations, and platform support goes to the upstream author **chenhg5** and contributors.

cc-connect is licensed under the **MIT License** (as stated in the upstream README and npm package). QueenCard keeps the same license and preserves all upstream copyright notices. See [License](#license) and `NOTICE`.

QueenCard is an independent community fork. It is **not** affiliated with or endorsed by the cc-connect project. This tree is **pinned** to the v1.5.0 baseline and does **not** follow upstream auto-update.

## Features

QueenCard tracks upstream cc-connect v1.5.0 and changes what touches the Feishu card experience, agent usability from chat, and release safety. Everything upstream supports (agents, platforms, slash commands, scheduling) works as documented upstream.

### Implemented

- **Colored card header per bot** — `header_color`, `header_icon`, and `display_name` on `[projects.display]`. Running / done / stopped keep the bot color; errors always use red.
- **Three-line status footer on Feishu rich cards**
  1. model · reasoning effort · elapsed time
  2. context usage with an eight-segment progress bar and percentage
  3. current working directory
- **Thinking stays out of the answer** — reasoning from ACP-based agents (including `agent_thought_chunk`) is no longer mixed into the reply body.
- **Collapsible thinking and tool-call panels** — `card_show_thinking` / `card_show_tools` (default true).
- **Card action buttons** — switch model, switch effort, new conversation, stop (with confirm). Disable with `card_buttons = false`.
- **Optional quota / usage display** — off by default; opt in with `card_show_quota` plus `quota_command` or `quota_url`.
- **Codex model & effort controls** — six reasoning-effort levels and an up-to-date model list, switchable from chat.
- **Grok model & effort controls** — query and switch the active Grok model and its reasoning effort from chat.
- **Self-update disabled, version pinned** — builds are fixed to a known upstream tag and produced from source; see [Why no auto-update](#why-queencard-does-not-follow-upstream-auto-update).

### Planned

These are **not** in current builds:

- Richer quota detail views (the hook already returns `resets_at`, unused on the main card)
- Additional card actions beyond the four buttons above

## Installation

Prerequisites are the same as upstream: install and authenticate at least one agent CLI (Claude Code, Codex, Gemini CLI, …) before starting QueenCard. See the upstream README for agent setup.

The installed command name remains `cc-connect` so existing service units keep working. The product name is QueenCard.

### Build from source (recommended)

Requires Go 1.22+.

```bash
git clone https://github.com/jklthinking/QueenCard.git
cd QueenCard
make build          # or: go build -trimpath -o cc-connect ./cmd/cc-connect
# Linux amd64 pinned build (sha256 printed):
#   GO_BIN=go bash scripts/build-queencard.sh
```

Building from source is the recommended path: it matches how QueenCard releases are produced and guarantees you run exactly the code you reviewed.

### Prebuilt binaries

Download `cc-connect-linux-amd64` from the GitHub **Releases** page, verify the published sha256, then:

```bash
chmod +x cc-connect-linux-amd64
sudo mv cc-connect-linux-amd64 /usr/local/bin/cc-connect
cc-connect           # first run creates the config file
```

## For AI agents

If you are an AI assistant installing or upgrading QueenCard for a human operator, follow these steps in order. Do not skip the backup. Do not run `cc-connect update`.

1. **Backup the running program and the config.** Typical locations are `$(command -v cc-connect)` for the binary and `~/.cc-connect/config.toml` for config. Copy both with a dated suffix, for example `cc-connect.bak-YYYYMMDD` and `config.toml.bak-YYYYMMDD`. Record the original file mode and owner.
2. **Build or install.** Clone `https://github.com/jklthinking/QueenCard.git` and run `make build` or `bash scripts/build-queencard.sh`. Alternatively download the Release asset and check its sha256 against `SHA256SUMS`. Install by replacing the previous `cc-connect` binary (keep that filename if the service unit calls it).
3. **Configure identity** under each `[[projects]]` so bots are distinguishable:

   ```toml
   [projects.display]
   header_color = "blue"      # blue/wathet/indigo/turquoise/green/yellow/orange/violet/purple/grey/default
   header_icon = "🧭"         # optional emoji
   display_name = "Guide-A"   # empty defaults to the project name
   card_mode = "rich"         # required for the QueenCard footer and buttons
   ```

   See `docs/palette.md`. Do not use `red` or `carmine` as identity colors (reserved for errors).
4. **Do not run `cc-connect update`.** QueenCard disables self-update. An upstream installer would replace QueenCard with stock cc-connect and drop every card change.
5. **Restart the existing service unit only.** Do not start a second cc-connect process.
6. **What you should see in Feishu / Lark after a successful install:** a rich reply card with the configured icon and display name in a colored header; the answer body; collapsible thinking / tool panels when the agent emits them; a three-line footer (model · effort · elapsed, context bar, working directory); and four buttons on a finished turn (switch model, switch effort, new conversation, stop). Stop is available while a turn is running.
7. **Rollback:** copy the dated backup binary and `config.toml` back over the live files, restore mode/owner, and restart the same service unit.

## Configuration

QueenCard uses the same `config.toml` layout as upstream (`config.example.toml`). Only the options relevant to QueenCard's card features are listed here; for everything else, follow the upstream documentation.

Under `[[projects]]` → `[projects.display]`:

```toml
[projects.display]
card_mode = "rich"              # Feishu Card 2.0; required for the status footer
header_color = "blue"
header_icon = "🧭"
display_name = "Guide-A"
show_context_indicator = true   # context usage line + progress bar (default: true)
show_workdir_indicator = true   # working-directory line (default: true; false = two-line footer)
card_buttons = true             # four turn controls (default: true)
card_show_thinking = true
card_show_tools = true
card_show_quota = false         # optional external quota hook
```

Footer labels follow the global `language` setting through the existing i18n mechanism. Thinking separation, Codex/Grok controls, and disabled self-update are behavior changes and require no extra keys.

## Why QueenCard does not follow upstream auto-update

Upstream ships a `cc-connect update` command that downloads the latest release binary. QueenCard deliberately disables it:

1. **Pinned behavior.** Chat-facing behavior (card layout, command handling) must not change under a running deployment without a deliberate upgrade decision.
2. **Patch integrity.** A stock upstream binary would silently remove every QueenCard change. Self-updating would downgrade QueenCard into plain cc-connect.
3. **Reviewable upgrades.** Each QueenCard release rebases onto a specific upstream tag. Upgrading means: fetch the new QueenCard release (or rebuild from source), review the delta, and swap the binary on your own schedule.

## Roadmap

- Richer quota detail views
- Additional card actions
- Rebase onto new upstream stable tags as they are released
- Contribute generic bug fixes upstream where the maintainer is interested

## Acknowledgments

QueenCard is based on [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect) (MIT, author chenhg5). Thank you to the cc-connect project. Baseline: **v1.5.0 / 17c61062**. This fork is locked and does not follow upstream automatically. See `NOTICE`.

## License

MIT License — the same license as upstream cc-connect.

Copyright (c) chenhg5 and cc-connect contributors  
Copyright (c) 2026 JKL Thinking / jklthinking

Upstream copyright is retained; see `LICENSE` and `NOTICE`.
