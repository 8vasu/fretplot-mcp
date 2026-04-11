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

This server uses **Tools and Prompts**.

### Why an MCP server for fretplot?

You could just open a Claude Code session, point it at the fretplot source, and ask it to generate diagrams.
That works. The MCP server adds value in specific ways:

- **Context window cost.** `doc_fretplot.tex` plus all referenced example files add up quickly.
  The snippet tools return only the documentation sections relevant to the query, keeping context
  lean regardless of how large the documentation grows.
- **Portability.** Works in any MCP client without copy-pasting the fretplot docs each time.

## Architecture

- **Language:** Go
- **Transport:** stdio (spawned as a subprocess by Claude Code)
- **Knowledge source:** `doc_fretplot.tex` only - the server learns everything from the documentation,
  exactly as a human user would learn from reading the PDF

### Documentation as the single source of truth

The server does not read `fretplot.lua` or `fretplot.sty`. All knowledge is derived from `doc_fretplot.tex`
at runtime - the same document that compiles to `doc_fretplot.pdf`. This means:

- Any improvement to the documentation automatically improves the MCP server.
- No manual syncing of implementation details into the server.
- The server's knowledge is always consistent with what a user reading the PDF would know.

### Sparse checkout

Rather than cloning the entire fretplot repository, the server fetches only what is needed:

- `doc_fretplot.tex` (root file)
- `include/` directory (example `.fp` and `.tex` files referenced from the doc)

This is done with a partial clone and non-cone sparse checkout, which allows specifying
exact files and directories rather than the cone-mode default of always including all root-level files:

```
git clone --filter=blob:none --no-checkout <repo>
git sparse-checkout set --no-cone /doc_fretplot.tex /include/
git checkout
```

`fretplot.lua`, `fretplot.sty`, and all other files are never downloaded.

On first run the server clones fretplot automatically; on subsequent runs it pulls updates.

### fretplot storage location

The server clones fretplot into the OS-appropriate user data directory:

| OS      | Path                                                   |
|---------|--------------------------------------------------------|
| Linux   | `$XDG_DATA_HOME/fretplot-mcp/fretplot/` (default: `~/.local/share/fretplot-mcp/fretplot/`) |
| macOS   | `~/Library/Application Support/fretplot-mcp/fretplot/` |
| Windows | `%APPDATA%\fretplot-mcp\fretplot\`                     |

Note: Go's standard library has no `os.UserDataDir()` function (unlike `os.UserCacheDir` and `os.UserConfigDir`).
`userDataDir()` in `platform.go` implements the correct platform logic manually using `runtime.GOOS`.

## Tools and prompts

**Tools** - each takes a `query` string and returns the relevant documentation section(s) for the LLM to generate the answer:

| Tool | Documentation used | Covers |
|------|--------------------|--------|
| `fp_snippet` | The fretplot file format | `.fp` syntax, parameters, examples |
| `fps_snippet` | The fretplot scale style file format | `.fps` syntax, style customization |
| `tex_snippet` | Introduction + LaTeX macros | `\fpscale`, `\fptotikz`, preamble, built-in scale/arpeggio macros |

`tex_snippet` handles scale macro lookup - the built-in scale/arpeggio table is part of the
LaTeX macros section of the documentation.

The tools do not generate code themselves. Each tool returns the relevant documentation and the
LLM generates the correct snippet from that context.

**Prompts** - one per tool, each taking a `query` argument. Prompts guarantee the right tool is
invoked rather than relying on the LLM to select it. Available in any MCP client, no per-user setup required.

| Prompt | Invokes |
|--------|---------|
| `fp_snippet` | `fp_snippet` tool |
| `fps_snippet` | `fps_snippet` tool |
| `tex_snippet` | `tex_snippet` tool |

## Usage

### Prerequisites

- Go 1.13+
- Git

### Build

```bash
git clone https://github.com/8vasu/fretplot-mcp
cd fretplot-mcp
go build .
```

### Connect to Claude Code

```bash
claude mcp add --transport stdio --scope user fretplot -- /path/to/fretplot-mcp
```

The `--scope user` flag registers the server globally (available in all Claude Code sessions).
Omitting it defaults to project scope - not what you want for a general-purpose tool.

Then start a new Claude Code session. The server will clone fretplot on first run, pull on subsequent runs.

## Implementation notes

### Startup sequence

`main()` runs in order:

1. Clones or pulls the fretplot repo (sparse checkout) via `syncRepo`.
2. Creates the MCP server.
3. Calls `addTools`, which calls `ParseDocSections()` to read and parse the documentation.
4. Registers prompts, then starts serving over stdio.

### Parsing

`ParseDocSections()` in `parse.go` parses the already-synced documentation:

1. Reads `doc_fretplot.tex` and strips everything after `\end{document}`.
2. Removes comments and layout commands (`\newpage`, `\maketitle`, etc.).
3. Inlines file references: `\lstinputlisting{FILE}` is replaced by the file's content;
   `\fpdocexample{NAME}` is replaced by `include/NAME/src.fp` and `include/NAME/full.tex`;
   `\input{}` (compiled TikZ output) is dropped.
4. Splits on `\n\section` - the raw section title becomes the map key, the body the value.

Returns a `map[string]string` keyed by exact LaTeX section titles as they appear in the source.

### Configuration

`config.go` is the single place containing all static configuration: repo URL, filenames,
directory names, sparse checkout paths, and tool/prompt definitions (`tools` map). Adding a new
tool means adding one entry to the `tools` map - no other files need changing.

### go.mod dependency fetch

Dependencies are fetched automatically on first build. No manual `go get` required:

```bash
go build .  # fetches dependencies, compiles
```
