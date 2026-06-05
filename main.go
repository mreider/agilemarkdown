package main

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/autocomplete"
	"github.com/mreider/agilemarkdown/commands"
	"github.com/urfave/cli/v3"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	version = "0.0.0"
)

func main() {
	// No filesystem side effects at startup. Earlier builds auto-created
	// .am/config.yaml + .gitignore and `git commit`ed them whenever `am`
	// ran inside *any* git repo — so opening an unrelated repo in an
	// editor that shells out to `am` (e.g. the VS Code extension probing
	// with `am list-backlogs`) silently polluted and committed to it.
	// Project setup now happens only on explicit `am init` /
	// `am create-backlog`.
	err := setBashAutoComplete()
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't set bash autocomplete: %v\n", err)
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
			commands.DeleteItemCommand,
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
			commands.UnstartCommand,
			commands.SetTypeCommand,
			commands.BugCommand,
			commands.ChoreCommand,
			commands.LogCommand,
			commands.SetReleaseDateCommand,
			commands.ProjectCommand,
			commands.ConfigCommand,
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
			commands.AlignCommand,
			commands.DashboardCommand,
			commands.NextCommand,
			commands.BriefCommand,
			commands.JournalCommand,
			commands.ModeCommand,
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
