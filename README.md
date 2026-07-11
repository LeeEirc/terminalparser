# terminalparser

`terminalparser` 是一个面向命令输出的 Go ANSI/VT 解析库。它使用 Ghostty 的
`libghostty-vt` 维护真实终端状态，可正确处理光标移动、擦除、颜色、标题、主/备选
屏幕等控制序列，并将最终屏幕导出为纯文本。

与 `dev` 分支中的手写解析器相比，本实现不再按 shell 或操作系统区分解析器；SSH、
PTY、本地进程和 websocket 收到的字节流都使用同一个 `TerminalVT`。

## 安装

```bash
go get github.com/LeeEirc/terminalparser@latest
```

本包通过 cgo 使用 `go.mitchellh.com/libghostty`。`go get` 会取得全部 Go 依赖；编译
前还需要安装 `libghostty-vt`，并让 `pkg-config` 能找到它：

```bash
# 在 ghostty 源码目录中构建并安装到独立目录
zig build -Demit-lib-vt -Doptimize=ReleaseFast --prefix /opt/libghostty-vt

export PKG_CONFIG_PATH=/opt/libghostty-vt/share/pkgconfig
go test ./...
```

默认静态链接。动态链接时使用 `-tags dynamic`，并同时配置运行时动态库搜索路径。
libghostty-vt 的构建与交叉编译要求见
[go-libghostty README](https://pkg.go.dev/go.mitchellh.com/libghostty#readme-usage)。
当前版本要求 Go 1.26、cgo、C 工具链以及 libghostty-vt；不支持
`CGO_ENABLED=0`。

## 一次性解析

适用于已经收集完整的命令输出：

```go
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/LeeEirc/terminalparser"
)

func main() {
	rows, err := terminalparser.Parse(
		[]byte("\x1b[32mbuild ok\x1b[0m\r\n"),
		terminalparser.WithSize(120, 40),
		terminalparser.WithMaxScrollback(5000),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(strings.Join(rows, "\n"))
}
```

如果只需要一个字符串，使用 `ParseString`。旧 `dev` API 的 `ParseOutput` 仍然可用；
它固定返回最多 500 个去除首尾空白后的非空行，并且因兼容旧签名而无法返回错误，
新代码应优先使用 `Parse`。

## 流式解析

适用于 SSH、PTY 或进程管道分段到达的数据：

```go
terminal, err := terminalparser.New(
	terminalparser.WithSize(80, 24),
	terminalparser.WithMaxScrollback(2000),
)
if err != nil {
	return err
}
defer terminal.Close()

// 每收到一段数据就写入；TerminalVT 可由多个 goroutine 安全调用。
if _, err := terminal.Write(chunk); err != nil {
	return err
}

text, err := terminal.String()
rows, err := terminal.ScreenRows()
title, err := terminal.Title()
x, err := terminal.CursorX()
y, err := terminal.CursorY()
alternate, err := terminal.IsScreenAlternate()
```

`String`/`ScreenRows` 是调用时刻的快照。`WithTrim` 控制行尾空格，`WithUnwrap`
控制是否合并由终端宽度造成的软换行。所有读取、写入、缩放、重置和关闭操作均由库
内部串行化；`Close` 可重复调用，关闭后的操作返回 `ErrClosed`。

## API 约定

- 坐标均为零起点，与当前 libghostty API 一致。
- `Resize` 的前两个参数是字符列/行数，后两个参数是单个字符单元的像素宽/高；像素
  信息未知时可传零。
- 默认尺寸为 `80x24`，默认保留 1000 行滚动历史，默认裁掉非空行的行尾空格。
- `TerminalVT` 不应被值复制；请始终保存并传递指针。
- 本库只解析终端的输出字节流，不启动命令、不管理 PTY，也不推断 shell prompt 或
  命令退出状态。

## 版本策略

libghostty 的 Go 绑定目前没有 v1 稳定承诺。本模块固定经过测试的 libghostty 伪版本，
并只暴露较小的包装 API；升级上游依赖时会在 `CHANGELOG.md` 中记录行为或 API 变化。
