package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/perihwk/chatpilfer/internal/provider"
	"github.com/perihwk/chatpilfer/internal/provider/slack"
	"github.com/perihwk/chatpilfer/internal/provider/teams"
)

func main() {
	var proxyServer = flag.String("proxy-server", "", "optional proxy url")
	var userAgent = flag.String("user-agent", "", "optional user agent")
	var dCookie = flag.String("slack-cookie", "", "the slack d cookie. Has prefix 'xoxd'")
	//var skypeToken = flag.String("skype-token", "", "Teams Skype bearer token. Value for the `X-Skypetoken` header")
	var teamsToken = flag.String("teams-token", "", "Teams bearer token with SubstrateSearch scp")
	var outputFolder = flag.String("output", "out", "base folder to write all the data to")

	var enabledQueries = provider.AllQueryGroups
	exclude := &excludeQueryGroup{queries: enabledQueries}
	flag.Var(exclude, "exclude", "query groups to exclude")
	//var oiCookie = flag.String("oi-cookie", "", "the oi cookie")
	//var token = flag.String("token", "", "token for a specific workspace. Has prefix 'xoxc'")
	flag.Parse()

	service := provider.NewService(enabledQueries, *outputFolder)
	if *dCookie != "" {
		slackProviders, err := slack.GetProvidersFromCookie(*proxyServer, *userAgent, *dCookie)
		if err != nil {
			log.Fatal(err)
		}

		for _, sp := range slackProviders {
			if err := sp.Initialize(); err != nil {
				log.Fatal(err)
			}
			service.AddProvider(sp.DisplayName, provider.Slack, sp)
		}
	}

	if *teamsToken != "" {
		teamsProvider := teams.NewTeamsProvider(http.DefaultClient, *teamsToken, *userAgent)
		service.AddProvider(teamsProvider.DisplayName, provider.Slack, teamsProvider)
	}

	service.PilferAll()
}

type excludeQueryGroup struct {
	queries map[string]*provider.QueryGroups
}

func (eq excludeQueryGroup) String() string {
	var output []string
	for name := range eq.queries {
		output = append(output, name)
	}
	return strings.Join(output, ",")
}

func (eq excludeQueryGroup) Set(value string) error {
	delete(eq.queries, value)
	return nil
}
