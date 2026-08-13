package main

import (
	"fmt"
	"strings"
)

// filePresets are built-in restic exclude patterns for common site/CMS
// layouts and for a generic user home directory — caches, temp files and
// already-compressed archives, not application code or dependencies
// (vendor/, node_modules/ are deliberately kept: restoring a site without
// them means re-running composer/npm install by hand during an incident,
// and they don't dominate backup size the way caches do).
var filePresets = map[string][]string{
	"none": {},

	"wordpress": {
		"wp-content/cache/**",
		"wp-content/uploads/cache/**",
		"wp-content/*backup*/**",
		"wp-content/updraft/**",
		"*.log",
		"*.zip", "*.tar.gz", "*.tar", "*.rar", "*.7z",
	},

	"bitrix": {
		"bitrix/cache/**",
		"bitrix/managed_cache/**",
		"bitrix/stack_cache/**",
		"bitrix/tmp/**",
		"bitrix/logs/**",
		"bitrix/backup/**",
		"upload/resize_cache/**",
		"*.log",
		"*.zip", "*.tar.gz", "*.tar", "*.rar", "*.7z",
	},

	"joomla": {
		"cache/**",
		"tmp/**",
		"administrator/cache/**",
		"logs/**",
		"*.log",
		"*.zip", "*.tar.gz", "*.tar", "*.rar", "*.7z",
	},

	"drupal": {
		"sites/*/files/css/**",
		"sites/*/files/js/**",
		"sites/*/files/styles/**",
		"sites/*/files/php/**",
		"tmp/**",
		"*.log",
		"*.zip", "*.tar.gz", "*.tar", "*.rar", "*.7z",
	},

	// A user's home directory: keep dotfiles like .ssh/.gnupg (restic backs
	// up everything under the path unless excluded — nothing needs to be
	// explicitly "included"), exclude known junk instead.
	"user": {
		".cache/**",
		".npm/**",
		".cargo/registry/**",
		".rustup/downloads/**",
		".local/share/Trash/**",
		".docker/**",
		"*.zip", "*.tar.gz", "*.tar", "*.rar", "*.7z",
	},
}

var supportedFilePresets = []string{"none", "wordpress", "bitrix", "joomla", "drupal", "user"}

// buildExcludeList is a pure function combining a preset's built-in
// patterns with the entry's own extra excludes. An empty Preset means
// "none" (no built-in patterns, only whatever's in Exclude).
func buildExcludeList(f FileBackupConfig) ([]string, error) {
	preset := f.Preset
	if preset == "" {
		preset = "none"
	}
	patterns, ok := filePresets[preset]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q for path %s, supported: %s", f.Preset, f.Path, strings.Join(supportedFilePresets, ", "))
	}
	out := make([]string, 0, len(patterns)+len(f.Exclude))
	out = append(out, patterns...)
	out = append(out, f.Exclude...)
	return out, nil
}
