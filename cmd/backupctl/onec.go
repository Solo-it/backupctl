package main

import (
	"fmt"
	"strings"
)

// writeOneCDump appends the dump command(s) for one 1c: entry. Full
// verification against a real 1C:Enterprise install is still outstanding
// (see README) — this follows the documented `1cv8 DESIGNER` console-mode
// syntax, but hasn't been run against a live server yet.
func writeOneCDump(sb *strings.Builder, oc OneCConfig) error {
	if oc.Name == "" {
		return fmt.Errorf("1c: entry is missing name")
	}
	if oc.Path == "" {
		return fmt.Errorf("1c: entry %q is missing path", oc.Name)
	}
	if oc.Binary == "" {
		return fmt.Errorf("1c: entry %q is missing binary (path to the 1cv8 executable — varies by platform version, no default)", oc.Name)
	}
	if oc.User == "" {
		return fmt.Errorf("1c: entry %q is missing user", oc.Name)
	}
	hasPassword := oc.Password != ""
	hasPasswordFile := oc.PasswordFile != ""
	if hasPassword == hasPasswordFile {
		return fmt.Errorf("1c: entry %q must set exactly one of password or password_file", oc.Name)
	}

	passVar := fmt.Sprintf("ONEC_PASS_%s", shellSafeIdent(oc.Name))
	if hasPasswordFile {
		sb.WriteString(fmt.Sprintf("%s=\"$(cat %s)\"\n", passVar, shSingleQuote(oc.PasswordFile)))
	} else {
		// Embedded directly in backup.sh — visible in `ps aux` while
		// 1cv8 runs (the DESIGNER CLI takes the password as a plain
		// argument, there's no other way to pass it) and sits in
		// backup.yaml in cleartext. password_file avoids the second
		// part; the ps exposure is inherent to 1C's own CLI and can't
		// be avoided either way.
		sb.WriteString(fmt.Sprintf("%s=%s\n", passVar, shSingleQuote(oc.Password)))
	}

	dtPath := fmt.Sprintf("$DUMPS/%s.dt", oc.Name)
	sb.WriteString(fmt.Sprintf(
		"%s DESIGNER %s %s /P\"$%s\" /DumpIB %s /DisableStartupDialogs /DisableStartupMessages\n",
		shSingleQuote(oc.Binary),
		shSingleQuote(`/F"`+oc.Path+`"`),
		shSingleQuote(`/N"`+oc.User+`"`),
		passVar,
		shSingleQuote(dtPath),
	))

	if oc.DumpConfig {
		cfPath := fmt.Sprintf("$DUMPS/%s.cf", oc.Name)
		sb.WriteString(fmt.Sprintf(
			"%s DESIGNER %s %s /P\"$%s\" /DumpCfg %s /DisableStartupDialogs /DisableStartupMessages\n",
			shSingleQuote(oc.Binary),
			shSingleQuote(`/F"`+oc.Path+`"`),
			shSingleQuote(`/N"`+oc.User+`"`),
			passVar,
			shSingleQuote(cfPath),
		))
	}

	sb.WriteString(fmt.Sprintf("unset %s\n", passVar))
	return nil
}

// shSingleQuote wraps s in POSIX sh single quotes, safe for any content
// including spaces and embedded double quotes — exactly what's needed
// for 1C's own /F"..." /N"..." argument convention, which must arrive at
// the process as one literal argv element, quotes included.
func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellSafeIdent turns a 1c: entry name into something usable as (part
// of) a shell variable name — letters/digits/underscore only.
func shellSafeIdent(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	return sb.String()
}
