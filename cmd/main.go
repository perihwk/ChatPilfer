package main

import (
	"flag"
	"log"

	"github.com/perihwk/chatpilfer/internal/provider/slack"
)

func main() {
	var proxyServer = flag.String("proxy-server", "", "optional proxy url")
	var userAgent = flag.String("user-agent", "", "optional user agent")
	var dCookie = flag.String("d-cookie", "", "the d cookie. Has prefix 'xoxd'")
	//var oiCookie = flag.String("oi-cookie", "", "the oi cookie")
	//var token = flag.String("token", "", "token for a specific workspace. Has prefix 'xoxc'")
	flag.Parse()

	slackProviders, err := slack.GetProvidersFromCookie(*proxyServer, *userAgent, *dCookie)
	if err != nil {
		log.Fatal(err)
	}

	for _, provider := range slackProviders {
		log.Println(provider.DisplayName)

		if err := provider.Initialize(); err != nil {
			log.Fatal(err)
		}

		if _, err := provider.GetPinnedChats(); err != nil {
			log.Fatal(err)
		}
	}
}
