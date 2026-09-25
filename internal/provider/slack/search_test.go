package slack

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const searchTest = `{
    "ok": true,
    "query": "vimrc*",
    "messages": {
        "total": 3,
        "pagination": {
            "total_count": 3,
            "page": 2,
            "per_page": 1,
            "page_count": 3,
            "first": 2,
            "last": 2
        },
        "paging": {
            "count": 1,
            "total": 3,
            "page": 2,
            "pages": 3
        },
        "matches": [
            {
                "team": "T0C2VF5DB8Q",
                "channel": {
                    "id": "C0C2VF5TQSU",
                },
                "user": "U0C1KP6TMQX",
                "ts": "1789708112.273789",
                "files": [
                    {
                        "id": "F0C3LSQS6AU",
                        "mimetype": "text/plain",
                        "size": 1272,
                        "name": "vimrc",
                        "title": "vimrc",
                        "url_private_download": "https://files.slack.com/files-pri/T0C2VF5DB8Q-F0C3LSQS6AU/download/vimrc"
                    }
                ],
                "text": "here is my vimrc when it isn't pinned",
                "permalink": "https://chatpilfer.slack.com/archives/C0C2VF5TQSU/p1789708112273789",
            }
        ]
    }
}`

const workspaceUsersOK = `{
    "ok": true,
    "members": [
        {
            "id": "U1",
            "name": "johndoe",
            "team_id": "T1",
            "real_name": "Johne Doe",
            "is_admin": true,
            "is_owner": true,
            "profile": {
                "email": "u1@chatpilfer.com"
            }
        }
    ],
    "response_metadata": {
        "next_cursor": ""
    }
}`

const workspaceUsersCursor1 = `{
    "ok": true,
    "offset": "U1",
    "members": [
        {
            "id": "U0",
            "name": "slackbot",
            "team_id": "T1",
            "real_name": "Slackbot",
            "is_admin": true,
            "is_owner": false,
            "profile": {
            }
        }
    ],
    "response_metadata": {
        "next_cursor": "dXNlcjpVMQ=="
    }
}`

const workspaceUsersCursor2 = `{
    "ok": true,
    "members": [
        {
            "id": "U1",
            "name": "johndoe",
            "team_id": "T1",
            "real_name": "Johne Doe",
            "is_admin": true,
            "is_owner": true,
            "profile": {
                "email": "u1@chatpilfer.com"
            }
        }
    ],
    "response_metadata": {
        "next_cursor": ""
    }
}`

const workspaceUsersMultiple = `{
    "ok": true,
    "members": [
        {
            "id": "U0",
            "name": "slackbot",
            "team_id": "T1",
            "real_name": "Slackbot",
            "is_admin": false,
            "is_owner": false,
            "profile": {
            }
        },
        {
            "id": "U1",
            "name": "johndoe",
            "team_id": "T1",
            "real_name": "John Doe",
            "is_admin": true,
            "is_owner": true,
            "profile": {
                "email": "u1@chatpilfer.com"
            }
        },
        {
            "id": "U2",
            "name": "alice",
            "team_id": "T1",
            "real_name": "Alice Real",
            "is_admin": false,
            "is_owner": false,
            "has_2fa": false,
            "profile": {
                "email": "u2@chatpilfer.com"
            }
        }
    ],
    "response_metadata": {
        "next_cursor": ""
    }
}`

func TestGetWorkspaceUsers(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		cursor    []string
		responses map[string]string
		expected  []User
		expectErr bool
	}{
		{
			name:      "basic test",
			cursor:    []string{""},
			responses: map[string]string{"": workspaceUsersOK},
			expected: []User{
				{
					UserID:   "U1",
					TeamID:   "T1",
					FullName: "Johne Doe",
					Username: "johndoe",
					IsAdmin:  true,
					IsOwner:  true,
					Has2FA:   false,
					Email:    "u1@chatpilfer.com",
				},
			},
			expectErr: false,
		},
		{
			name:      "limits and cursor",
			limit:     1,
			cursor:    []string{"", "dXNlcjpVMQ=="},
			responses: map[string]string{"": workspaceUsersCursor1, "dXNlcjpVMQ==": workspaceUsersCursor2},
			expected: []User{
				{
					UserID:   "U0",
					TeamID:   "T1",
					FullName: "Slackbot",
					Username: "slackbot",
					IsAdmin:  true,
					IsOwner:  false,
					Has2FA:   false,
				},
				{
					UserID:   "U1",
					TeamID:   "T1",
					FullName: "Johne Doe",
					Username: "johndoe",
					IsAdmin:  true,
					IsOwner:  true,
					Has2FA:   false,
					Email:    "u1@chatpilfer.com",
				},
			},
		},
		{
			name:      "multiple users in a response",
			cursor:    []string{""},
			responses: map[string]string{"": workspaceUsersMultiple},
			expected: []User{
				{
					UserID:   "U0",
					TeamID:   "T1",
					FullName: "Slackbot",
					Username: "slackbot",
					IsAdmin:  false,
					IsOwner:  false,
					Has2FA:   false,
				},
				{
					UserID:   "U1",
					TeamID:   "T1",
					FullName: "John Doe",
					Username: "johndoe",
					IsAdmin:  true,
					IsOwner:  true,
					Has2FA:   false,
					Email:    "u1@chatpilfer.com",
				},
				{
					UserID:   "U2",
					TeamID:   "T1",
					FullName: "Alice Real",
					Username: "alice",
					IsAdmin:  false,
					IsOwner:  false,
					Has2FA:   false,
					Email:    "u2@chatpilfer.com",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sp := &SlackSearchProvider{
				httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					cursor := req.URL.Query().Get("cursor")
					body, ok := test.responses[cursor]
					if !ok {
						t.Fatalf("unexpected cursor %s", cursor)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
						Header:     make(http.Header),
					}, nil
				})},
			}

			got, err := sp.GetWorkspaceUsers(test.limit)
			if test.expectErr {
				if err == nil {
					t.Fatalf("GetWorkspaceUsers() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetWorkspaceUsers() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("GetWorkspaceUsers() = %+v, want %+v", got, test.expected)
			}
		})
	}
}

const pinnedFileRespOK = `{
    "ok": true,
    "items": [
        {
            "created_by": "U1",
            "channel": "C1",
            "message": {
                "text": "test pinned file",
                "permalink": "https://chatpilfer.slack.com/archives/C1/P1",
                "files": [
                    {
                        "id": "F1",
                        "filetype": "text",
                        "url_private_download": "https://files.slack.com/files-pri/team-file/downloa/test"
                    }
                ]
            }
        }
    ]
}`

const pinnedTextRespOK = `{
    "ok": true,
    "items": [
        {
            "created_by": "U1",
            "channel": "C2",
            "message": {
                "text": "test",
                "permalink": "https://chatpilfer.slack.com/archives/C2/P1"
            }
        }
    ]
}`

const pinnedMultipleRespOK = `{
	"ok": true,
	"items": [
        {
            "created_by": "U1",
            "channel": "C1",
            "message": {
                "text": "test pinned file",
                "permalink": "https://chatpilfer.slack.com/archives/C1/P1",
                "files": [
                    {
                        "id": "F1",
                        "filetype": "text",
                        "url_private_download": "https://files.slack.com/files-pri/team-file/downloa/test"
                    }
                ]
            }
        },
		{
            "created_by": "U1",
            "channel": "C1",
            "message": {
                "text": "test",
                "permalink": "https://chatpilfer.slack.com/archives/C1/P1"
            }
        }
    ]
}`

func TestGetPinnedChats(t *testing.T) {
	tests := []struct {
		name      string
		channels  []Channel
		responses map[string]string
		expected  []PinnedItem
		expectErr bool
	}{
		{
			name:      "No channels error",
			channels:  nil,
			expectErr: true,
		},
		{
			name:      "Simple Text Test",
			channels:  []Channel{{ID: "C2"}},
			responses: map[string]string{"C2": pinnedTextRespOK},
			expected: []PinnedItem{
				{
					CreatedBy: "U1",
					Channel:   "C2",
					Message: PinnedMessage{
						Text:      "test",
						Permalink: "https://chatpilfer.slack.com/archives/C2/P1",
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "Single channel multiple items",
			channels:  []Channel{{ID: "C1"}},
			responses: map[string]string{"C1": pinnedMultipleRespOK},
			expected: []PinnedItem{
				{
					CreatedBy: "U1",
					Channel:   "C1",
					Message: PinnedMessage{
						Text:      "test pinned file",
						Permalink: "https://chatpilfer.slack.com/archives/C1/P1",
						Files: []File{
							{
								ID:          "F1",
								Filetype:    "text",
								DownloadUrl: "https://files.slack.com/files-pri/team-file/downloa/test",
							},
						},
					},
				},
				{
					CreatedBy: "U1",
					Channel:   "C1",
					Message: PinnedMessage{
						Text:      "test",
						Permalink: "https://chatpilfer.slack.com/archives/C1/P1",
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "Multiple channels",
			channels:  []Channel{{ID: "C1"}, {ID: "C2"}},
			responses: map[string]string{"C1": pinnedFileRespOK, "C2": pinnedTextRespOK},
			expected: []PinnedItem{
				{
					CreatedBy: "U1",
					Channel:   "C1",
					Message: PinnedMessage{
						Text:      "test pinned file",
						Permalink: "https://chatpilfer.slack.com/archives/C1/P1",
						Files: []File{
							{
								ID:          "F1",
								Filetype:    "text",
								DownloadUrl: "https://files.slack.com/files-pri/team-file/downloa/test",
							},
						},
					},
				},
				{
					CreatedBy: "U1",
					Channel:   "C2",
					Message: PinnedMessage{
						Text:      "test",
						Permalink: "https://chatpilfer.slack.com/archives/C2/P1",
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "One failing channel",
			channels:  []Channel{{ID: "C1"}, {ID: "C2"}},
			responses: map[string]string{"C1": pinnedFileRespOK, "C2": unauthenticatedResp},
			expected: []PinnedItem{
				{
					CreatedBy: "U1",
					Channel:   "C1",
					Message: PinnedMessage{
						Text:      "test pinned file",
						Permalink: "https://chatpilfer.slack.com/archives/C1/P1",
						Files: []File{
							{
								ID:          "F1",
								Filetype:    "text",
								DownloadUrl: "https://files.slack.com/files-pri/team-file/downloa/test",
							},
						},
					},
				},
				{
					Channel: "C2",
					Error:   "not_authed",
				},
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sp := &SlackSearchProvider{
				cache: &cache{
					channels: test.channels,
				},
				httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					channel := req.URL.Query().Get("channel")
					body, ok := test.responses[channel]
					if !ok {
						t.Fatalf("unexpected channel %s", channel)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
						Header:     make(http.Header),
					}, nil
				})},
			}

			got, err := sp.GetPinnedChats()
			if test.expectErr {
				if err == nil {
					t.Fatalf("GetPinnedChats() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetPinnedChats() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("GetPinnedChats() = %+v, want %+v", got, test.expected)
			}
		})
	}
}
