package git

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
	"time"
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

func AddAll() error {
	args := []string{"add", "-A"}
	_, err := runGitCommand(args)
	return err
}

func Add(fileName string) error {
	args := []string{"add", fileName}
	_, err := runGitCommand(args)
	return err
}

func Commit(msg string) error {
	args := []string{"commit", "-m", msg}
	_, err := runGitCommand(args)
	return err
}

func CommitNoEdit() error {
	args := []string{"commit", "--no-edit"}
	_, err := runGitCommand(args)
	return err
}

func Fetch() error {
	args := []string{"fetch"}
	_, err := runGitCommand(args)
	return err
}

func Merge() (string, error) {
	args := []string{"merge", "--commit"}
	return runGitCommand(args)
}

func AbortMerge() error {
	args := []string{"merge", "--abort"}
	_, err := runGitCommand(args)
	return err
}

func Push() error {
	args := []string{"push"}
	_, err := runGitCommand(args)
	return err
}

func SetUpstream() error {
	args := []string{"branch", "--set-upstream-to", "origin"}
	_, err := runGitCommand(args)
	return err
}

func Status() (string, error) {
	args := []string{"status"}
	return runGitCommand(args)
}

func ConflictFiles() ([]string, error) {
	args := []string{"diff", "--name-only", "--diff-filter=U"}
	out, err := runGitCommand(args)
	if err != nil {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

func CheckoutOurVersion(fileName string) error {
	args := []string{"checkout", "--ours", fileName}
	_, err := runGitCommand(args)
	return err
}

func InitCommitInfo(fileName string) (user string, created time.Time, err error) {
	args := []string{"log", "--reverse", "--format=format:%an|%ai", "--follow", "--", fileName}
	out, err := runGitCommand(args)
	if err != nil {
		return "", time.Time{}, err
	}
	firstLine := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	if firstLine == "" {
		return "", time.Time{}, nil
	}
	parts := strings.SplitN(firstLine, "|", 2)
	user = parts[0]
	created, _ = time.Parse("2006-01-02 15:04:05 -0700", parts[1])
	return user, created, nil
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
