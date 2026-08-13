package main

import (
	"strings"
	"testing"
)

func TestBuildExcludeList_KnownPreset(t *testing.T) {
	f := FileBackupConfig{Path: "/var/www/site", Preset: "wordpress"}
	got, err := buildExcludeList(f)
	if err != nil {
		t.Fatalf("buildExcludeList: %v", err)
	}
	found := false
	for _, p := range got {
		if p == "wp-content/cache/**" {
			found = true
		}
	}
	if !found {
		t.Errorf("wordpress preset missing wp-content/cache/**: %v", got)
	}
}

func TestBuildExcludeList_EmptyPresetMeansNone(t *testing.T) {
	f := FileBackupConfig{Path: "/srv/data", Exclude: []string{"*.bak"}}
	got, err := buildExcludeList(f)
	if err != nil {
		t.Fatalf("buildExcludeList: %v", err)
	}
	if len(got) != 1 || got[0] != "*.bak" {
		t.Errorf("expected only the custom exclude, got %v", got)
	}
}

func TestBuildExcludeList_CombinesPresetAndCustom(t *testing.T) {
	f := FileBackupConfig{Path: "/var/www/site", Preset: "joomla", Exclude: []string{"*.mp4"}}
	got, err := buildExcludeList(f)
	if err != nil {
		t.Fatalf("buildExcludeList: %v", err)
	}
	if got[len(got)-1] != "*.mp4" {
		t.Errorf("custom exclude should be appended last, got %v", got)
	}
	if len(got) <= 1 {
		t.Errorf("expected preset patterns plus the custom one, got %v", got)
	}
}

func TestBuildExcludeList_UnknownPreset(t *testing.T) {
	f := FileBackupConfig{Path: "/var/www/site", Preset: "typo3"}
	if _, err := buildExcludeList(f); err == nil {
		t.Fatal("expected an error for an unknown preset")
	} else if !strings.Contains(err.Error(), "typo3") {
		t.Errorf("error should name the bad preset: %v", err)
	}
}

func TestFilePresets_VendorAndNodeModulesNotExcluded(t *testing.T) {
	for name, patterns := range filePresets {
		for _, p := range patterns {
			if strings.Contains(p, "vendor") || strings.Contains(p, "node_modules") {
				t.Errorf("preset %q excludes dependency dir %q — decided to keep these for fast restores", name, p)
			}
		}
	}
}

func TestSupportedFilePresets_MatchesMap(t *testing.T) {
	for _, name := range supportedFilePresets {
		if _, ok := filePresets[name]; !ok {
			t.Errorf("supportedFilePresets lists %q but filePresets has no entry for it", name)
		}
	}
	for name := range filePresets {
		found := false
		for _, s := range supportedFilePresets {
			if s == name {
				found = true
			}
		}
		if !found {
			t.Errorf("filePresets has %q but supportedFilePresets doesn't list it", name)
		}
	}
}
