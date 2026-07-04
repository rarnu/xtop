# xtop

[![CI](https://github.com/rarnu/xtop/actions/workflows/ci.yml/badge.svg)](https://github.com/rarnu/xtop/actions/workflows/ci.yml)

[English](./README.md)

`xtop` 是一款受 `top` 启发的终端系统监控工具，采用 Matrix 绿色主题。它以紧凑的 2x3 卡片仪表盘展示 CPU、内存、磁盘、GPU、网络和进程信息，支持鼠标操作和内置进程管理器。

![](screenshot/screenshot1.png)

## 功能

- **2x3 卡片仪表盘**：第一行是 CPU、磁盘、GPU；第二行是内存、网络、进程。
- **卡片自动缩放**：每个卡片根据终端大小自动调整；内容超出时显示垂直滚动条。
- **鼠标支持**：滚动卡片、拖拽滚动条、点击进程行、与弹窗交互。
- **进程管理器**：按 `P`/`Enter` 或点击进程卡片上的按钮打开全屏进程表。支持按 PID、用户、状态、CPU、内存、启动时间或命令排序，可直接结束或强制结束进程。
- **进程详情弹窗**：点击内存/网络/磁盘/GPU 卡片中的进程，查看详情并结束进程。
- **多语言**：根据系统 locale 自动从 `/etc/xtop/lang/`、`~/.xtop/lang/` 或 `./lang/` 加载语言文件。缺少翻译时自动回退到英文。
- **跨平台**：支持 Linux (amd64/arm64) 和 macOS (arm64/amd64) 构建。部分平台特性需要本地 CGO 构建才能完全可用。

## 安装

### 从源码构建

需要 Go 1.26 或更高版本。

```bash
git clone https://github.com/rarnu/xtop.git
cd xtop
go build ./cmd/xtop
```

### 交叉编译

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build ./cmd/xtop

# Linux arm64
GOOS=linux GOARCH=arm64 go build ./cmd/xtop

# MacOS arm64
GOOS=darwin GOARCH=arm64 go build ./cmd/xtop
```

## 使用

```bash
./xtop
```

| 按键 | 操作 |
|------|------|
| `q` / `Ctrl+C` | 退出 |
| `P` / `Enter` | 打开进程管理器 |
| `a` | 打开关于对话框 |
| `↑` / `↓` | 在进程管理器中选择进程 |
| `1-7` 或点击表头 | 排序进程表 |
| `k` | 结束选中的进程 |
| `K` / `f` | 强制结束选中的进程 |
| `ESC` | 关闭弹窗 / 返回 |

使用鼠标滚轮滚动卡片，拖拽滚动条。

## 语言文件

翻译文件按以下优先级加载：

1. `/etc/xtop/lang/<locale>.json`
2. `~/.xtop/lang/<locale>.json`
3. `./lang/<locale>.json`

locale 从 `LC_ALL`、`LC_MESSAGES` 或 `LANG` 环境变量检测。若找不到对应语言文件，则回退到英文。

## 许可证

[MIT](./LICENSE)
