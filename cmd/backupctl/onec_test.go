package main

import (
	"strings"
	"testing"
)

func TestWriteOneCDump_FullDumpOnly(t *testing.T) {
	var sb strings.Builder
	oc := OneCConfig{
		Name:         "mybase",
		Path:         "/home/user1cv8/infobases/mybase",
		Binary:       "/opt/1cv8/x86_64/8.3.23.1912/1cv8",
		User:         "Администратор",
		PasswordFile: "/root/.onec-mybase-password",
	}
	if err := writeOneCDump(&sb, oc); err != nil {
		t.Fatalf("writeOneCDump: %v", err)
	}
	got := sb.String()

	for _, want := range []string{
		`ONEC_PASS_mybase="$(cat '/root/.onec-mybase-password')"`,
		`DESIGNER`,
		`/F"/home/user1cv8/infobases/mybase"`,
		`/N"Администратор"`,
		`/P"$ONEC_PASS_mybase"`,
		`/DumpIB '$DUMPS/mybase.dt'`,
		`/DisableStartupDialogs`,
		`/DisableStartupMessages`,
		`unset ONEC_PASS_mybase`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("script does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "DumpCfg") {
		t.Errorf("DumpConfig=false should not emit /DumpCfg:\n%s", got)
	}
}

func TestWriteOneCDump_WithConfigDump(t *testing.T) {
	var sb strings.Builder
	oc := OneCConfig{
		Name:         "mybase",
		Path:         "/srv/1c/mybase",
		Binary:       "/opt/1cv8/x86_64/8.3.23.1912/1cv8",
		User:         "admin",
		PasswordFile: "/root/.onec-pass",
		DumpConfig:   true,
	}
	if err := writeOneCDump(&sb, oc); err != nil {
		t.Fatalf("writeOneCDump: %v", err)
	}
	got := sb.String()
	if !strings.Contains(got, "/DumpIB '$DUMPS/mybase.dt'") {
		t.Errorf("missing .dt dump:\n%s", got)
	}
	if !strings.Contains(got, "/DumpCfg '$DUMPS/mybase.cf'") {
		t.Errorf("missing .cf dump:\n%s", got)
	}
}

func TestWriteOneCDump_InlinePassword(t *testing.T) {
	var sb strings.Builder
	oc := OneCConfig{
		Name:     "mybase",
		Path:     "/srv/1c/mybase",
		Binary:   "/opt/1cv8/1cv8",
		User:     "admin",
		Password: "hunter2",
	}
	if err := writeOneCDump(&sb, oc); err != nil {
		t.Fatalf("writeOneCDump: %v", err)
	}
	got := sb.String()
	if !strings.Contains(got, "ONEC_PASS_mybase='hunter2'") {
		t.Errorf("inline password not set as expected:\n%s", got)
	}
	if strings.Contains(got, "$(cat") {
		t.Errorf("should not read from a file when password is inline:\n%s", got)
	}
}

func TestWriteOneCDump_RequiresExactlyOnePasswordSource(t *testing.T) {
	cases := []OneCConfig{
		{Name: "a", Path: "/p", Binary: "/b", User: "u"},                                    // neither
		{Name: "a", Path: "/p", Binary: "/b", User: "u", Password: "x", PasswordFile: "/f"}, // both
	}
	for _, oc := range cases {
		var sb strings.Builder
		if err := writeOneCDump(&sb, oc); err == nil {
			t.Errorf("expected an error for %+v", oc)
		}
	}
}

func TestWriteOneCDump_RequiredFieldsMissing(t *testing.T) {
	base := OneCConfig{Name: "a", Path: "/p", Binary: "/b", User: "u", PasswordFile: "/f"}

	cases := []struct {
		name   string
		modify func(*OneCConfig)
	}{
		{"name", func(o *OneCConfig) { o.Name = "" }},
		{"path", func(o *OneCConfig) { o.Path = "" }},
		{"binary", func(o *OneCConfig) { o.Binary = "" }},
		{"user", func(o *OneCConfig) { o.User = "" }},
	}
	for _, c := range cases {
		oc := base
		c.modify(&oc)
		var sb strings.Builder
		if err := writeOneCDump(&sb, oc); err == nil {
			t.Errorf("missing %s: expected an error", c.name)
		}
	}
}

func TestShSingleQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{`hello`, `'hello'`},
		{`with space`, `'with space'`},
		{`it's`, `'it'\''s'`},
		{`/F"path"`, `'/F"path"'`},
	}
	for _, c := range cases {
		got := shSingleQuote(c.in)
		if got != c.want {
			t.Errorf("shSingleQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShellSafeIdent(t *testing.T) {
	got := shellSafeIdent("my-base 1")
	for _, r := range got {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			t.Errorf("shellSafeIdent produced an unsafe rune %q in %q", r, got)
		}
	}
}
