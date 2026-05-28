package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	usersRe = regexp.MustCompile(`^\d+\s+(.*)\s+<([^>]+)>$`)
)

func CurrentUser() (name, email string, err error) {
	name, err = runGitCommand([]string{"config", "user.name"})
	if err != nil {
		return "", "", nil
	}
	email, err = runGitCommand([]string{"config", "user.email"})
	if err != nil {
		return "", "", nil
	}
	return name, email, nil
}

func KnownUsers() (names, emails []string, err error) {
	args := []string{"shortlog", "--summary", "-e", "-n", "HEAD"}
	output, err := runGitCommand(args)
	if err != nil {
		return nil, nil, err
	}
	lines := strings.Split(output, "\n")
	userNames := make([]string, 0, len(lines))
	userEmails := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		match := usersRe.FindStringSubmatch(line)
		if match != nil {
			userNames = append(userNames, match[1])
			userEmails = append(userEmails, match[2])
		}
	}
	return userNames, userEmails, nil
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

func Commit(msg, author string) error {
	args := []string{"commit", "-m", msg}
	if author != "" {
		args = append(args, "--author", author)
	}
	out, err := runGitCommand(args)
	if err != nil && strings.Contains(out, "nothing to commit, working tree clean") {
		err = nil
	}
	return err
}

func CommitNoEdit(author string) error {
	args := []string{"commit", "--no-edit"}
	if author != "" {
		args = append(args, "--author", author)
	}
	out, err := runGitCommand(args)
	if err != nil && strings.Contains(out, "nothing to commit, working tree clean") {
		err = nil
	}
	return err
}

func Fetch() error {
	args := []string{"fetch"}
	_, err := runGitCommand(args)
	return err
}

func HasRemote() (bool, error) {
	out, err := runGitCommand([]string{"remote"})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
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
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
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

func RemoteOriginURL() (url string, err error) {
	url, err = runGitCommand([]string{"config", "--get", "remote.origin.url"})
	if err != nil {
		return "", nil
	}
	return url, nil
}

func Init() error {
	_, err := runGitCommand([]string{"init"})
	return err
}

func RepoVersion(repoDir, path string) (string, error) {
	return runGitCommandInDirectory(repoDir, []string{"show", fmt.Sprintf("HEAD:%s", path)})
}

func ModifiedFiles(dir string) ([]string, error) {
	out, err := runGitCommandInDirectory(dir, []string{"ls-files", "-m"})
	if err != nil {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

func GetRootGitDirectory(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		if IsRootGitDirectory(dir) {
			return dir
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return ""
		}
		dir = parentDir
	}
}

func IsRootGitDirectory(dir string) bool {
	gitFolder := filepath.Join(dir, ".git")
	_, err := os.Stat(gitFolder)
	return err == nil
}

// HistoryEntry is one commit that touched a file, exposed to JSON
// consumers (CLI --json, MCP) for the per-item History panel.
type HistoryEntry struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Email   string `json:"email,omitempty"`
	When    string `json:"when"`
	Subject string `json:"subject"`
}

// HistoryFor returns commits that touched path, newest first.
// Uses --follow so renames are tracked. limit <= 0 returns all.
func HistoryFor(path string, limit int) ([]HistoryEntry, error) {
	args := []string{"log", "--follow", "--no-decorate", "--pretty=format:%H%x1f%an%x1f%ae%x1f%aI%x1f%s"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	}
	args = append(args, "--", path)
	out, err := runGitCommand(args)
	if err != nil {
		return nil, err
	}
	entries := make([]HistoryEntry, 0)
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 5)
		if len(parts) < 5 {
			continue
		}
		entries = append(entries, HistoryEntry{
			Hash: parts[0], Author: parts[1], Email: parts[2], When: parts[3], Subject: parts[4],
		})
	}
	return entries, nil
}

func runGitCommand(args []string) (string, error) {
	return runGitCommandInDirectory("", args)
}

func runGitCommandInDirectory(dir string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Dir = dir
	err := cmd.Run()
	lines := make([]string, 0, 2)
	if out.Len() > 0 {
		lines = append(lines, out.String())
	}
	if stderr.Len() > 0 {
		lines = append(lines, stderr.String())
	}
	output := strings.TrimSpace(strings.Join(lines, "\n"))
	return output, createVerboseError(err, output)
}

func createVerboseError(err error, out string) error {
	if err == nil {
		return nil
	}

	if out == "" {
		return err
	}

	return fmt.Errorf("%s\n%s", err.Error(), out)
}
