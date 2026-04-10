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

This server uses **Tools, Resources, and Prompts**.

### Why an MCP server for fretplot?

You could just open a Claude Code session, point it at the fretplot source, and ask it to generate diagrams.
That works. The MCP server adds value in specific ways:

- **Context window cost.** `doc_fretplot.tex` plus all referenced example files add up quickly.
  The MCP server pre-parses the documentation into structured resources, so only the relevant sections
  are loaded for any given task.
- **Targeted retrieval.** The snippet tools return only the documentation sections relevant to the
  query, keeping context lean regardless of how large the documentation grows.
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

- `doc_fretplot.tex` (checked out automatically in cone mode as a root file)
- `include/` directory (contains example `.fp` and `.tex` files referenced from the doc)

This is done with a partial clone and non-cone sparse checkout, which allows specifying
exact files and directories rather than the cone-mode default of always including all root-level files:

```
git clone --filter=blob:none --no-checkout <repo>
git sparse-checkout set --no-cone /doc_fretplot.tex /include/
git checkout
```

`fretplot.lua`, `fretplot.sty`, and all other files are never downloaded.

### fretplot storage location

The server clones fretplot into the OS-appropriate user data directory:

| OS      | Path                                                   |
|---------|--------------------------------------------------------|
| Linux   | `$XDG_DATA_HOME/fretplot-mcp/fretplot/` (default: `~/.local/share/fretplot-mcp/fretplot/`) |
| macOS   | `~/Library/Application Support/fretplot-mcp/fretplot/` |
| Windows | `%APPDATA%\fretplot-mcp\fretplot\`                     |

Note: Go's standard library has no `os.UserDataDir()` function (unlike `os.UserCacheDir` and `os.UserConfigDir`).
`userDataDir()` in `main.go` implements the correct platform logic manually using `runtime.GOOS`.

## Tools and resources

**Tools:**

| Tool | Input | Output |
|------|-------|--------|
| `list_scales` | `query` | All `\fp*` scale/arpeggio macros; query filters or focuses the answer |
| `fp_snippet` | `query` | Relevant `.fp` format documentation to answer the query |
| `fps_snippet` | `query` | Relevant `.fps` format documentation to answer the query |
| `tex_snippet` | `query` | Relevant LaTeX/macro documentation to answer the query |

**Resources:** one per subsection of `doc_fretplot.tex`, URI scheme `fretplot://<section>/<subsection>`.
The count tracks the documentation automatically; as of the initial release, 15 resources are registered.

**Prompts:** one per tool, each taking a `query` argument. Prompts guarantee the right tool is invoked
for the task rather than relying on the LLM to select it. Available in any MCP client, no per-user setup required.

| Prompt | Argument | Invokes |
|--------|----------|---------|
| `list_scales` | `query` | `list_scales` tool |
| `fp_snippet` | `query` | `fp_snippet` tool |
| `fps_snippet` | `query` | `fps_snippet` tool |
| `tex_snippet` | `query` | `tex_snippet` tool |

## Usage

### Prerequisites

- Go 1.13+
- Git

### Build

```bash
git clone https://github.com/8vasu/fretplot-mcp
cd fretplot-mcp
go get -u ./...
go mod tidy
go build .
```

### Connect to Claude Code

```bash
claude mcp add --transport stdio --scope user fretplot -- /path/to/fretplot-mcp
```

The `--scope user` flag registers the server globally (available in all Claude Code sessions).
Omitting it defaults to project scope - not what you want for a general-purpose tool.

Then start a new Claude Code session. The server will clone fretplot on first run, pull on subsequent runs.
To verify it works, try: "list all fretplot scales".

## Development log

### Stage 1 - Minimal working MCP server

**Goal:** end-to-end plumbing only. Claude Code connects, git sync works, one tool call round-trips.

1. Initialized Go module (`github.com/soumendra/fretplot-mcp`, Go 1.25.5).

2. Added the official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk v1.4.1`).
   Note: `go get` before `main.go` existed did not populate `go.sum` with transitive dependencies.
   Running `go get -u ./... && go mod tidy` after writing `main.go` completed it in one step.

3. Wrote `main.go` with:
   - `userDataDir()` - cross-platform data directory, implemented manually since `os.UserDataDir()`
     does not exist in Go's standard library.
   - `syncFretplot()` - clones fretplot on first run, `git pull`s on subsequent runs. Warns and
     continues if sync fails (server still starts).
   - `listScales` tool - parses the scale macro table in `doc_fretplot.tex` and returns a formatted
     table of all built-in scale/arpeggio macros with their names and interval formulas.

**First run output:**
```
2026/04/07 22:37:52 Cloning fretplot (sparse) into /home/user/.local/share/fretplot-mcp/fretplot
{"jsonrpc":"2.0","method":"notifications/tools/list_changed","params":{}}
```
The server blocks waiting for a client on stdin after startup. When run directly in the terminal
with no MCP client, this is expected - use Ctrl-C to exit.

---

### Stage 2 - Documentation resources

**Goal:** expose the full fretplot documentation as MCP resources, one per subsection.

**Key decision: documentation as the sole source of truth.**

An initial approach of hard-coding resource content as Go string constants was rejected as
unmaintainable: any change to the fretplot documentation would require a manual update to the
server. Instead, all resource content is parsed from `doc_fretplot.tex` at runtime. The server
treats the documentation the same way a human user would: learn everything from it, use nothing else.

This also drove the sparse checkout decision. `fretplot.lua` and `fretplot.sty` are never fetched
because the server never needs them.

**Files added/changed:**

- `doc.go` (new): LaTeX parser with `ParseDocSections()` and `ParseScaleMacros()`.
- `resources.go` (new): Registers one MCP resource per parsed subsection.
- `main.go`: Switched to sparse checkout; `listScales` updated to use `ParseScaleMacros()`.

**Parsing logic:**

`ParseDocSections()` walks `doc_fretplot.tex` line by line after `\begin{document}`:

- Each `\section{}` starts a new top-level context; its intro text is prepended to the first subsection.
- Each `\subsection{}` produces one `DocSection` with URI `fretplot://<section>/<subsection>`.
- `\lstinputlisting{FILE}` is replaced by the inlined content of `FILE`.
- `\fpdocexample{NAME}` is replaced by the inlined content of `include/NAME/src.fp` and
  `include/NAME/full.tex`, with a compile instruction appended.
- `\input{}` lines are dropped (compiled TikZ output, not source).
- Layout lines (`\newpage`, `\maketitle`, comments, etc.) are skipped.

Section-level slugs are mapped to short names: `introduction`, `macros`, `fp`, `fps`.

---

### Stage 3 - Snippet tools

**Goal:** give the LLM tools to answer targeted questions about fretplot syntax, returning only
the relevant documentation rather than burning the entire doc into context.

**Design principle:** the snippet tools do not generate code themselves. Each tool takes a `query`
string, returns the relevant documentation sections, and the LLM generates the correct snippet from
that context. A user asking "how do I rotate a diagram 90 degrees?" gets the exact `.fp` lines they
need - not a boilerplate file they didn't ask for.

The three tools map to the three file types in fretplot:

| Tool | Documentation sections returned |
|------|----------------------------------|
| `fp_snippet` | `.fp` file format reference and examples |
| `fps_snippet` | `.fps` file format reference and examples |
| `tex_snippet` | Introduction and macros reference (covers `\fpscale`, `\fpfret`, preamble, etc.) |

`tex_snippet` is a superset of any single-macro query - it handles everything from a one-line
`\fpscale` call to a complete compilable document.

**Files added/changed:**

- `tools.go` (new): `addTools()` registers the three snippet tools. `makeSnippetHandler()` closes
  over the pre-formatted documentation string for each tool.
- `resources.go`: signature changed from `addResources(server, docPath)` to
  `addResources(server, sections)` - no longer re-parses the file.
- `main.go`: `ParseDocSections()` is now called once and the result passed to both `addResources`
  and `addTools`.

---

### Stage 4 - MCP Prompts

**Goal:** add one MCP Prompt per tool so users can force the right tool to be invoked directly,
without relying on the LLM to select it. Prompts are part of the MCP spec and work in any MCP
client - not just Claude Code.

**Design:** each prompt takes a single `query` argument and returns a user message instructing the
LLM to call the corresponding tool. This is a thin routing layer: the prompt guarantees tool
selection, the tool returns the relevant documentation, the LLM generates the answer.

**Why prompts instead of Claude Code skills?** Skills (`.md` files in `~/.claude/commands/`) are
Claude Code-specific and require per-user setup. MCP prompts ship with the server and are
automatically available to any MCP client that connects.

**`list_scales` updated** to accept an optional `query` argument (consistent with the other three
tools), allowing targeted questions about specific scales or arpeggios rather than always returning
the full table.

**Files added/changed:**

- `prompts.go` (new): `addPrompts()` registers four prompts using `makePromptHandler()`, which
  closes over the tool name to produce the routing message.
- `main.go`: `addPrompts(server)` called after `addTools`; `list_scales` signature updated to
  accept `snippetInput`.
