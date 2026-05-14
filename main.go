package main

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/commands"
	"github.com/mreider/agilemarkdown/git"
	"github.com/urfave/cli/v3"
	"log"
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

	if i, err := strconv.ParseInt(version, 10, 64); err == nil {
		version = time.Unix(i, 0).UTC().Format("2006.01.02.150405")
	}

	app := &cli.Command{
		Name:                  "agilemarkdown",
		Version:               version,
		Description:           "A framework for managing a backlog using Git, Markdown, and YAML",
		Usage:                 "A framework for managing a backlog using Git, Markdown, and YAML",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			commands.InitCommand,
			commands.CreateBacklogCommand,
			commands.CreateItemCommand,
			commands.NewSyncCommand(),
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
			commands.StartCommand,
			commands.FinishCommand,
			commands.DeliverCommand,
			commands.AcceptCommand,
			commands.RejectCommand,
			commands.RankCommand,
			commands.IceCommand,
			commands.UnIceCommand,
			commands.ShowCommand,
			commands.EstimateCommand,
			commands.TagCommand,
			commands.EpicCommand,
			commands.HypothesisCommand,
			commands.StrengthCommand,
			commands.CycleTimeCommand,
			commands.RejectionRateCommand,
			commands.TeamAgreementsCommand,
			commands.RecordLearningCommand,
			commands.BlockCommand,
			commands.UnblockCommand,
			commands.CommentCommand,
			commands.TaskCommand,
			commands.AcceptanceCommand,
			commands.AlignCommand,
			commands.DashboardCommand,
			commands.NextCommand,
			commands.AcceptancePromptCommand,
			commands.CoachCheckCommand,
			commands.CoachStatusCommand,
			commands.IterationFitCommand,
			commands.InceptionCommand,
			commands.SprintCommand,
			commands.RetroCommand,
			commands.PullCommand,
			commands.ListBacklogsCommand,
			commands.ListItemsCommand,
			commands.GetItemCommand,
			commands.GetCommentsCommand,
			commands.TypeMixCommand,
			commands.WhoamiCommand,
			commands.HistoryCommand,
			commands.SearchCommand,
			commands.SetDescriptionCommand,
			commands.NewMCPCommand(version),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func setBashAutoComplete() error {
	return autocomplete.AddAliasWithBashAutoComplete("")
}
