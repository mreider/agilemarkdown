package commands

import (
	"fmt"
	"github.com/buger/goterm"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"strconv"
	"time"
)

var ProgressCommand = cli.Command{
	Name:      "progress",
	Usage:     "Show the progress of a backlog over time",
	ArgsUsage: "NUMBER_OF_WEEKS",
	Action: func(c *cli.Context) error {
		var weekCount int
		if c.NArg() > 0 {
			weekCount, _ = strconv.Atoi(c.Args()[0])
		}
		if weekCount <= 0 {
			weekCount = 12
		}

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		bck, err := backlog.LoadBacklog(".")
		if err != nil {
			return err
		}

		currentDate := time.Now().UTC()
		items := bck.ItemsByStatus("l")
		pointsByWeekDelta := make(map[int]float64)
		for _, item := range items {
			modified := item.Modified()
			weekDelta := WeekDelta(currentDate, modified)
			if -weekCount < weekDelta && weekDelta <= 0 {
				itemPoints, _ := strconv.ParseFloat(item.Estimate(), 64)
				pointsByWeekDelta[weekDelta] += itemPoints
			}
		}

		chart := goterm.NewLineChart(84, 20)

		data := new(goterm.DataTable)
		data.AddColumn("Week")
		data.AddColumn("Points")

		for i := -weekCount + 1; i <= 0; i++ {
			data.AddRow(float64(i), pointsByWeekDelta[i])
		}

		fmt.Println(chart.Draw(data))

		return nil
	},
}
