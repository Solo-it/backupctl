// Package mdterm renders markdown as readable ANSI text for console
// --info-style commands. The idea: README.md stays the single source of
// documentation (renders nicely on GitHub), and through go:embed +
// mdterm.Render the very same file becomes the built-in help — there's
// nothing extra to keep in sync.
//
// Not full CommonMark — headings, `code`, **bold**, links, code blocks,
// and tables: exactly what actually shows up in a CLI tool's README.
//
// Typical use in a binary:
//
//	//go:embed README.md
//	var readmeText string
//
//	func printInfo() {
//		fmt.Print(mdterm.Render(readmeText, mdterm.SupportsColor()))
//	}
package mdterm

import (
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
)

// Render turns markdown into text for the terminal. color enables ANSI codes.
func Render(md string, color bool) string {
	lines := strings.Split(md, "\n")
	var out strings.Builder

	var tableBuf []string
	flushTable := func() {
		if len(tableBuf) == 0 {
			return
		}
		out.WriteString(renderTable(tableBuf, color))
		tableBuf = nil
	}

	inCodeBlock := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")

		if strings.HasPrefix(strings.TrimSpace(trimmed), "```") {
			flushTable()
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			out.WriteString(sgr(color, dim, "  "+trimmed))
			out.WriteString("\n")
			continue
		}

		if strings.HasPrefix(trimmed, "|") {
			tableBuf = append(tableBuf, trimmed)
			continue
		}
		flushTable()

		switch {
		case strings.HasPrefix(trimmed, "### "):
			text := strings.TrimPrefix(trimmed, "### ")
			out.WriteString(sgr(color, bold+yellow, text))
			out.WriteString("\n")
		case strings.HasPrefix(trimmed, "## "):
			text := strings.TrimPrefix(trimmed, "## ")
			out.WriteString(sgr(color, bold+cyan, text))
			out.WriteString("\n")
		case strings.HasPrefix(trimmed, "# "):
			text := strings.TrimPrefix(trimmed, "# ")
			out.WriteString(sgr(color, bold+magenta, text))
			out.WriteString("\n")
			out.WriteString(sgr(color, magenta, strings.Repeat("═", displayWidth(text))))
			out.WriteString("\n")
		case strings.HasPrefix(trimmed, "> "):
			out.WriteString(sgr(color, dim, strings.TrimPrefix(trimmed, "> ")))
			out.WriteString("\n")
		default:
			out.WriteString(renderInline(trimmed, color))
			out.WriteString("\n")
		}
	}
	flushTable()

	return out.String()
}

const (
	reset   = "\x1b[0m"
	bold    = "\x1b[1m"
	dim     = "\x1b[2m"
	cyan    = "\x1b[36m"
	yellow  = "\x1b[33m"
	magenta = "\x1b[35m"
)

func sgr(color bool, codes, text string) string {
	if !color {
		return text
	}
	return codes + text + reset
}

var (
	boldRe = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	codeRe = regexp.MustCompile("`([^`]+)`")
	linkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func renderInline(s string, color bool) string {
	s = linkRe.ReplaceAllString(s, "$1 ($2)")
	s = codeRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := codeRe.FindStringSubmatch(m)[1]
		return sgr(color, cyan, inner)
	})
	s = boldRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := boldRe.FindStringSubmatch(m)[1]
		return sgr(color, bold, inner)
	})
	return s
}

// renderTable aligns a markdown table via tabwriter and bolds the header
// row. Skips the second row (the ---|--- separator).
func renderTable(rows []string, color bool) string {
	var w strings.Builder
	tw := tabwriter.NewWriter(&w, 0, 2, 2, ' ', 0)

	for i, row := range rows {
		if i == 1 && isTableSeparator(row) {
			continue
		}
		cells := splitTableRow(row)
		for j, cell := range cells {
			if j > 0 {
				tw.Write([]byte("\t"))
			}
			rendered := renderInline(cell, color)
			if i == 0 {
				rendered = sgr(color, bold, rendered)
			}
			tw.Write([]byte(rendered))
		}
		tw.Write([]byte("\n"))
	}
	tw.Flush()
	return w.String()
}

func isTableSeparator(row string) bool {
	for _, r := range row {
		if r != '|' && r != '-' && r != ' ' && r != ':' {
			return false
		}
	}
	return true
}

func splitTableRow(row string) []string {
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	const escapedPipe = "\x00"
	row = strings.ReplaceAll(row, `\|`, escapedPipe)
	parts := strings.Split(row, "|")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(strings.TrimSpace(p), escapedPipe, "|")
	}
	return parts
}

func displayWidth(s string) int {
	return len([]rune(s))
}

// SupportsColor reports whether stdout is a terminal and NO_COLOR isn't
// explicitly set (see no-color.org). Check this on the caller's side
// before Render — mdterm itself doesn't decide anything about os.Stdout,
// since the output might well be redirected to a file.
func SupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
