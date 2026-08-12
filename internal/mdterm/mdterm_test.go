package mdterm

import (
	"strings"
	"testing"
)

func TestRender_Headers_NoColor(t *testing.T) {
	got := Render("# Title\n\n## Section\n\n### Sub\n", false)
	for _, want := range []string{"Title", "Section", "Sub", "═"} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
}

func TestRender_Headers_Color(t *testing.T) {
	got := Render("# Title\n", true)
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI codes with color=true:\n%q", got)
	}
	if !strings.Contains(got, "Title") {
		t.Errorf("heading text was lost:\n%q", got)
	}
}

func TestRender_NoColor_StripsANSI(t *testing.T) {
	got := Render("# Title\n**bold** `code`\n", false)
	if strings.Contains(got, "\x1b[") {
		t.Errorf("expected no ANSI codes with color=false:\n%q", got)
	}
	if !strings.Contains(got, "bold") || !strings.Contains(got, "code") {
		t.Errorf("text was lost:\n%q", got)
	}
}

func TestRenderInline_CodeAndBold(t *testing.T) {
	got := renderInline("text **important** and `code`", false)
	if strings.Contains(got, "*") || strings.Contains(got, "`") {
		t.Errorf("markdown markup should not remain: %q", got)
	}
	if !strings.Contains(got, "important") || !strings.Contains(got, "code") {
		t.Errorf("text was lost: %q", got)
	}
}

func TestRenderInline_Link(t *testing.T) {
	got := renderInline("see [restic](https://restic.net/)", false)
	if !strings.Contains(got, "restic") || !strings.Contains(got, "https://restic.net/") {
		t.Errorf("link was not rendered: %q", got)
	}
}

func TestRender_CodeBlock(t *testing.T) {
	md := "text before\n```bash\nbackupctl --info\n```\ntext after\n"
	got := Render(md, false)
	if !strings.Contains(got, "backupctl --info") {
		t.Errorf("code fence contents were lost:\n%s", got)
	}
	if strings.Contains(got, "```") {
		t.Errorf("``` markers should not end up in the output:\n%s", got)
	}
}

func TestSplitTableRow(t *testing.T) {
	got := splitTableRow("| `--gen-config` | server and client | generates `backup.yaml` (`-mode=server\\|client\\|full`) |")
	if len(got) != 3 {
		t.Fatalf("len(cells) = %d, want 3: %#v", len(got), got)
	}
	if !strings.Contains(got[2], "server|client|full") {
		t.Errorf("escaped | was not restored: %q", got[2])
	}
}

func TestIsTableSeparator(t *testing.T) {
	if !isTableSeparator("|---|---|---|") {
		t.Error("table separator was not recognized")
	}
	if isTableSeparator("| `--gen-config` | text |") {
		t.Error("a regular table row was mistakenly treated as a separator")
	}
}

func TestRender_Table(t *testing.T) {
	md := "| Command | Does |\n|---|---|\n| `--info` | guide |\n"
	got := Render(md, false)
	if !strings.Contains(got, "Command") || !strings.Contains(got, "--info") || !strings.Contains(got, "guide") {
		t.Errorf("table was not rendered:\n%s", got)
	}
	if strings.Contains(got, "---") {
		t.Errorf("the separator row should not end up in the output:\n%s", got)
	}
}

func TestRender_MultiSectionDocDoesNotPanic(t *testing.T) {
	md := `# backupctl

Text with ` + "`" + `code` + "`" + ` and **bold**, a link [restic](https://restic.net/).

## Commands

| Command | Does |
|---|---|
| ` + "`" + `--init-server` + "`" + ` | set up the server |

` + "```bash\nbackupctl --info\n```" + `

> A note in a blockquote.
`
	got := Render(md, false)
	for _, want := range []string{"backupctl", "code", "bold", "restic", "--init-server", "set up the server", "backupctl --info", "note"} {
		if !strings.Contains(got, want) {
			t.Errorf("render lost %q:\n%s", want, got)
		}
	}
}
