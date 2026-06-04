package autocomplete

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func addLineToConfig(configPath, line string) error {
	stat, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No shell rc file to augment, so there is nothing to do.
			// This is the normal case for anyone whose login shell isn't
			// bash (zsh users have no ~/.bash_profile), and the
			// autocomplete setup runs on every invocation — so it must
			// stay silent rather than surface an error that callers would
			// otherwise see prepended to stdout (e.g. corrupting --json).
			return nil
		}
		return err
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	hasLine := false
	lines := strings.Split(string(content), "\n")
	for _, ln := range lines {
		if ln == line {
			hasLine = true
			break
		}
	}

	if hasLine {
		return nil
	}

	for lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = append(lines, "")
	lines = append(lines, line)
	lines = append(lines, "")
	lines = append(lines, "")

	tempFilePath := configPath + "." + strconv.FormatInt(time.Now().Unix(), 10)
	err = os.WriteFile(tempFilePath, []byte(strings.Join(lines, "\n")), stat.Mode())
	if err != nil {
		return err
	}
	return os.Rename(tempFilePath, configPath)
}

func BashRcPath() string {
	bashRcName := ".bashrc"
	if runtime.GOOS == "darwin" {
		bashRcName = ".bash_profile"
	}

	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, bashRcName)
}
