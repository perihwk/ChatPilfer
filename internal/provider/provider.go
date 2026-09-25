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
)

func (pt *ProviderType) ToString() string {
	switch *pt {
	case Slack:
		return "slack"
	case Teams:
		return "teams"
	default:
		return ""
	}
}

type ChatProvider interface {
	Initialize() error
	SearchMessagesRegex(queries []string, regex string) (map[string][]any, error)
	SearchMessages(queries []string) (map[string][]any, error)
}

type Entry struct {
	Name    string
	Type    ProviderType
	Queries map[string]*QueryGroups
	ChatProvider
}

type service struct {
	defaultQueries map[string]*QueryGroups
	providers      map[string]*Entry
}

func NewService(defaultQueries map[string]*QueryGroups) *service {
	return &service{
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
		if err := os.MkdirAll(path.Join(provider.Type.ToString(), provider.Name), 0750); err != nil {
			providerErrors[provider.Name] = err
			continue
		}
		for _, queryGroup := range provider.Queries {
			fileCache, err := os.OpenFile(path.Join(provider.Type.ToString(), provider.Name, queryGroup.outputFile), os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				providerErrors[pName] = err
				break
			}

			queryResults, err := provider.ChatProvider.SearchMessages(queryGroup.queries)
			if err != nil {
				continue
			}

			s.writeOutput(queryResults, fileCache)
		}
	}
	return providerErrors
}

func (s *service) SearchCloudEndpoints(name string) (map[string][]any, error) {
	if s.providers[name] == nil {
		return nil, fmt.Errorf("no provider found with name %s", name)
	}
	entry := s.providers[name]

	if entry.Queries["cloud_endpoints"] == nil {
		return nil, fmt.Errorf("cloud endpoints queries are disabled for provider %s", name)
	}

	// create directory if it doesn't exist
	if err := os.MkdirAll(path.Join(entry.Type.ToString(), entry.Name), 0750); err != nil {
		return nil, err
	}

	fileCache, err := os.OpenFile(path.Join(entry.Type.ToString(), entry.Name, "cloud_endpoints.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	cloudEndpoints, err := entry.ChatProvider.SearchMessages(CloudQueries[:])
	if err != nil {
		return nil, err
	}

	s.writeOutput(cloudEndpoints, fileCache)
	return cloudEndpoints, nil
}

func (s *service) SearchPasswords(name string) (map[string][]any, error) {
	if s.providers[name] == nil {
		return nil, fmt.Errorf("no provider found with name %s", name)
	}
	entry := s.providers[name]

	if entry.Queries["credentials"] == nil {
		return nil, fmt.Errorf("cloud endpoints queries are disabled for provider %s", name)
	}

	// create directory if it doesn't exist
	if err := os.MkdirAll(path.Join(entry.Type.ToString(), entry.Name), 0750); err != nil {
		return nil, err
	}

	fileCache, err := os.OpenFile(path.Join(entry.Type.ToString(), entry.Name, "credentials.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	credentialResults, err := entry.SearchMessages(CredentialQueries[:])
	if err != nil {
		return nil, err
	}

	s.writeOutput(credentialResults, fileCache)
	return credentialResults, nil
}

func (s *service) SearchConnectionStrings(name string) (map[string][]any, error) {
	if s.providers[name] == nil {
		return nil, fmt.Errorf("no provider found with name %s", name)
	}
	entry := s.providers[name]

	if entry.Queries["connection-strings"] == nil {
		return nil, fmt.Errorf("cloud endpoints queries are disabled for provider %s", name)
	}

	// create directory if it doesn't exist
	if err := os.MkdirAll(path.Join(entry.Type.ToString(), entry.Name), 0750); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(entry.Type.ToString(), entry.Name), 0750); err != nil {
		return nil, err
	}

	fileCache, err := os.OpenFile(path.Join(entry.Type.ToString(), entry.Name, "connection_strings.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	connectionStrings, err := entry.SearchMessages(ConnectionStringQueries[:])
	if err != nil {
		return nil, err
	}

	s.writeOutput(connectionStrings, fileCache)
	return connectionStrings, nil
}

// write the output to a file for future reference
func (s *service) writeOutput(input any, outputFile *os.File) error {
	encoder := json.NewEncoder(outputFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(input); err != nil {
		return err
	}

	return nil
}
