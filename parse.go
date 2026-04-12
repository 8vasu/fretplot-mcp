// fretplot-mcp - MCP server for the fretplot LuaTeX package.
// Copyright (C) 2026 Soumendra Ganguly
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Regexes for stripping and expanding LaTeX source.
var (
	lstInputRe = regexp.MustCompile(`\\lstinputlisting(?:\[[^\]]*\])?\{([^}]+)\}`)
	fpdocRe    = regexp.MustCompile(`\\fpdocexample\{([^}]+)\}`)
	inputRe    = regexp.MustCompile(`(?m)^[ \t]*\\input\{[^}]*\}[ \t]*\n?`)
	commentRe  = regexp.MustCompile(`(?m)^%.*\n?`)
	layoutRe   = regexp.MustCompile(`(?m)^[ \t]*\\(newpage|normalsize|maketitle|tableofcontents|thispagestyle|captionsetup)[^\n]*\n?`)
)

// Inlines file includes in docText.
func inlineFiles(docText string) string {
	// Drop \input{} lines (compiled TikZ output, not source).
	docText = inputRe.ReplaceAllString(docText, "")

	// Replace each \fpdocexample{<exampleDirName>} with the contents of certain files in include/<exampleDirName>/.
	// The names of these files are in fpdocexampleFileNames (defined in config.go).
	docText = fpdocRe.ReplaceAllStringFunc(docText, func(match string) string {
		m := fpdocRe.FindStringSubmatch(match)
		return fpdocContent(m[1])
	})

	// Replace each \lstinputlisting{file} with the file's contents.
	docText = lstInputRe.ReplaceAllStringFunc(docText, func(match string) string {
		m := lstInputRe.FindStringSubmatch(match)
		path := filepath.Join(fretplotRepoDirPath, m[1])
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Sprintf("[file not found: %s]\n", m[1])
		}
		return fmt.Sprintf("(%s)\n%s", m[1], string(content))
	})

	return docText
}

// Returns the concatenated contents of certain files in include/<exampleDirName>/.
// The names of these files are in fpdocexampleFileNames (defined in config.go).
func fpdocContent(exampleDirName string) string {
	var sb strings.Builder

	// For each file in include/<exampleDirName>/ with name in
	// fpdocexampleFileNames, append a block of the form:
	//
	//   (<relPath>)
	//   <file_contents>   (or "[not found: <relPath>]" if the file is missing)
	//
	// e.g. for exampleDirName="amaj" and filName="src.fp":
	//
	//   (include/amaj/src.fp)
	//   ...file_contents...
	for _, filName := range fpdocexampleFileNames {
		relPath := filepath.Join(fretplotDocIncludeDirName, exampleDirName, filName)
		filPath := filepath.Join(fretplotRepoDirPath, relPath)

		sb.WriteString(fmt.Sprintf("\n(%s)\n", relPath))
		if filBytes, err := os.ReadFile(filPath); err == nil {
			sb.Write(filBytes)
		} else {
			sb.WriteString(fmt.Sprintf("[not found: %s]\n", relPath))
		}
	}

	// Append a compile hint so the LLM knows how to build the example.
	sb.WriteString(fmt.Sprintf("\nCompile with: lualatex --shell-escape %s.tex\n", exampleDirName))

	return sb.String()
}

// Parses fretplot documentation and returns a map where keys are section titles and values are
// contents of the corresponding sections.
func ParseDocSections() (map[fretplotDocSectionTitle]fretplotDocSectionContent, error) {
	// Read the fretplot documentation source.
	docBytes, err := os.ReadFile(fretplotDocFilePath)
	if err != nil {
		return nil, err
	}
	docText := string(docBytes)

	// Discard everything after \end{document}.
	if i := strings.Index(docText, `\end{document}`); i >= 0 {
		docText = docText[:i]
	}

	// Strip comments and layout-only commands that add no semantic content.
	docText = commentRe.ReplaceAllString(docText, "")
	docText = layoutRe.ReplaceAllString(docText, "")

	// Inline all referenced example files.
	docText = inlineFiles(docText)

	// Split on \section boundaries and build the map to return.
	docSectionTable := make(map[fretplotDocSectionTitle]fretplotDocSectionContent)
	for _, part := range strings.Split(docText, "\n\\section")[1:] {
		// Get section title. The title is the text between the first { and } after \section.
		titleEnd := strings.Index(part, "}")
		if titleEnd < 0 {
			continue
		}
		title := fretplotDocSectionTitle(part[1:titleEnd])

		// Get section content.
		content := fretplotDocSectionContent(`\section{` + string(title) + "}\n\n" + strings.TrimSpace(part[titleEnd+1:]))

		// Map the title to the content.
		docSectionTable[title] = content
	}

	return docSectionTable, nil
}
