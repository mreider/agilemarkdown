package git

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
)

var (
	usersRe = regexp.MustCompile(`^\d+\s+(.*)$`)
)

func CurrentUser() (string, error) {
	args := []string{"config", "user.name"}
	return runGitCommand(args)
}

func KnownUsers() ([]string, error) {
	args := []string{"shortlog", "--summary", "HEAD"}
	output, err := runGitCommand(args)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	users := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		match := usersRe.FindStringSubmatch(line)
		if match != nil {
			users = append(users, match[1])
		}
	}
	return users, nil
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
