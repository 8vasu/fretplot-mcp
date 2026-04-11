package main

import (
	"log"
	"path/filepath"
)

const fretplotMCPServerName = "fretplot-mcp"
const fretplotMCPServerVersion = "v0.1.0"
const fretplotRepoName = "fretplot"
const fretplotRepoURL = "https://github.com/8vasu/" + fretplotRepoName
const fretplotDocFileName = "doc_fretplot.tex"
const fretplotDocIncludeDirName = "include"

var fretplotSparsePaths = []string{
	"/" + fretplotDocFileName,
	"/" + fretplotDocIncludeDirName + "/",
}

var fpdocexampleFileNames = []string{
	"src.fp",
	"full.tex",
}

var (
	fretplotRepoDirPath string
	fretplotDocFilePath string
)

func init() {
	dataDir, err := userDataDir()
	if err != nil {
		log.Fatal(err)
	}
	fretplotRepoDirPath = filepath.Join(dataDir, fretplotMCPServerName, fretplotRepoName)
	fretplotDocFilePath = filepath.Join(fretplotRepoDirPath, fretplotDocFileName)
}

type toolConfig struct {
	description      string
	docSectionTitles []string
}

var tools = map[string]toolConfig{
	"fp_snippet": {
		"Generate .fp code for any fretplot diagram property or effect (rotation, string tuning, fret markers, capo, layout, etc.). Provide the query describing what you need; the tool returns the relevant .fp format documentation for you to generate the correct snippet.",
		[]string{"The fretplot file format"},
	},
	"fps_snippet": {
		"Generate .fps code for any fretplot scale style customization (note colors, shapes, labels, finger numbers, etc.). Provide the query describing what you need; the tool returns the relevant .fps format documentation for you to generate the correct snippet.",
		[]string{"The fretplot scale style file format"},
	},
	"tex_snippet": {
		"Generate fretplot LaTeX code for any macro usage: \\fpscale, \\fptotikz, \\fptemplate, document preamble setup, package options, TikZ integration, complete compilable documents, and built-in scale/arpeggio macros. Provide the query describing what you need; the tool returns the relevant documentation for you to generate the correct snippet.",
		[]string{"Introduction", `\LaTeX\ macros`},
	},
}
