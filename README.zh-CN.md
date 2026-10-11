# QueenCard

**cc-connect 的飞书优先分支：把 AI 智能体的聊天卡片做得好看、直观。**

[English](./README.md) | [中文](./README.zh-CN.md)

---

## 截图

> 占位说明——正式截图将在后续版本补入。
> 所有截图发布前必须打码（机器人令牌、用户 ID、群名、文件路径等敏感信息）。

- `docs/screenshots/card-overview.png` —— 带身份标题栏、面板、状态尾巴和按钮的完整回复卡片 *（占位）*
- `docs/screenshots/status-footer.png` —— 状态尾巴特写：模型·强度·耗时 / 上下文进度条 / 工作目录 *（占位）*
- `docs/screenshots/thinking-separated.png` —— 思考内容与正文分离的效果 *（占位）*

## 与上游的关系

QueenCard fork 自 [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect)，基于 **v1.5.0**（提交 `17c61062`）。cc-connect 把本地运行的 AI 编程智能体（Claude Code、Codex、Gemini CLI、Kimi CLI 等）接入飞书、Telegram、Discord、Slack 等聊天平台。桥接架构、各智能体接入和平台支持的全部功劳属于上游作者 **chenhg5** 与贡献者。

cc-connect 采用 **MIT 协议**（见上游 README 与 npm 声明）；QueenCard 沿用同一协议，并完整保留上游版权声明。见 [License](#license) 与 `NOTICE`。

QueenCard 是独立的社区 fork，与 cc-connect 项目无隶属关系、未获其背书。本仓库**锁定**在 v1.5.0 基线，**不跟随**上游自动更新。

## 特性

QueenCard 跟随上游 cc-connect v1.5.0，改动集中在飞书卡片体验、聊天内操控和发布安全。上游支持的其余能力（智能体、平台、斜杠命令、定时任务）均按上游文档正常使用。

### 已实现

- **按机器人着色的卡片标题栏**——在 `[projects.display]` 配置 `header_color`、`header_icon`、`display_name`。进行中 / 完成 / 停止保留项目色；出错一律 red。
- **飞书富卡片三行状态尾巴**
  1. 模型 · 推理强度 · 耗时
  2. 上下文用量，八段进度条加百分比
  3. 当前工作目录
- **思考不混入正文**——基于 ACP 的智能体（含 `agent_thought_chunk`）推理过程不再混进回复正文。
- **可折叠的思考与工具面板**——`card_show_thinking` / `card_show_tools`（默认开启）。
- **卡片按钮**——换模型、换强度、新对话、停止（停止带确认）。`card_buttons = false` 可关闭。
- **可选额度显示**——默认关闭；`card_show_quota` 加 `quota_command` 或 `quota_url`。
- **Codex 模型与强度控制**——六档推理强度、最新模型列表，均可在聊天中切换。
- **Grok 模型与强度控制**——可在聊天中查询和切换 Grok 的模型与推理强度。
- **禁用自更新、版本固定**——构建固定在已知上游 tag，从源码产出；见 [为什么不跟随上游自动更新](#为什么不跟随上游自动更新)。

### 计划中

以下内容**尚未**出现在当前构建中：

- 更完整的额度详情页（钩子已能返回 `resets_at`，主卡暂不展示）
- 四个按钮之外的更多卡片动作

## 安装

前置条件与上游一致：先安装并登录至少一个智能体 CLI（Claude Code、Codex、Gemini CLI 等），再启动 QueenCard。智能体安装方式见上游 README。

安装后的命令名仍是 `cc-connect`，以便沿用现有服务单元。产品名是 QueenCard。

### 从源码构建（推荐）

需要 Go 1.22+。

```bash
git clone https://github.com/jklthinking/QueenCard.git
cd QueenCard
make build          # 或：go build -trimpath -o cc-connect ./cmd/cc-connect
# Linux amd64 固定构建（打印 sha256）：
#   GO_BIN=go bash scripts/build-queencard.sh
```

推荐从源码构建：QueenCard 的发布版也是这样产出的，可以确保你运行的就是你审阅过的代码。

### 预编译二进制

从 GitHub **Releases** 下载 `cc-connect-linux-amd64`，按发布页的 sha256 校验，然后：

```bash
chmod +x cc-connect-linux-amd64
sudo mv cc-connect-linux-amd64 /usr/local/bin/cc-connect
cc-connect           # 首次运行自动生成配置文件
```

## 给 AI 助手的安装说明

如果你是 AI 助手，正在替操作者安装或升级 QueenCard，请按下面的顺序做。不要跳过备份。不要运行 `cc-connect update`。

1. **备份正在运行的程序和配置。** 程序常见路径是 `$(command -v cc-connect)`，配置常见路径是 `~/.cc-connect/config.toml`。两者都复制一份带日期后缀的备份，例如 `cc-connect.bak-YYYYMMDD` 和 `config.toml.bak-YYYYMMDD`。记下原文件的权限和属主。
2. **构建或安装。** 克隆 `https://github.com/jklthinking/QueenCard.git`，执行 `make build` 或 `bash scripts/build-queencard.sh`。也可以下载 Release 附件，用 `SHA256SUMS` 核对 sha256。安装时替换原来的 `cc-connect` 二进制（若服务单元调用的是这个名字，就保持这个文件名）。
3. **在 `[projects.display]` 配置身份**，让多个机器人一眼可辨：

   ```toml
   [projects.display]
   header_color = "blue"      # blue/wathet/indigo/turquoise/green/yellow/orange/violet/purple/grey/default
   header_icon = "🧭"         # 可选 emoji
   display_name = "Guide-A"   # 空则回退为项目名
   card_mode = "rich"         # 状态尾巴和按钮依赖此项
   ```

   详见 `docs/palette.md`。不要把 `red` / `carmine` 配成身份色（出错保留色）。
4. **不要运行 `cc-connect update`。** QueenCard 已禁用自更新。上游安装器会把 QueenCard 换成原版 cc-connect，卡片改动全部消失。
5. **只重启现有的服务单元。** 不要再拉起第二个 cc-connect 进程。
6. **装好后飞书里应看到：** 带配置图标和显示名的彩色标题栏；回复正文；智能体有输出时的可折叠思考/工具面板；三行状态尾巴（模型·强度·耗时、上下文进度条、工作目录）；完成回合上的四个按钮（换模型、换强度、新对话、停止）。进行中可点停止。
7. **回滚：** 把带日期的备份程序和 `config.toml` 拷回原位，恢复权限和属主，再重启同一个服务单元。

## 配置

QueenCard 使用与上游相同的 `config.toml` 结构（`config.example.toml`）。这里只列出与 QueenCard 卡片特性相关的配置项，其余配置请参照上游文档。

在 `[[projects]]` 下的 `[projects.display]` 段：

```toml
[projects.display]
card_mode = "rich"              # 飞书 Card 2.0；状态尾巴依赖此项
header_color = "blue"
header_icon = "🧭"
display_name = "Guide-A"
show_context_indicator = true   # 上下文用量行与进度条（默认 true）
show_workdir_indicator = true   # 工作目录行（默认 true；设为 false 则尾巴为两行）
card_buttons = true             # 四个回合按钮（默认 true）
card_show_thinking = true
card_show_tools = true
card_show_quota = false         # 可选外部额度钩子
```

状态尾巴的文案跟随全局 `language` 设置，走现有的 i18n 机制。思考分离、Codex/Grok 控制、禁用自更新属于行为变更，无需额外配置项。

## 为什么不跟随上游自动更新

上游提供 `cc-connect update` 命令，会下载最新发布版二进制。QueenCard 有意禁用了它：

1. **行为固定。** 面向聊天的行为（卡片布局、命令处理）不应在运行中的部署上被悄悄改变，升级必须是一次明确的决定。
2. **补丁完整性。** 换成上游原版二进制会静默抹掉 QueenCard 的全部改动——自更新等于把 QueenCard 降级成原版 cc-connect。
3. **可审阅的升级。** 每个 QueenCard 版本都 rebase 到指定的上游 tag。升级路径是：获取新的 QueenCard 发布版（或从源码重新构建）、审阅差异、按自己的节奏替换二进制。

## Roadmap

- 更完整的额度详情页
- 更多卡片动作
- 随上游新的稳定 tag 发布做 rebase
- 在上游维护者有兴趣的方向上回馈通用修复

## 致谢

QueenCard 基于 [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect)（MIT，作者 chenhg5）。感谢 cc-connect 项目。基线版本：**v1.5.0 / 17c61062**。本项目锁定，不跟随上游自动更新。详见 `NOTICE`。

## License

MIT License——与上游 cc-connect 相同。

Copyright (c) chenhg5 and cc-connect contributors  
Copyright (c) 2026 JKL Thinking / jklthinking

保留上游版权声明，详见仓库内 `LICENSE` 与 `NOTICE`。
