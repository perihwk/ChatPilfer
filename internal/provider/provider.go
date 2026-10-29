package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type ProviderType int

const (
	Slack ProviderType = iota
	Teams
	ClaudeChat
)

func (pt *ProviderType) ToString() string {
	switch *pt {
	case Slack:
		return "slack"
	case Teams:
		return "teams"
	case ClaudeChat:
		return "claude-chat"
	default:
		return ""
	}
}

type ChatProvider interface {
	Initialize() error
	Pilfer(outputDir string, queries map[string]*QueryGroups) error
	SearchMessagesRegex(queries []string, regex string) (map[string][]any, error)
	SearchMessages(queries []string) (map[string][]any, error)
	SearchFiles(queries []string) (map[string][]any, error)
}

type Entry struct {
	Name    string
	Type    ProviderType
	Queries map[string]*QueryGroups
	ChatProvider
}

type service struct {
	outputFolder   string
	defaultQueries map[string]*QueryGroups
	providers      map[string]*Entry
}

func NewService(defaultQueries map[string]*QueryGroups, outputFolder string) *service {
	return &service{
		outputFolder:   outputFolder,
		defaultQueries: defaultQueries,
		providers:      make(map[string]*Entry),
	}
}

func (s *service) AddProvider(name string, providerType ProviderType, provider ChatProvider) {
	s.providers[name] = &Entry{
		Name:         name,
		Type:         providerType,
		ChatProvider: provider,
		Queries:      s.defaultQueries,
	}
}

func (s *service) RemoveProvider(name string) {
	delete(s.providers, name)
}

func (s *service) UpdateProvider(name string, providerQueries map[string]*QueryGroups) error {
	if s.providers[name] != nil {
		s.providers[name].Queries = providerQueries
	} else {
		return fmt.Errorf("no provider found with name %s", name)
	}
	return nil
}

func (s *service) PilferAll() map[string]error {
	providerErrors := make(map[string]error)
	for pName, provider := range s.providers {
		providerOutputDir := path.Join(s.outputFolder, provider.Type.ToString(), provider.Name)
		if err := os.MkdirAll(providerOutputDir, 0750); err != nil {
			providerErrors[provider.Name] = err
			continue
		}

		if err := provider.ChatProvider.Pilfer(providerOutputDir, provider.Queries); err != nil {
			providerErrors[pName] = err
			continue
		}
	}
	return providerErrors
}

// write the output to a file for future reference
func WriteOutput(input any, outputFile *os.File) error {
	encoder := json.NewEncoder(outputFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(input); err != nil {
		return err
	}

	return nil
}
