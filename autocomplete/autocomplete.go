package autocomplete

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

const bashAutoCompleteScript = `#! /bin/bash

: ${PROG:=$(basename ${BASH_SOURCE})}

_cli_bash_autocomplete() {
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts=$( "%s" --generate-bash-completion )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _cli_bash_autocomplete $PROG

unset PROG
`

func getBashAutoCompleteScriptPath() (string, error) {
	absCmdPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmdDir := filepath.Dir(absCmdPath)
	autoCompleteScriptPath := filepath.Join(cmdDir, fmt.Sprintf("%s_bash_autocomplete", filepath.Base(absCmdPath)))
	_, err = os.Stat(autoCompleteScriptPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	autoCompleteScript := []byte(fmt.Sprintf(bashAutoCompleteScript, absCmdPath))
	if err == nil {
		currentScript, err := ioutil.ReadFile(autoCompleteScriptPath)
		if err == nil && bytes.Equal(autoCompleteScript, currentScript) {
			return autoCompleteScriptPath, nil
		}
	}

	err = ioutil.WriteFile(autoCompleteScriptPath, autoCompleteScript, 0644)
	if err != nil {
		return "", err
	}
	return autoCompleteScriptPath, nil
}

func AddAliasWithBashAutoComplete(alias string) error {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil
	}

	autoCompleteScriptPath, err := getBashAutoCompleteScriptPath()
	if err != nil {
		return err
	}

	absCmdPath, err := os.Executable()
	if err != nil {
		return err
	}
	if alias == "" {
		alias = filepath.Base(absCmdPath)
	} else {
		err = addLineToConfig(BashRcPath(), fmt.Sprintf("alias %s='%s'", alias, absCmdPath))
		if err != nil {
			return err
		}
	}
	return addLineToConfig(BashRcPath(), fmt.Sprintf("PROG=%s source %s", alias, autoCompleteScriptPath))
}
