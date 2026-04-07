# fretplot-mcp

An MCP (Model Context Protocol) server for the [fretplot](https://github.com/8vasu/fretplot) LuaTeX package.
Allows Claude Code (and other MCP clients) to generate fretplot diagrams from plain-English music theory descriptions.

## What is MCP?

MCP (Model Context Protocol) is an open protocol that lets LLM clients (Claude Code, Claude Desktop, Cursor, etc.)
discover and call capabilities exposed by external servers. It is a standardized plugin system: write a server once,
any MCP-compatible client can use it.

The transport is JSON-RPC 2.0 over either stdio (process pipes) or HTTP with SSE. Claude Code spawns the server
as a subprocess and communicates over stdin/stdout - no networking required for local use.

### The three server primitives

| Primitive    | What it is                                                                 | Invoked by     |
|--------------|----------------------------------------------------------------------------|----------------|
| **Tool**     | A callable function with a name, description, and typed JSON input schema  | The LLM        |
| **Resource** | Readable data identified by a URI (files, docs, records)                   | The client/user|
| **Prompt**   | A named, parameterized message template                                     | The user       |

This server uses **Tools** only (for now).

### Why an MCP server for fretplot?

You could just open a Claude Code session, point it at the fretplot source, and ask it to generate diagrams.
That works. The MCP server adds value in specific ways:

- **Music theory computation belongs in code.** Calculating all fret positions for every note of a scale across
  6 strings is deterministic arithmetic. A server function does it perfectly every time; the LLM just describes
  *what* it wants.
- **Context window cost.** `fretplot.lua` (38 KB) + `fretplot.sty` (15 KB) + `doc_fretplot.tex` (26 KB) = ~80 KB
  burned per session just to establish domain knowledge. The MCP server encodes that knowledge in executable tools.
- **Validation.** The server can syntax-check generated `.fp` files and catch invalid parameter combinations before
  handing broken LaTeX back to the user.
- **Portability.** Works in any MCP client without copy-pasting the fretplot source each time.

## Architecture

- **Language:** Go
- **Transport:** stdio (spawned as a subprocess by Claude Code)
- **Lua runtime:** `texlua` (from TeX Live) - no embedded Lua VM
- **fretplot source:** cloned from GitHub into the OS data directory on each server startup

### Why texlua instead of an embedded Lua VM?

Three options were considered:

| Option | Lua runtime | fretplot source | Assessment |
|--------|-------------|-----------------|------------|
| **A: Embed Lua VM** | Bundled (e.g. gopher-lua) | Bundled | Lua version may not match LuaTeX; adds Go/Rust binding complexity |
| **B: texlua + TeX Live fretplot** | texlua from TeX Live | From TeX Live distro | TeX Live controls fretplot version; may lag CTAN |
| **C: texlua + git clone (chosen)** | texlua from TeX Live | Cloned from GitHub | texlua guarantees Lua compatibility; we control fretplot version |

`fretplot.lua` uses only one LuaTeX-specific call: `tex.sprint`, which streams output back into
the TeX engine. Everything else is pure Lua. Under option C, a small shim replaces `tex.sprint` with a
table accumulator before calling into `fretplot.lua`:

```lua
local chunks = {}
tex = { sprint = function(s) table.insert(chunks, s) end }
-- call fretplot functions ...
local output = table.concat(chunks)
```

### Why not bundle fretplot.lua via `go:embed`?

Bundling would require manually copying files from the fretplot project on every change. Since Claude Code
users have internet access, the server instead clones/pulls the fretplot repo from GitHub on each startup.
No manual syncing, no stale copies.

### Why is fretplot-mcp a separate repo from fretplot?

The fretplot project is dedicated to CTAN deployment. Keeping MCP tooling separate maintains that focus
and allows fretplot-mcp to be pinned independently on GitHub.

### fretplot storage location

The server clones fretplot into the OS-appropriate user data directory:

| OS      | Path                                                   |
|---------|--------------------------------------------------------|
| Linux   | `$XDG_DATA_HOME/fretplot-mcp/fretplot/` (default: `~/.local/share/fretplot-mcp/fretplot/`) |
| macOS   | `~/Library/Application Support/fretplot-mcp/fretplot/` |
| Windows | `%APPDATA%\fretplot-mcp\fretplot\`                     |

Note: Go's standard library has no `os.UserDataDir()` function (unlike `os.UserCacheDir` and `os.UserConfigDir`).
`userDataDir()` in `main.go` implements the correct platform logic manually using `runtime.GOOS`.

## Usage

### Prerequisites

- Go 1.13+
- Git
- TeX Live (for `texlua`) - required for Lua-based tools in Stage 2+

### Build

```bash
git clone https://github.com/8vasu/fretplot-mcp
cd fretplot-mcp
go get -u ./...
go mod tidy
go build .
./fretplot-mcp
```

### Connect to Claude Code

```bash
claude mcp add --transport stdio --scope user fretplot -- /path/to/fretplot-mcp
```

The `--scope user` flag registers the server globally (available in all Claude Code sessions).
Omitting it defaults to project scope, which creates a project metadata block in `~/.claude.json`
for the current directory - not what you want for a general-purpose tool.

Then start a new Claude Code session. The server will clone fretplot on first run, pull on subsequent runs.
To verify it works, try: "list all fretplot scales".

## Development log

### Stage 1 - Minimal working MCP server

**Goal:** end-to-end plumbing only. Claude Code connects, git sync works, one tool call round-trips.
No texlua calls, no `.fp` generation, no music theory computation yet.

**Steps:**

1. Created `fretplot-mcp/` directory.

2. Initialized Go module:
   ```
   go mod init github.com/soumendra/fretplot-mcp
   ```
   Go version resolved: 1.25.5.

3. Added the official MCP Go SDK:
   ```
   go get -u ./...
   go mod tidy
   ```
   SDK version resolved: `github.com/modelcontextprotocol/go-sdk v1.4.1`.

   Note: running `go get github.com/modelcontextprotocol/go-sdk` before `main.go` existed added the
   module to `go.mod` but did not populate `go.sum` with transitive dependencies, because Go only
   resolves checksums for packages that are actually imported by existing code. Once `main.go` was
   written, `go get -u ./... && go mod tidy` completed `go.sum` in one step.

4. Wrote `main.go` with:
   - `userDataDir()` - cross-platform data directory (Linux/macOS/Windows), implemented manually
     since `os.UserDataDir()` does not exist in Go's standard library.
   - `syncFretplot()` - clones fretplot from `https://github.com/8vasu/fretplot` on first run,
     `git pull`s on subsequent runs. Logs a warning and continues if sync fails (server still starts).
   - `parseScales()` - scans `fretplot.sty` with a regex matching zero-argument `\newcommand{\fp*}`
     definitions, pairing each with the `% comment` line immediately above it in the file.
   - `listScales` tool - calls `parseScales` and returns a formatted table of all built-in scale and
     arpeggio macros with their interval formulas (in semitones).
   - `main()` - syncs fretplot, registers the `list_scales` tool, runs over `mcp.StdioTransport{}`.

**Tools exposed in Stage 1:**

| Tool | Input | Output |
|------|-------|--------|
| `list_scales` | none | All `\fp*` scale/arpeggio macros from `fretplot.sty` with names and interval formulas |

**First run behavior:**

On first run the server clones fretplot into the OS data directory, logging progress to stderr.
It then emits a `notifications/tools/list_changed` JSON-RPC notification on stdout - this is the
MCP handshake announcing that tools are available. The server then blocks waiting for a client on
stdin. When run directly in the terminal with no MCP client, this hang is expected - use Ctrl-C to exit.
