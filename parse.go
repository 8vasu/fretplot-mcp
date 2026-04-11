package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	lstInputRe = regexp.MustCompile(`\\lstinputlisting(?:\[[^\]]*\])?\{([^}]+)\}`)
	fpdocRe    = regexp.MustCompile(`\\fpdocexample\{([^}]+)\}`)
	inputRe    = regexp.MustCompile(`(?m)^[ \t]*\\input\{[^}]*\}[ \t]*\n?`)
	commentRe  = regexp.MustCompile(`(?m)^%.*\n?`)
	layoutRe   = regexp.MustCompile(`(?m)^[ \t]*\\(newpage|normalsize|maketitle|tableofcontents|thispagestyle|captionsetup)[^\n]*\n?`)
)

func inlineFiles(docText, docDir string) string {
	docText = inputRe.ReplaceAllString(docText, "")

	docText = fpdocRe.ReplaceAllStringFunc(docText, func(match string) string {
		m := fpdocRe.FindStringSubmatch(match)
		return fpdocContent(m[1], docDir)
	})

	docText = lstInputRe.ReplaceAllStringFunc(docText, func(match string) string {
		m := lstInputRe.FindStringSubmatch(match)
		path := filepath.Join(docDir, m[1])
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Sprintf("[file not found: %s]\n", m[1])
		}
		return fmt.Sprintf("(%s)\n%s", m[1], string(content))
	})

	return docText
}

func fpdocContent(subdirName, docDir string) string {
	var sb strings.Builder
	for _, filName := range fpdocexampleFileNames {
		relPath := filepath.Join(fretplotDocIncludeDirName, subdirName, filName)
		filPath := filepath.Join(docDir, relPath)
		sb.WriteString(fmt.Sprintf("\n(%s)\n", relPath))
		if c, err := os.ReadFile(filPath); err == nil {
			sb.Write(c)
		} else {
			sb.WriteString(fmt.Sprintf("[not found: %s]\n", relPath))
		}
	}
	sb.WriteString(fmt.Sprintf("\nCompile with: lualatex --shell-escape %s.tex\n", subdirName))
	return sb.String()
}

func ParseDocSections() (map[string]string, error) {
	dataDir, err := userDataDir()
	if err != nil {
		return nil, err
	}
	docDir := filepath.Join(dataDir, fretplotMCPServerName, fretplotRepoName)

	if err := syncRepo(docDir, fretplotRepoURL, fretplotSparsePaths); err != nil {
		fmt.Printf("Warning: %s sync failed: %v\n", fretplotRepoName, err)
	}

	docPath := filepath.Join(docDir, fretplotDocFileName)
	docBytes, err := os.ReadFile(docPath)
	if err != nil {
		return nil, err
	}

	docText := string(docBytes)
	if i := strings.Index(docText, `\end{document}`); i >= 0 {
		docText = docText[:i]
	}

	docText = commentRe.ReplaceAllString(docText, "")
	docText = layoutRe.ReplaceAllString(docText, "")
	docText = inlineFiles(docText, docDir)

	doc := make(map[string]string)
	for _, part := range strings.Split(docText, "\n\\section")[1:] {
		titleEnd := strings.Index(part, "}")
		if titleEnd < 0 {
			continue
		}
		title := part[1:titleEnd]
		content := `\section{` + title + "}\n\n" + strings.TrimSpace(part[titleEnd+1:])
		doc[title] = content
	}
	return doc, nil
}
