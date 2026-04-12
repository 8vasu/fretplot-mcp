# fretplot-mcp

An MCP (Model Context Protocol) server for the [fretplot](https://github.com/8vasu/fretplot) ([CTAN](https://ctan.org/pkg/fretplot)) $Lua\TeX$ package.

Allows MCP clients to generate fretplot and fretplot-specific $\LaTeX$ code from natural language descriptions of musical scales and chord voicings.

## MCP tools and prompts

Each tool takes a `query` string.

| Tool    | Purpose                                                                                    |
|---------|--------------------------------------------------------------------------------------------|
| `fp`    | `.fp` syntax, parameters, examples                                                         |
| `fps`   | `.fps` syntax, style customization                                                         |
| `fptex` | `\fptotikz`, `\fptemplate`, `\fpstemplate`, `\fpscale`, and built-in scale/arpeggio macros |

Each tool has a prompt of the same name as the tool associated with it, and each prompt takes a `query` string as argument to pass to the corresponding tool.

## Installation

### Prerequisites

- [Go](https://go.dev/)
- [Git](https://git-scm.com/)

**Note:** Lua, $\LaTeX$, or fretplot are not required.

### Build

```sh
$ git clone https://github.com/8vasu/fretplot-mcp
$ cd fretplot-mcp
$ go build .
```

### Connect to Claude Code

```sh
claude mcp add --transport stdio --scope user fretplot -- /path/to/fretplot-mcp
```
