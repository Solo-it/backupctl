package main

import (
	_ "embed"
	"fmt"

	"github.com/Solo-it/backupctl/internal/mdterm"
)

// README.md / README.ru.md are the single source of truth for the guide:
// the same text is read by a human on GitHub and by --info in the
// terminal (embed bakes the file into the binary at build time, so there
// is nothing extra to keep in sync).
//
//go:embed README.md
var readmeTextEN string

//go:embed README.ru.md
var readmeTextRU string

func printInfo(lang string) {
	text := readmeTextEN
	if lang == "ru" {
		text = readmeTextRU
	}
	fmt.Print(mdterm.Render(text, mdterm.SupportsColor()))
}
