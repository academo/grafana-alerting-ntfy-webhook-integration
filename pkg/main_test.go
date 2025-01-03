package main

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestValidateFlags(t *testing.T) {
	tests := map[string]struct {
		url     string
		wantErr bool
	}{
		"valid url":     {"https://ntfy.sh/topic", false},
		"empty url":     {"", true},
		"invalid url":   {"not-a-url", true},
		"no topic":      {"https://ntfy.sh/", true},
		"invalid chars": {"https://ntfy.sh/topic$%^", true},
		"custom server": {"https://custom.ntfy/mytopic", false},
		"http url":      {"http://ntfy.sh/topic", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			*ntfyUrl = tc.url
			err := validateFlags()
			if (err != nil) != tc.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestPrepareNotification(t *testing.T) {
	tests := map[string]struct {
		input AlertsPayload
		want  NtfyNotification
	}{
		"basic notification": {
			input: AlertsPayload{
				Message:     "test message",
				Title:       "test title",
				ExternalURL: "http://grafana/alert",
				Alerts:      []Alert{{SilenceURL: "http://grafana/silence"}},
			},
			want: NtfyNotification{
				Message: "test message",
				Title:   "test title",
				Topic:   "test-topic",
				Actions: []NtfyAction{
					{Action: "view", Label: "Open in Grafana", Url: "http://grafana/alert", Clear: true},
					{Action: "view", Label: "Silence", Url: "http://grafana/silence", Clear: false},
				},
			},
		},
		"empty alerts": {
			input: AlertsPayload{
				Message: "msg",
				Alerts:  []Alert{},
			},
			want: NtfyNotification{
				Message: "msg",
				Topic:   "test-topic",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			topic = "test-topic"
			got := prepareNotification(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("prepareNotification() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSendNotification(t *testing.T) {
	originalUsername := *username
	originalPassword := *password

	tests := map[string]struct {
		client     *mockHttpClient
		authHeader string
		username   string
		password   string
		wantErr    bool
		checkReq   func(*testing.T, *http.Request)
	}{
		"success": {
			client: &mockHttpClient{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(nil)),
				},
			},
			checkReq: func(t *testing.T, req *http.Request) {
				if auth := req.Header.Get("Authorization"); auth != "" {
					t.Errorf("unexpected Authorization header: %v", auth)
				}
			},
		},
		"with auth header": {
			client: &mockHttpClient{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(nil)),
				},
			},
			authHeader: "Bearer token",
			checkReq: func(t *testing.T, req *http.Request) {
				if auth := req.Header.Get("Authorization"); auth != "Bearer token" {
					t.Errorf("expected Authorization header %q, got %q", "Bearer token", auth)
				}
			},
		},
		"with basic auth": {
			client: &mockHttpClient{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(nil)),
				},
			},
			username: "user",
			password: "pass",
			checkReq: func(t *testing.T, req *http.Request) {
				username, password, ok := req.BasicAuth()
				if !ok {
					t.Error("expected basic auth, got none")
				}
				if username != "user" || password != "pass" {
					t.Errorf("expected basic auth user/pass, got %q/%q", username, password)
				}
			},
		},
		"non-200 response": {
			client: &mockHttpClient{
				response: &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewBuffer(nil)),
				},
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			*username = tc.username
			*password = tc.password

			var capturedReq *http.Request
			mockClient := &mockHttpClient{
				response: tc.client.response,
				err:      tc.client.err,
				onDo: func(req *http.Request) {
					capturedReq = req
				},
			}

			payload := NtfyNotification{Message: "test"}
			err := sendNotification(payload, tc.authHeader, mockClient)

			if (err != nil) != tc.wantErr {
				t.Errorf("sendNotification() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.checkReq != nil && capturedReq != nil {
				tc.checkReq(t, capturedReq)
			}
		})
	}

	// Restore original values
	*username = originalUsername
	*password = originalPassword
}

// Update mockHttpClient to capture request
type mockHttpClient struct {
	response *http.Response
	err      error
	onDo     func(*http.Request)
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	if m.onDo != nil {
		m.onDo(req)
	}
	return m.response, m.err
}
