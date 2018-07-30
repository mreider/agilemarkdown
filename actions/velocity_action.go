package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
)

type VelocityAction struct {
	backlogDir string
	weekCount  int
}

func NewVelocityAction(backlogDir string, weekCount int) *VelocityAction {
	return &VelocityAction{backlogDir: backlogDir, weekCount: weekCount}
}

func (a *VelocityAction) Execute() error {
	bck, err := backlog.LoadBacklog(a.backlogDir)
	if err != nil {
		return err
	}
	chart, err := backlog.BacklogView{}.VelocityText(bck, a.weekCount, 84)
	if err != nil {
		return err
	}
	fmt.Println(chart)

	return nil
}
