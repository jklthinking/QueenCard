# QueenCard palette / QueenCard 配色

Choose identity in `[projects.display]`: `header_color`, `header_icon` (optional emoji),
`display_name` (defaults to project name), and `card_buttons` (default `true`).
Global `[display]` values provide defaults; project values override them.

在 `[projects.display]` 选择机器人身份：`header_color`、`header_icon`（可空 emoji）、
`display_name`（默认为项目名）、`card_buttons`（默认 `true`）。项目配置覆盖全局 `[display]`。

The header color represents the robot. Running, completed and stopped cards retain
its color; errors always use `red`. State appears in the subtitle.
标题栏颜色代表机器人，进行中、完成、停止均保留项目色；出错一律 `red`，状态写在副标题。

| Color / 颜色 | Suggested use / 建议用途 |
| --- | --- |
| blue | Navigation, general assistants / 导航、通用助手 |
| wathet | Gentle assistance / 轻量陪伴 |
| indigo | Analysis / 分析 |
| turquoise | Coordination / 协作 |
| green | Engineering / 工程 |
| yellow | Teaching / 教学 |
| orange | Creative work / 创意 |
| violet | Research / 研究 |
| purple | Specialist work / 专项 |
| red | Reserved for errors / 出错保留色 |
| carmine | Reserved for errors / 出错保留色 |
| grey | Neutral assistants / 中性助手 |

`default` is also accepted and uses the Feishu default theme. Invalid colors,
`red`, and `carmine` in identity configuration produce a warning and fall back to
`default`. Color names are trimmed and normalized to lowercase.
也可使用 `default`，采用飞书默认主题。非法颜色及配置中的 `red`、`carmine` 会记录警告并
回退 `default`。颜色名去除首尾空格并转为小写。保留色不能作为机器人常态配色。

## Multiple robots / 多机器人示例

| Robot / 机器人 | header_icon | display_name | header_color |
| --- | --- | --- | --- |
| 🧭 Guide-A | 🧭 | Guide-A | blue |
| 🐰 Builder-B | 🐰 | Builder-B | green |
| 🐻 Research-C | 🐻 | Research-C | violet |
| 🐈‍⬛ Studio-D | 🐈‍⬛ | Studio-D | orange |
| 🎓 Tutor-E | 🎓 | Tutor-E | yellow |

Keep display names short for mobile clients. / 手机端建议采用简短显示名。

## Optional quota hook / 可选额度钩子

Disabled by default. Set `card_show_quota = true`, and either `quota_command`
(a shell command) or `quota_url` (HTTP GET). If both exist, command takes priority.
默认关闭。开启 `card_show_quota = true`，配置 `quota_command`（shell 命令）或
`quota_url`（HTTP GET）；同时配置时命令优先。

Return a single JSON object: `{"used_percent":63,"label":"本周额度","resets_at":"2026-10-12T00:00:00+08:00"}`.
`used_percent` is required, 0–100, and represents used quota. `label` is optional,
overrides `quota_label`, and otherwise defaults to the localized weekly quota label.
`resets_at` is optional and reserved for later detail views.
返回单个 JSON 对象；`used_percent` 必填，0–100，表示已用百分比；`label` 可选，覆盖
`quota_label`，再回退到本地化的“本周额度”；`resets_at` 可选，主卡暂不显示。

Hooks run only for terminal metadata, with a two-second timeout and 8 KiB output
limit. Results and failures are cached per project for `quota_cache_seconds`
(default 300); failures hide the quota and log a warning once per cache interval.
钩子仅在终态读取，超时两秒、输出上限 8 KiB。结果和失败均按项目缓存，
`quota_cache_seconds` 默认 300 秒；失败隐藏额度，每个缓存周期记录一次警告。

Optional `card_avatar_img_key` accepts an existing Feishu image key and omits the
header image when empty. `card_show_thinking` and `card_show_tools` default to
`true` and independently hide the corresponding panels without changing quiet
mode's accumulation of steps.
可选 `card_avatar_img_key` 使用已有飞书图片 key，空值不显示头像。
`card_show_thinking`、`card_show_tools` 默认均为 `true`，独立控制面板显示；
不改变 quiet 模式保留步骤的行为。
