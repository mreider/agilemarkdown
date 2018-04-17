package git

import (
	"bytes"
	"os/exec"
	"strings"
)

type Git struct {
}

func CurrentUser() (string, error) {
	args := []string{"config", "user.name"}
	return runGitCommand(args)
}

func runGitCommand(args []string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	lines := make([]string, 0, 2)
	if out.Len() > 0 {
		lines = append(lines, out.String())
	}
	if stderr.Len() > 0 {
		lines = append(lines, stderr.String())
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), err
}
