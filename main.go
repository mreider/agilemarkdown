package main

import (
	"fmt"
	"github.com/mreider/agilemarkdown/commands"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	err := setBashAutoComplete()
	if err != nil {
		fmt.Printf("can't set bash autocomplete: %v\n", err)
	}

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Description = "A framework for managing a backlog using Git, Markdown, and YAML"
	app.Usage = app.Description

	app.Commands = []cli.Command{
		commands.CreateBacklogCommand,
		commands.CreateItemCommand,
		commands.SyncCommand,
		commands.WorkCommand,
		commands.PointsCommand,
		commands.AssignUserCommand,
		commands.ChangeStatusCommand,
		commands.ProgressCommand,
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

const bashAutoCompleteScript = `#! /bin/bash

: ${PROG:=$(basename ${BASH_SOURCE})}

_cli_bash_autocomplete() {
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _cli_bash_autocomplete $PROG

unset PROG
`

func setBashAutoComplete() error {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil
	}

	cmdDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	autoCompleteScriptPath := filepath.Join(cmdDir, fmt.Sprintf("%s_bash_autocomplete", filepath.Base(os.Args[0])))
	_, err := os.Stat(autoCompleteScriptPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		err = ioutil.WriteFile(autoCompleteScriptPath, []byte(bashAutoCompleteScript), 0644)
		if err != nil {
			return err
		}
	}

	bashRcName := ".bashrc"
	if runtime.GOOS == "darwin" {
		bashRcName = ".bash_profile"
	}

	usr, _ := user.Current()
	bashRcPath := filepath.Join(usr.HomeDir, bashRcName)
	stat, err := os.Stat(bashRcPath)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadFile(bashRcPath)
	if err != nil {
		return err
	}

	autoCompleteLine := fmt.Sprintf("PROG=%s source %s", filepath.Base(os.Args[0]), autoCompleteScriptPath)
	hasAutoCompleteLine := false

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == autoCompleteLine {
			hasAutoCompleteLine = true
			break
		}
	}

	if hasAutoCompleteLine {
		return nil
	}

	for lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = append(lines, "")
	lines = append(lines, autoCompleteLine)
	lines = append(lines, "")
	lines = append(lines, "")

	tempFilePath := bashRcPath + "." + strconv.FormatInt(time.Now().Unix(), 10)
	err = ioutil.WriteFile(tempFilePath, []byte(strings.Join(lines, "\n")), stat.Mode())
	if err != nil {
		return err
	}
	return os.Rename(tempFilePath, bashRcPath)
}
