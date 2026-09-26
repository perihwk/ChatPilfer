# ChatPilfer
Analyze, identify, and extract interesting information from communication applications.

This application was inspired by [emtunc's project SlackPirate](https://github.com/emtunc/SlackPirate) which I used many times while pentesting. However I consistently found myself wanting to use it for more than just Slack as most of my engagements did not include Slack. Additionally I found that the regex searching was generally not very useful for me so I scaled that back and instead focused more on making good searches and returning the text of the message along with the permalink.

Furthermore I wanted a convenient way to query multiple chat programs at once. If there is one thing I have consistently found across organizations it is that they love having multiple ways to communicate with each other. A mixture of Slack and Teams is by far the most common and so those are the first two I am creating ChatPilfer for. However with the advent of generalized LLMs obviously those are incredibly useful to be able to query. 

## Usage
```go run cmd/main.go --help```

  * Display the help menu - this includes information about all the groups of queries you can exclude

go run cmd/main.go --slack-cookie <d-cookie> --proxy-server http://<proxy-url>:<proxy-port> --exclude "connection-strings" --exclude "active-directory"

 ```go run cmd/main.go  --slack-cookie <d-cookie>```

* Identify all Slack workspaces the supplied cookie can access and then search them with all the enabled queries

 ```go run cmd/main.go  --proxy-server <proxy-url>```

* Route all HTTP requests through the chosen proxy-server. This is especially useful when circumventing [detections Slack has put into place](https://slack.engineering/catching-compromised-cookies/).

 ```go run cmd/main.go  --exclude "active-directory" --exclude "credentials"```

 * Exclude the groups of queries you don't want to run

  ```go run cmd/main.go --interactive```

  * WIP eventually this will be the full blown TUI