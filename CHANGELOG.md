# Changelog

本文记录 `terminalparser` 对外可观察的变化。日期使用 `YYYY-MM-DD`。

## Unreleased - 2026-07-11

### Added

- 新增 `New` 与 `Option` API，以及 `WithSize`、`WithMaxScrollback`、`WithTrim`、
  `WithUnwrap` 配置项。
- 新增带错误返回的一次性解析 API：`Parse` 和 `ParseString`。
- 新增 `CursorRow` 和 `Size`，分别用于低分配读取光标物理行及当前终端尺寸。
- 新增 `Reset`，并为关闭后的操作统一返回 `ErrClosed`。
- 新增包文档、安装说明、一次性/流式使用示例和公开 API 行为约定。
- 新增 VT 控制序列、标题、光标、备选屏幕、并发访问、关闭和兼容入口测试。

### Changed

- 对照 `dev` 分支移除了手写 VTE 状态机的实现方式，统一由当前 libghostty-vt
  解析 ANSI/VT 字节流。不再维护 `MongoShParser`、`TmuxParser`、`USqlParser`、
  `WindowsParser` 等针对调用场景的重复解析器。
- libghostty 从 `b203652ca87e` 升级并固定到 2026-07-10 的
  `102a50836ce6`（Go 模块版本
  `v0.0.0-20260710165742-102a50836ce6`）。
- `TerminalVT` 现在封装并串行化所有底层终端访问；`Close` 改为幂等。
- `NewTerminalVT` 保留为兼容构造函数，但默认启用 1000 行滚动历史；新代码推荐
  `New(WithSize(...))`。
- `ParseOutput` 保留 `dev` 的无错误返回签名和“最多 500 个非空行”用途；解析精度由
  libghostty 提供。需要错误处理或自定义尺寸时应迁移到 `Parse`。
- `_test` SSH 集成程序已从直接操作旧版 libghostty 迁移为消费本模块的公开 API，
  并为终端注册表补充并发安全和连接清理。
- 光标坐标明确采用 libghostty 的零起点语义。`dev` 中部分解析器使用一作为起点，
  迁移时需检查坐标比较逻辑。

### Removed

- 移除 `TerminalVT.VT` 公开字段以及 `Lock`/`Unlock`。调用者不再能绕过并发和生命
  周期保护直接操作非线程安全的 libghostty 句柄。
- 不再公开 `dev` 的内部屏幕模型和辅助对象，包括 `Screen`、`Row`、`Cursor`、
  `TerminalParser`、`TerminalScreen`、`TRow`、`TRingRowBuffer`、`OutPutScreen` 以及
  ASCII/CSI 辅助表。这些类型从未形成稳定、完整的终端语义；外部调用应迁移到
  `TerminalVT`、`Parse` 或 `ParseString`。

### Build

- 最低 Go 版本为 1.26。
- 通过 cgo 链接 libghostty-vt。`go get` 可直接取得模块，但最终构建仍必须按上游要求
  提供 libghostty-vt 和 `pkg-config`；此限制来自 libghostty Go 绑定。
