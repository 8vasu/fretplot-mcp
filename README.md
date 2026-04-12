# fretplot-mcp

An MCP (Model Context Protocol) server for the [fretplot](https://github.com/8vasu/fretplot) ([CTAN](https://ctan.org/pkg/fretplot)) LuaTeX package.

Allows MCP clients to generate fretplot and fretplot-specific LaTeX code from natural language descriptions of musical scales and chord voicings.

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

**Note:** Lua, LaTeX, or fretplot are not required.

### Build

```sh
$ git clone https://github.com/8vasu/fretplot-mcp
$ cd fretplot-mcp
$ go build .
```

## Usage in [Claude Code](https://www.anthropic.com/product/claude-code)

1. Connect to Claude Code:
```sh
$ claude mcp add --transport stdio --scope user fretplot -- /path/to/fretplot-mcp
```
2. Start a new claude Claude Code session:
```sh
$ claude
```

The next steps can proceed in 2 ways: in one you pick your own MCP prompt to force invocation of the correct MCP tool, and in the other you simply write the entire query in natural language and depend on the LLM to guess and pick an MCP tool for you.

### Method 1

3. In the Claude Code session, type `/fp` and select an MCP prompt from the menu using Up/Down arrow keys and then pressing Enter.
4. Type your query. Here are some examples of how your Claude Code prompt might look when you have finished typing your query:
```
/mcp__fretplot__fp rotate a diagram by 90 degrees clockwise
/mcp__fretplot__fps make C# a red triangle
/mcp__fretplot__fptex B natural minor scale diagram
```
5. Press Enter and wait for the MCP server and the LLM to do their magic!

### Method 2

3. Describe what you want at the Claude Code prompt in natural language. For example, `generate fretplot code to scale a diagarm by a factor of 2` or `generate fps code to render B flat as a green circle`.
4. Press Enter and wait for the LLM to pick an MCP tool. If it does not pick an MCP tool and instead tries to search for fretplot in the filesystem, please deny, type `use the MCP server`, and press Enter.
5. If the LLM has picked the right MCP tool, confirm its usage. Otherwise, deny and ask it to look for a different MCP tool.
