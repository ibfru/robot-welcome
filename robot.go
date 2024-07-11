package main

import (
	"community-robot-lib/config"
	"community-robot-lib/framework"
	"community-robot-lib/utils"
	"encoding/json"
	"fmt"
	sdk "git-platform-sdk"
	sig "github.com/opensourceways/robot-sig-info-cache"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

const (
	botName        = "welcome"
	welcomeMessage = `
Hi ***%s***, welcome to the %s Community.
I'm the Bot here serving you. You can find the instructions on how to interact with me at **[Here](%s)**.
If you have any questions, please contact the SIG: [%s](https://gitee.com/openeuler/community/tree/master/sig/%s), and any of the maintainers: @%s`
	welcomeMessage2 = `
Hi ***%s***, welcome to the %s Community.
I'm the Bot here serving you. You can find the instructions on how to interact with me at **[Here](%s)**.
If you have any questions, please contact the SIG: [%s](https://gitee.com/openeuler/community/tree/master/sig/%s), and any of the maintainers: @%s, any of the committers: @%s`
	welcomeMessage3 = `
Hi ***%s***, welcome to the %s Community.
I'm the Bot here serving you. You can find the instructions on how to interact with me at **[Here](%s)**.
If you have any questions, please contact the SIG: [%s](https://gitee.com/openeuler/community/tree/master/sig/%s), and any of the maintainers.
`
)

type robot struct {
	cli    *sdk.ClientTarget
	sigCli *sig.SDK
}

func newRobot(cli *sdk.ClientTarget, sigSdk *sig.SDK) *robot {
	return &robot{cli: cli, sigCli: sigSdk}
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) getConfig(cfg config.Config, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}

	return nil, fmt.Errorf("no config for this repo:%s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(f framework.HandlerRegister) {
	f.RegisterIssueHandler(bot.handleIssue)
	f.RegisterPullRequestHandler(bot.handlePullRequest)
}

const (
	Issue = iota
	PullRequest
)

type eventArgs struct {
	event   *sdk.GenericEvent
	cnf     *botConfig
	log     *logrus.Entry
	flag    int
	author  string
	sigName string
}

func (bot *robot) handlePullRequest(e *sdk.GenericEvent, pc config.Config, log *logrus.Entry) error {
	if e.Action != sdk.ActionStateCreated {
		return nil
	}

	cfg, err := bot.getConfig(pc, e.Org, e.Repo)
	if err != nil {
		return err
	}

	p := &eventArgs{
		flag:   PullRequest,
		event:  e,
		author: e.PRAuthor,
		cnf:    cfg,
		log:    log,
	}

	bot.handleNewcomerLabel(p)
	return bot.handle(p)
}

func (bot *robot) handleIssue(e *sdk.GenericEvent, pc config.Config, log *logrus.Entry) error {
	if e.Action != sdk.ActionStateCreated {
		return nil
	}

	cfg, err := bot.getConfig(pc, e.Org, e.Repo)
	if err != nil {
		return err
	}

	p := &eventArgs{
		flag:   Issue,
		event:  e,
		author: e.IssueAuthor,
		cnf:    cfg,
		log:    log,
	}

	return bot.handle(p)
}

func (bot *robot) handleNewcomerLabel(p *eventArgs) {
	mErr := utils.NewMultiErrors()
	resp, err := http.Get(fmt.Sprintf("https://ipb.osinfra.cn/pulls?author=%s", p.author))
	if err != nil {
		mErr.AddError(err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)
	type T struct {
		Total int `json:"total,omitempty"`
	}

	var t T
	err = json.Unmarshal(body, &t)
	if err != nil {
		mErr.AddError(err)
	}

	if t.Total == 0 {
		if err = bot.cli.AddPRLabels(&sdk.PRParameter{
			Org:    p.event.Org,
			Repo:   p.event.Repo,
			Number: p.event.PRNumber,
			Labels: []string{"newcomer"},
		}); err != nil {
			mErr.AddError(err)
		}
	}

	if le := mErr.Err(); le != nil {
		p.log.Error(le)
	}
}

func (bot *robot) handle(p *eventArgs) error {

	sigName, err := bot.sigCli.GetSigNameByOrgRepo(p.event.Org, p.event.Repo)

	if err != nil {
		return err
	}

	p.sigName = sigName
	comment, err := bot.generateComment(p)

	mErr := utils.NewMultiErrors()
	if p.flag == Issue {
		err = bot.cli.AddIssueComment(&sdk.IssueParameter{
			Org:     p.event.Org,
			Repo:    p.event.Repo,
			Number:  p.event.IssueNumber,
			Comment: comment,
		})
	} else {
		err = bot.cli.AddPRComment(&sdk.PRParameter{
			Org:     p.event.Org,
			Repo:    p.event.Repo,
			Number:  p.event.PRNumber,
			Comment: comment,
		})
	}
	if err != nil {
		mErr.AddError(err)
	}

	label := fmt.Sprintf("sig/%s", sigName)
	if n := 20; len(label) > n {
		label = label[:n]
	}

	if err = bot.createLabelIfNeed(p.event.Org, p.event.Repo, label); err != nil {
		p.log.Errorf("create repo label:%s, err:%s", label, err.Error())
	}

	if p.flag == Issue {
		err = bot.cli.AddIssueLabels(&sdk.IssueParameter{
			Org:    p.event.Org,
			Repo:   p.event.Repo,
			Number: p.event.IssueNumber,
			Labels: []string{label},
		})
	} else {
		err = bot.cli.AddPRLabels(&sdk.PRParameter{
			Org:    p.event.Org,
			Repo:   p.event.Repo,
			Number: p.event.PRNumber,
			Labels: []string{label},
		})
	}
	if err != nil {
		mErr.AddError(err)
	}

	return mErr.Err()
}

func (bot *robot) generateComment(p *eventArgs) (string, error) {

	if p.cnf.NoNeedToNotice {
		return fmt.Sprintf(welcomeMessage3, p.author, p.cnf.CommunityName, p.cnf.CommandLink, p.sigName, p.sigName), nil
	}

	var maintainers []string
	maintainersFromGitPlatform, err := bot.cli.ListCollaborator(p.event.Org, p.event.Repo)
	if err != nil {
		return "", err
	}

	// 仓库自己配置 maintainers - 仓库下不同目录归属不同的 owner
	if p.cnf.WelcomeSimpler {
		contentMap, err1 := bot.sigCli.GetContentByPath("")
		repoOwner := matchOwnerByPRChanges(contentMap, p)
		if err1 == nil && len(repoOwner) != 0 {
			maintainers = append(maintainersFromGitPlatform, repoOwner...)
		}
	} else {

		maintainersFromSigInfo, err2 := bot.sigCli.GetRepositoryMaintainerByOrgRepo(p.event.Org, p.event.Repo)
		if err2 != nil {
			return "", err2
		}
		maintainers = append(maintainersFromGitPlatform, maintainersFromSigInfo...)
	}

	if p.cnf.NeedAssign {
		// missing assign issue
		if err = bot.cli.AssignPR(&sdk.PRParameter{
			Org:       p.event.Org,
			Repo:      p.event.Repo,
			Number:    p.event.PRNumber,
			Reviewers: maintainers,
		}); err != nil {
			return "", err
		}
	}

	committers, _ := bot.sigCli.GetRepositoryCommitterByOrgRepo(p.event.Org, p.event.Repo)
	if len(committers) != 0 {
		return fmt.Sprintf(
			welcomeMessage2, p.author, p.cnf.CommunityName, p.cnf.CommandLink,
			p.sigName, p.sigName, strings.Join(maintainers, " , @"), strings.Join(committers, " , @"),
		), nil
	}

	return fmt.Sprintf(
		welcomeMessage, p.author, p.cnf.CommunityName, p.cnf.CommandLink,
		p.sigName, p.sigName, strings.Join(maintainers, " , @"),
	), nil
}

func (bot *robot) createLabelIfNeed(org, repo, label string) error {
	arg := &sdk.LabelParameter{
		Org:  org,
		Repo: repo,
		Name: label,
	}
	repoLabels, err := bot.cli.GetRepoLabels(arg)
	if err != nil {
		return err
	}

	if repoLabels.Has(label) {
		return nil
	}

	return bot.cli.AddRepoLabels(arg)
}
