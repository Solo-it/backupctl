package main

import (
	"fmt"
	"os"
	"os/exec"
)

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func userExists(name string) bool {
	return exec.Command("id", name).Run() == nil
}

func runShell(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
