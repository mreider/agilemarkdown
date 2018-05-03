package main

import (
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"github.com/mreider/agilemarkdown/commands"
	"gopkg.in/urfave/cli.v1"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	err := setBashAutoComplete()
	if err != nil {
		fmt.Printf("can't set bash autocomplete: %v\n", err)
	}

	rand.Seed(time.Now().Unix())

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
		commands.AliasCommand,
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func setBashAutoComplete() error {
	return autocomplete.AddAliasWithBashAutoComplete("")
}
