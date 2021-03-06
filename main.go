package main

import (
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/commands"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	version = "0.0.0"
)

func main() {
	rootDir, _ := filepath.Abs(".")
	gitRootDir := git.GetRootGitDirectory(rootDir)
	if gitRootDir != "" {
		rootDir = gitRootDir
		root := backlog.NewBacklogsStructure(rootDir)
		err := commands.AddConfigAndGitIgnore(root)
		if err != nil {
			fmt.Println(err)
		}
		backlog.NewUserList(root.UsersDirectory())
	}
	err := setBashAutoComplete()
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
		commands.NewSyncCommand(),
		commands.WorkCommand,
		commands.PointsCommand,
		commands.AssignUserCommand,
		commands.ChangeStatusCommand,
		commands.VelocityCommand,
		commands.AliasCommand,
		commands.ImportCommand,
		commands.ArchiveCommand,
		commands.TimelineCommand,
		commands.DeleteTagCommand,
		commands.ChangeTagCommand,
		commands.CreateUserCommand,
		commands.DeleteUserCommand,
		commands.ChangeUserCommand,
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func setBashAutoComplete() error {
	return autocomplete.AddAliasWithBashAutoComplete("")
}
