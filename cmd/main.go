package main

import (
	"flag"
	"log"
	"strings"

	"github.com/perihwk/chatpilfer/internal/provider"
	"github.com/perihwk/chatpilfer/internal/provider/slack"
)

func main() {
	var proxyServer = flag.String("proxy-server", "", "optional proxy url")
	var userAgent = flag.String("user-agent", "", "optional user agent")
	var dCookie = flag.String("slack-cookie", "", "the slack d cookie. Has prefix 'xoxd'")

	var enabledQueries = provider.AllQueryGroups
	exclude := &excludeQueryGroup{queries: enabledQueries}
	flag.Var(exclude, "exclude", "query groups to exclude")
	//var oiCookie = flag.String("oi-cookie", "", "the oi cookie")
	//var token = flag.String("token", "", "token for a specific workspace. Has prefix 'xoxc'")
	flag.Parse()

	service := provider.NewService(enabledQueries)
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

	errors := service.PilferAll()
	if len(errors) > 0 {
		log.Println(errors)
	}
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
