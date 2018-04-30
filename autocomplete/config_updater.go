package autocomplete

import (
	"io/ioutil"
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
		return err
	}
	content, err := ioutil.ReadFile(configPath)
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
	err = ioutil.WriteFile(tempFilePath, []byte(strings.Join(lines, "\n")), stat.Mode())
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
