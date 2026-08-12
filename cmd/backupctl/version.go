package main

import "fmt"

// version/commit/date are set via -ldflags at build time. GoReleaser's
// default ldflags template targets exactly these variable names:
//
//	-X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	fmt.Printf("backupctl %s (%s, %s)\n", version, commit, date)
}
