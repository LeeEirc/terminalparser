# 终端输出解析器设计与使用说明

本文档说明本仓库的终端输出解析能力、核心数据结构与扩展点，便于维护与二次开发。

## 目标与能力范围

- 将终端数据流解析为“行”并维护光标位置，尽可能贴近常见 TTY 行为。
- 处理常用的 C0 控制字符、CSI/OSC 序列，支持基本的擦除、插入、移动等操作。
- 以固定容量环形缓冲区存放行，避免内存无序增长。
- 为 tmux、Windows、usql、mongosh 等特定场景提供基于 go-vte 的解析屏幕实现。

参考：`./_test/ansicode.txt`（控制序列样例）。

## 快速开始

示例：最小化解析输出为行文本

```go
s := terminalparser.NewScreen(60, 80) // 行数、列数（目前主要用于初始化，逻辑上不强依赖）
s.Feed([]byte("hello\r\nworld\n"))
rows := s.GetRows() // []*Row
for _, r := range rows {
    fmt.Println(r.String())
}
```

或直接一次性解析：

```go
out := s.Parse([]byte(stream)) // []string
```

调试日志：设置环境变量 `TERMINALPARSER=1` 将开启 `Printf/Println` 的调试输出。

## 架构总览

核心文件与职责：

- `screen.go`：自研解析器入口。维护输入缓冲、逐字节解析、C0/CSI/OSC 分派、行与光标更新。
- `row.go`：单行数据结构 `Row` 与固定容量环形缓冲 `RingRowBuffer` 的实现。
- `csi_func.go`：CSI 指令分派表（A/B/C/D/E/F/G/H/J/K/P/X/d/h/l/m 等）。
- `cursor.go`：光标 `Cursor` 的基本移动与边界处理。
- `ascii_table.go`：ASCII 与控制字符常量，含 C0、参数区、字母区等集合。
- `output.go`：提供 `ParseOutput` 和简化的输出屏幕实现（基于 go-vte）。
- 终端/协议适配：`tmux_screen.go`、`windows_screen.go`、`usql_screen.go`、`mongosh_screen.go`（均基于 go-vte）。
- 日志：`log.go`（由环境变量控制是否输出调试日志）。

### 数据流简述

1. `Screen.Feed([]byte)` 写入缓冲区并触发 `TryParse()`。
2. `parse` 逐 rune 解码，区分：
   - ESC 开头的序列：进一步分派到 CSI/OSC/Intermediate 处理。
   - C0 控制字符：如 BEL、BS、CR、LF 等，更新光标或行内容。
   - 可打印字符：写入当前行，考虑字符显示宽度（CJK 等宽度由 runewidth 计算）。
3. `Rows` 以环形缓冲保存最近 N 行（默认 500）。

## 核心类型与行为

### Screen

关键字段：

- `Rows *RingRowBuffer`：行环形缓冲区（默认容量 500，见 `NewScreen`）。
- `Cursor *Cursor`：光标（X 列、Y 行，外部语义以 1 为起点）。
- `buffer bytes.Buffer`：输入缓存，尽量保证只在序列完整时解析。
- `ColLens`、`RowLens`：初始列宽与行数（当前实现主要作为参考）。

关键方法：

- `Feed(p []byte)`/`TryParse()`：写缓冲并解析。
- `Parse(p []byte) []string`：便捷方法，解析后直接返回当前所有行的字符串切片。
- 擦除相关：`eraseRight`、`eraseLeft`、`eraseAbove`、`eraseBelow`、`eraseAll`、`eraseFromCursor`。
- 字符插入与删除：`appendCharacter`、`deleteChars(ps)`。
- 光标行获取：`GetCursorRow()`（若不存在则创建）。

控制序列处理：

- C0：见 `parseC0Sequence`。典型行为：
  - BS(0x08) 后退一列；CR(0x0d) 回车将 X 置 0；LF(0x0a) 下一行并可能推进环形缓冲。
- CSI：见 `csi_func.go` 的 `CSIFuncMap`。已覆盖常见的光标移动（A/B/C/D/E/F/G/H）、
  擦除（J/K）、删除/插入字符（P/@/X）、设置行（d）等；部分序列标注为未实现时打印调试信息。
- OSC：目前仅做基本终止符（BEL/ST）识别，其他暂未深入处理。

### Row（单行）

字段要点：

- `dataRune []rune`：行内容。
- `currentX`、`currentRuneIndex`：通过 `runewidth` 维护光标到 rune 索引的映射（多宽字符友好）。
- `tipRune`/`tipRecord`：为 fish 等补全提示保留的临时尾部片段（`String()` 会去掉 tipRune）。

常用操作：

- 追加字符：`appendCharacter(code)`（会推进光标与索引，记录补全片段）。
- 插入字符：`insertCharacters(data []rune)`。
- 删除/擦除：`deleteChars(ps)`、`eraseRight()`。
- 光标对齐：`changeCursorToX(x)` 内部用 `runewidth` 重新计算 `currentRuneIndex`。

多宽字符处理：

- 通过 `github.com/mattn/go-runewidth` 计算显示宽度，确保 East Asian 宽字符与组合字符位置正确。

### RingRowBuffer（环形行缓冲）

设计动机：避免 `[]*Row` 无上限增长导致的内存压力与 GC 抖动。

实现要点：

- 基于 `container/ring`，固定容量 `size`；写满后覆盖最早的行。
- `Values()` 返回当前有效行（当 `full=true` 时从“最旧”到“最新”顺序）。
- `Append(*Row)`、`Last()`、`EraseAll()`、`EraseAbove(idx)`、`EraseBelow(idx)` 提供常用操作。

复杂度与容量：

- 追加与获取最后一行为 O(1)，遍历为 O(n)；
- 默认容量为 500（见 `NewScreen`），如需调整可修改 `NewScreen` 或对外暴露配置（建议后续改进）。

### Cursor（光标）

- `X,Y` 为 1 基坐标；`MoveUp/Down/Left/Right` 做基本边界保护（`Left` 不小于 1，`Up` 不小于 0）。
- 解析 CR/LF 等控制符会影响 `Cursor` 与 `Rows` 的推进；注意内部某些地方将 X 暂置为 0，随后通过行操作对齐。

## 已有的协议/终端适配

为部分复杂场景（tmux、Windows、usql、mongosh）提供了基于 `github.com/danielgatis/go-vte` 的屏幕实现：

- `tmux_screen.go`、`windows_screen.go`、`usql_screen.go`、`mongosh_screen.go`
- 这些文件实现了 VTE 的回调接口（Print/Execute/CsiDispatch 等），将解析结果写入 `TmuxRow`/`VTRow` 行结构。
- 与自研 `Screen` 并行存在，按业务场景择用。

同时提供 `output.go` 中的 `ParseOutput`，适合“只要干净文本输出”的简化场景。

## 输入/输出解析器示例（实验性）

`_test/parser.go` 中的 `TerminalParser` 展示了如何：

- 维护输入/输出状态机（InputState/OutputState）。
- 自动识别 PS1（提示符）与命令边界，在命令完成后截取其输出。
- 结合 go-vte 屏幕实现处理不同终端行为。

该示例更贴近实际交互式终端的采集需求，可作为上层产品的参考实现。

## 环形缓冲区迁移说明（现状与约束）

- 需求背景：早期 `rows` 使用动态切片，长会话下内存增长不可控；现已迁移为 `RingRowBuffer`。
- 现状：`NewScreen` 默认创建 `NewRingRowBuffer(500)`。写满后覆盖最早行；`Values()` 始终返回有序切片用于展示/比对。
- 兼容性：
  - 依赖 `Rows` 长度无限的逻辑需调整为“只看最近 N 行”。
  - 光标行 `GetCursorRow()` 调用在环形满额时仍指向“最后一行”（最新行），行为不变。
- 可能的改进：
  - 将容量对外可配置（构造参数或可选项）。
  - 补充 `Snapshot()`/`Iterator` 以减少 `Values()` 的临时切片分配。

## 边界与兼容性注意事项

- 多宽字符：中日韩字符、表情符号、组合字符的宽度计算依赖 `runewidth`，光标 X 以显示宽度为准。
- 控制字符：Delete(0x7F) 恒忽略；BEL(0x07) 仅提示；CR/LF 组合按常见终端行为移动。
- 擦除策略：`CSI J/K` 的不同参数对应“清屏/清行/从光标到右侧/到左侧”等，未覆盖的分支会打印调试信息。
- 粘贴模式：`?2004h/l` 仅设置 `pasteMode` 标志，当前未做特殊处理。

## 构建与运行

前置：Go 1.23+（见 `go.mod`）。

- 仅库构建：无额外步骤（文档改动不影响构建）。
- 运行测试：仓库包含 `screen_test.go` 与 `_test` 子模块，注意其中个别测试读取外部文件（如 `output_windows.txt`），可能需要自行准备或跳过。

建议命令（可选）：

```bash
# 根模块快速测试（可能因外部依赖文件缺失而失败）
go test ./...

# 仅运行根模块下的测试
go test

# 在调试时开启日志
TERMINALPARSER=1 go test -v
```

前端 UI（如需）：`_test/ui` 为 Vite + Vue 示例工程，用于可视化；不影响核心库。

## 路线图（Roadmap）

- 对外暴露 `RingRowBuffer` 容量配置；
- 完善更多 CSI/OSC 序列兼容与差异化行为；
- 为 `Screen` 增加快照/迭代器接口，减少一次性 `Values()` 带来的内存拷贝；
- 增强多宽字符/组合字符的边界测试与回归测试；
- 提供更统一的“命令-输出”抽取上层 API（类似 `_test/parser.go`）。

## 待确认的问题

请帮助确认以下事项，以便完善实现与文档：

1. 环形缓冲区默认容量 500 是否满足生产场景？是否需要在 `NewScreen` 对外开放可配置参数？
2. 目前自研 `Screen` 与 go-vte 屏幕（tmux/windows/usql/mongosh）在项目中各自适用的业务边界是什么？是否需要统一抽象？
3. 是否需要对 bracketed paste 模式（?2004h/l）做特殊处理（例如禁用某些 CSI 解析）？
4. 是否有特定的 PS1 格式/终端类型需要重点兼容（如自定义颜色、超长提示符、多行提示符）？
5. 测试中引用的外部样例文件（如 `output_windows.txt`）是否可以纳入仓库或用生成样例代替，以保证 CI 稳定？


