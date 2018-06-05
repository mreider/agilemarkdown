package main

import (
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/commands"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/users"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	version       = "0.0.0"
	configName    = ".config.json"
	defaultConfig = `
{
  "SmtpServer": "",
  "SmtpUser": "",
  "SmtpPassword": "",
  "RemoteGitUrlFormat": "%s/blob/master/%s"
}`
)

func main() {
	rootDir, _ := filepath.Abs(".")
	for rootDir != "" {
		_, err := os.Stat(filepath.Join(rootDir, ".git"))
		if err == nil {
			break
		}
		rootDir = filepath.Dir(rootDir)
	}
	addConfigAndGitIgnore(rootDir)

	cfgPath := filepath.Join(rootDir, configName)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Can't load the config file %s: %v\n", cfgPath, err)
	}

	users.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))

	err = setBashAutoComplete()
	if err != nil {
		fmt.Printf("can't set bash autocomplete: %v\n", err)
	}

	rand.Seed(time.Now().Unix())

	i, err := strconv.ParseInt(version, 10, 64)
	if err == nil {
		version = time.Unix(i, 0).UTC().Format("2006.01.02.150405")
	}

	app := cli.NewApp()
	app.Version = version
	app.EnableBashCompletion = true
	app.Description = "A framework for managing a backlog using Git, Markdown, and YAML"
	app.Usage = app.Description

	app.Commands = []cli.Command{
		commands.CreateBacklogCommand,
		commands.CreateItemCommand,
		commands.CreateIdeaCommand,
		commands.NewSyncCommand(cfg),
		commands.WorkCommand,
		commands.PointsCommand,
		commands.AssignUserCommand,
		commands.ChangeStatusCommand,
		commands.ProgressCommand,
		commands.AliasCommand,
		commands.ImportCommand,
		commands.ArchiveCommand,
		commands.CreateUserCommand,
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func setBashAutoComplete() error {
	return autocomplete.AddAliasWithBashAutoComplete("")
}

func addConfigAndGitIgnore(rootDir string) {
	configPath := filepath.Join(rootDir, configName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ioutil.WriteFile(configPath, []byte(strings.TrimLeftFunc(defaultConfig, unicode.IsSpace)), 0644)
	}
	gitIgnorePath := filepath.Join(rootDir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		ioutil.WriteFile(gitIgnorePath, []byte(configName), 0644)
	}
}
