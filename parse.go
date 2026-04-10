package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DocSection represents one section or subsection of doc_fretplot.tex.
type DocSection struct {
	URI     string
	Name    string
	Content string
}

// ScaleEntry is one row from the scale macros table in doc_fretplot.tex.
type ScaleEntry struct {
	macro   string
	formula string
	name    string
}

var (
	sectionRe    = regexp.MustCompile(`^\\section\{(.+)\}\s*$`)
	subsectionRe = regexp.MustCompile(`^\\subsection\{(.+)\}\s*$`)
	lstInputRe   = regexp.MustCompile(`\\lstinputlisting(?:\[[^\]]*\])?\{([^}]+)\}`)
	fpdocRe      = regexp.MustCompile(`\\fpdocexample\{([^}]+)\}`)
	inputRe      = regexp.MustCompile(`^\\input\{`)
	scaleMacroRe = regexp.MustCompile(`\\texttt\{\\textbackslash\s+(fp\w+)\}\s*&\s*([^&\\]+)&\s*([^\\]+?)\\\\`)
	layoutRe     = regexp.MustCompile(`^\\(newpage|normalsize|maketitle|tableofcontents|thispagestyle|captionsetup)\b`)
)

// sectionSlugMap maps the auto-derived slug of each top-level \section title
// to the short URI path component used in resource URIs.
var sectionSlugMap = map[string]string{
	"introduction":                         "introduction",
	"macros":                               "macros",
	"the-fretplot-file-format":             "fp",
	"the-fretplot-scale-style-file-format": "fps",
}

// latexTitleToSlug converts a LaTeX section/subsection title to a URI slug.
func latexTitleToSlug(s string) string {
	// \textbackslash WORD → WORD (strip the backslash, keep the macro name)
	s = regexp.MustCompile(`\\textbackslash\s+(\w+)`).ReplaceAllString(s, "$1")
	// \cmd{content} → content (repeat until no more matches)
	re := regexp.MustCompile(`\\[a-zA-Z@]+\{([^}]*)\}`)
	for re.MatchString(s) {
		s = re.ReplaceAllString(s, "$1")
	}
	// Remove remaining \cmd or \cmd\ sequences
	s = regexp.MustCompile(`\\[a-zA-Z@]+\\?\s*`).ReplaceAllString(s, "")
	// Slugify: lowercase, non-alphanumeric runs become hyphens
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// latexTitleToName converts a LaTeX title to a human-readable resource name.
func latexTitleToName(s string) string {
	// \textbackslash WORD → \WORD
	s = regexp.MustCompile(`\\textbackslash\s+(\w+)`).ReplaceAllString(s, `\$1`)
	// \cmd{content} → content
	re := regexp.MustCompile(`\\[a-zA-Z@]+\{([^}]*)\}`)
	for re.MatchString(s) {
		s = re.ReplaceAllString(s, "$1")
	}
	// Remove remaining \cmd variants
	s = regexp.MustCompile(`\\[a-zA-Z@]+\\?\s*`).ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func isLayoutLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return layoutRe.MatchString(trimmed) || strings.HasPrefix(trimmed, "%")
}

// resolveFileLine handles \lstinputlisting and \fpdocexample by inlining
// the referenced file content. Returns (content, true) if the line was
// resolved, or ("", true) if the line should be dropped, or (line, false)
// if the line is unchanged.
func resolveFileLine(line, docDir string) (string, bool) {
	trimmed := strings.TrimSpace(line)

	// Drop \input{} - these reference compiled TikZ output, not source
	if inputRe.MatchString(trimmed) {
		return "", true
	}

	// \fpdocexample{NAME} → inline src.fp and full.tex
	if m := fpdocRe.FindStringSubmatch(trimmed); m != nil {
		return fpdocContent(m[1], docDir), true
	}

	// \lstinputlisting[...]{FILE} → inline file content
	if m := lstInputRe.FindStringSubmatch(trimmed); m != nil {
		path := filepath.Join(docDir, m[1])
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Sprintf("[file not found: %s]\n", m[1]), true
		}
		return fmt.Sprintf("(%s)\n%s", m[1], string(content)), true
	}

	return line, false
}

// fpdocContent inlines the src.fp and full.tex files for a \fpdocexample call.
func fpdocContent(name, docDir string) string {
	fpPath := filepath.Join(docDir, "include", name, "src.fp")
	texPath := filepath.Join(docDir, "include", name, "full.tex")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("(proj/include/%s/src.fp)\n", name))
	if c, err := os.ReadFile(fpPath); err == nil {
		sb.Write(c)
	} else {
		sb.WriteString(fmt.Sprintf("[not found: include/%s/src.fp]\n", name))
	}
	sb.WriteString(fmt.Sprintf("\n(proj/%s.tex)\n", name))
	if c, err := os.ReadFile(texPath); err == nil {
		sb.Write(c)
	} else {
		sb.WriteString(fmt.Sprintf("[not found: include/%s/full.tex]\n", name))
	}
	sb.WriteString(fmt.Sprintf("\nCompile with: lualatex --shell-escape %s.tex\n", name))
	return sb.String()
}

// ParseDocSections parses doc_fretplot.tex and returns one DocSection per
// section/subsection. File references (\lstinputlisting, \fpdocexample) are
// resolved by inlining the content of the referenced files. Section-level
// introductory text (before the first subsection) is prepended to the first
// subsection's content.
func ParseDocSections(docPath string) ([]DocSection, error) {
	f, err := os.Open(docPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	docDir := filepath.Dir(docPath)

	var sections []DocSection
	var curURI, curName string
	var curLines []string
	var sectionLines []string // intro text at section level before first subsection
	var sectionSlug string
	atSectionLevel := false
	inDoc := false

	save := func() {
		if curURI == "" {
			return
		}
		content := strings.TrimSpace(strings.Join(curLines, "\n"))
		if content != "" {
			sections = append(sections, DocSection{URI: curURI, Name: curName, Content: content})
		}
		curLines = nil
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, `\begin{document}`) {
			inDoc = true
			continue
		}
		if !inDoc {
			continue
		}
		if strings.Contains(line, `\end{document}`) {
			break
		}

		trimmed := strings.TrimSpace(line)

		if m := sectionRe.FindStringSubmatch(trimmed); m != nil {
			save()
			slug := latexTitleToSlug(m[1])
			if mapped, ok := sectionSlugMap[slug]; ok {
				sectionSlug = mapped
			} else {
				sectionSlug = slug
			}
			curName = latexTitleToName(m[1])
			curURI = "fretplot://" + sectionSlug
			atSectionLevel = true
			sectionLines = nil
			continue
		}

		if m := subsectionRe.FindStringSubmatch(trimmed); m != nil {
			if atSectionLevel {
				// Stash any section-level intro; prepend it to the first subsection
				sectionLines = curLines
				curLines = nil
				atSectionLevel = false
			} else {
				save()
			}
			subSlug := latexTitleToSlug(m[1])
			curName = latexTitleToName(m[1])
			curURI = "fretplot://" + sectionSlug + "/" + subSlug
			if len(sectionLines) > 0 {
				curLines = append(sectionLines, curLines...)
				sectionLines = nil
			}
			continue
		}

		if isLayoutLine(trimmed) {
			continue
		}

		resolved, wasResolved := resolveFileLine(line, docDir)
		if wasResolved {
			if resolved != "" {
				curLines = append(curLines, resolved)
			}
		} else {
			curLines = append(curLines, line)
		}
	}
	save()

	return sections, scanner.Err()
}

// ParseScaleMacros parses the scale macros table from doc_fretplot.tex.
// It matches rows of the form:
//
//	\texttt{\textbackslash fpXXX} & formula & description\\
func ParseScaleMacros(docPath string) ([]ScaleEntry, error) {
	content, err := os.ReadFile(docPath)
	if err != nil {
		return nil, err
	}
	var entries []ScaleEntry
	for _, m := range scaleMacroRe.FindAllStringSubmatch(string(content), -1) {
		entries = append(entries, ScaleEntry{
			macro:   `\` + m[1],
			formula: strings.TrimSpace(m[2]),
			name:    strings.TrimSpace(m[3]),
		})
	}
	return entries, nil
}
