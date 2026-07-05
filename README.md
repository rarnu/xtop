# xtop

[![CI](https://github.com/rarnu/xtop/actions/workflows/ci.yml/badge.svg)](https://github.com/rarnu/xtop/actions/workflows/ci.yml)

[中文](./README_zh_CN.md)

`xtop` is a Matrix-green terminal system monitor inspired by `top`. It displays CPU, memory, disk, GPU, network and process information in a compact 2x3 card dashboard, with mouse support and a built-in process manager.

![](screenshot/screenshot1.png)

## Features

- **2x3 card dashboard**: CPU, Disk, GPU on the first row; Memory, Network, Process on the second row.
- **Auto-resizing cards**: each card scales with the terminal size; vertical scrollbars appear when content overflows.
- **Mouse support**: scroll cards, drag scrollbars, click process rows, and interact with dialogs.
- **Process manager**: press `P`/`Enter` or click the button on the process card to open a full-screen process table. Sort by PID, user, status, CPU, memory, start time or command. Kill or force-kill processes directly.
- **Process detail popup**: click a process in the Memory/Network/Disk/GPU cards to view details and terminate it.
- **Multi-language**: automatically loads language files from `~/.xtop/lang/` based on system locale. Falls back to English when a translation is missing.
- **Cross-platform**: builds on Linux (amd64/arm64) and macOS (arm64/amd64). Some platform-specific features require native CGO builds for full functionality.

## Installation

### One-liner install

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | bash
```

This detects your OS/architecture, downloads the latest release from GitHub and installs `xtop` to `/usr/local/bin` (or `~/.local/bin` if you do not have write access). Language files are installed to `~/.xtop/lang`.

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | XTOP_VERSION=0.1.0 bash
```

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | XTOP_INSTALL_DIR=$HOME/bin bash
```

Uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | bash -s -- --uninstall
```

### Build from source

Requires Go 1.26 or later.

```bash
git clone https://github.com/rarnu/xtop.git
cd xtop
go build ./cmd/xtop
```

### Cross-compile

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build ./cmd/xtop

# Linux arm64
GOOS=linux GOARCH=arm64 go build ./cmd/xtop

# MacOS arm64
GOOS=darwin GOARCH=arm64 go build ./cmd/xtop
```

## Usage

```bash
./xtop
```

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `P` / `Enter` | Open process manager |
| `a` | Open about dialog |
| `↑` / `↓` | Select process in manager |
| `1-7` or click header | Sort process table |
| `k` | Kill selected process |
| `K` / `f` | Force-kill selected process |
| `ESC` | Close dialog / back |

Use the mouse wheel to scroll cards and drag scrollbars.

## MCP Server

`xtop` can run as a [Model Context Protocol](https://modelcontextprotocol.io/) server so AI assistants can query live system monitoring data.

Start the server:

```bash
xtop mcp
```

Configure Claude Desktop, Cursor or any MCP-compatible client:

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

The server is read-only and exposes:

- **Resources** under `xtop://snapshot/*`: `latest`, `cpu`, `mem`, `disk`, `gpu`, `net`, `proc`.
- **Tools**: `get_system_summary`, `get_cpu_info`, `get_memory_info`, `get_disk_info`, `get_gpu_info`, `get_network_info`, `get_process_info`.

No tool performs side effects (no kill, no restart, no write).

### SSE mode

Run the server over Server-Sent Events and choose a custom port:

```bash
xtop mcp --transport sse --port 8080
```

The default port is `3001`. Then configure your MCP client with the SSE URL:

```json
{
  "mcpServers": {
    "xtop": {
      "url": "http://127.0.0.1:8080/sse"
    }
  }
}
```

## Command-line mode

When started with any flag, `xtop` runs in plain command-line mode instead of launching the TUI.

```bash
./xtop [flags]
```

| Flag | Description |
|------|-------------|
| `--help` | Show help information |
| `--version` | Show version information |
| `--all` | Output all information (CPU, memory, disk, GPU, network, processes) |
| `--cpu` | Output CPU usage |
| `--mem` | Output memory usage |
| `--disk` | Output disk usage |
| `--gpu` | Output GPU usage |
| `--net` | Output network usage |
| `--proc` | Output current process information |
| `--json` | Output in JSON format (default is plain text) |
| `--stream <x>` | Stream output every `x` seconds (`x` must be an integer ≥ 1) |

If no content flag (`--cpu`, `--mem`, etc.) is specified, `--all` is assumed. For example, `./xtop --json` is equivalent to `./xtop --all --json`.

Examples:

```bash
./xtop --all --json --stream 5     # all info, JSON, every 5 seconds
./xtop --cpu --mem                 # CPU and memory, plain text
./xtop --cpu --mem --json          # CPU and memory, JSON
./xtop --cpu --mem --json --stream 5
./xtop --cpu --mem --json --stream 5 --proc
./xtop --cpu --mem --proc --gpu
```

## Language files

Translation files are loaded from `~/.xtop/lang/<locale>.json`.

The locale is detected from `LC_ALL`, `LC_MESSAGES` or `LANG`. If a file is not found, English is used as fallback.

## License

[MIT](./LICENSE)
