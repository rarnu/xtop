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

### 一句话安装

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | bash
```

该脚本会自动检测你的操作系统和架构，从 GitHub Releases 下载最新版本并安装 `xtop` 到 `/usr/local/bin`（如果没有写权限则安装到 `~/.local/bin`）。语言文件会安装到 `/etc/xtop/lang` 或 `~/.xtop/lang`。

安装指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | XTOP_VERSION=0.1.0 bash
```

安装到自定义目录：

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | XTOP_INSTALL_DIR=$HOME/bin bash
```

卸载：

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | bash -s -- --uninstall
```

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

## MCP 服务

`xtop` 可以作为 [Model Context Protocol](https://modelcontextprotocol.io/) 服务器运行，让 AI 助手查询实时系统监控数据。

启动服务：

```bash
xtop mcp
```

在 Claude Desktop、Cursor 或其他兼容 MCP 的客户端中配置：

```json
{
  "mcpServers": {
    "xtop": {
      "command": "xtop",
      "args": ["mcp"]
    }
  }
}
```

该服务为只读，暴露以下内容：

- **Resources**（`xtop://snapshot/*`）：`latest`、`cpu`、`mem`、`disk`、`gpu`、`net`、`proc`。
- **Tools**：`get_system_summary`、`get_cpu_info`、`get_memory_info`、`get_disk_info`、`get_gpu_info`、`get_network_info`、`get_process_info`。

所有工具均为查询，不执行任何副作用操作（不会结束、重启进程或写入文件）。

### SSE 模式

通过 Server-Sent Events 运行服务器，并自定义端口：

```bash
xtop mcp --transport sse --port 8080
```

默认端口为 `3001`。然后在 MCP 客户端中配置 SSE URL：

```json
{
  "mcpServers": {
    "xtop": {
      "url": "http://127.0.0.1:8080/sse"
    }
  }
}
```

## 命令行模式

当带有任何参数启动时，`xtop` 会以普通命令行模式执行，而不是启动 TUI。

```bash
./xtop [参数]
```

| 参数 | 说明 |
|------|------|
| `--help` | 显示帮助信息 |
| `--version` | 显示版本信息 |
| `--all` | 输出所有信息（CPU、内存、磁盘、GPU、网络、进程） |
| `--cpu` | 输出 CPU 使用情况 |
| `--mem` | 输出内存使用情况 |
| `--disk` | 输出磁盘使用情况 |
| `--gpu` | 输出 GPU 使用情况 |
| `--net` | 输出网络使用情况 |
| `--proc` | 输出当前进程情况 |
| `--json` | 以 JSON 格式输出（默认普通文本） |
| `--stream <x>` | 每隔 x 秒输出一次数据流（x 必须是大于等于 1 的整数） |

如果没有指定内容参数（如 `--cpu`、`--mem` 等），则默认启用 `--all`。例如 `./xtop --json` 等价于 `./xtop --all --json`。

常用组合：

```bash
./xtop --all --json --stream 5     # 输出所有信息，JSON 格式，每 5 秒一次
./xtop --cpu --mem                 # 输出 CPU 和内存，普通文本
./xtop --cpu --mem --json          # 输出 CPU 和内存，JSON 格式
./xtop --cpu --mem --json --stream 5
./xtop --cpu --mem --json --stream 5 --proc
./xtop --cpu --mem --proc --gpu
```

## 语言文件

翻译文件按以下优先级加载：

1. `/etc/xtop/lang/<locale>.json`
2. `~/.xtop/lang/<locale>.json`
3. `./lang/<locale>.json`

locale 从 `LC_ALL`、`LC_MESSAGES` 或 `LANG` 环境变量检测。若找不到对应语言文件，则回退到英文。

## 许可证

[MIT](./LICENSE)
