package slack

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

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

type testTransport struct {
	responses map[string]string
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	channel := req.URL.Query().Get("channel")
	body := tt.responses[channel]
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func newTestProvider(channels []Channel, responses map[string]string) *SlackSearchProvider {
	return &SlackSearchProvider{
		cache: &cache{
			channels: channels,
		},
		httpClient: &http.Client{Transport: &testTransport{responses: responses}},
	}
}

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
			sp := newTestProvider(test.channels, test.responses)

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
