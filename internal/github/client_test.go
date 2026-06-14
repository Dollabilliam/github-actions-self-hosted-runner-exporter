package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListWorkflowRunsPaginatesAndSendsGitHubHeaders(t *testing.T) {
	var requests int

	client, err := NewClient(ClientConfig{
		BaseURL: "https://api.example.test",
		Token:   "test-token",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++

				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Fatalf("Authorization header = %q", r.Header.Get("Authorization"))
				}
				if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
					t.Fatalf("X-GitHub-Api-Version header = %q", r.Header.Get("X-GitHub-Api-Version"))
				}

				if r.URL.Path != "/repos/example-org/example-service/actions/runs" {
					t.Fatalf("path = %q", r.URL.Path)
				}

				body := workflowRunsResponse{
					WorkflowRuns: []WorkflowRun{{ID: 101}},
				}
				if r.URL.Query().Get("page") == "1" {
					body.WorkflowRuns = make([]WorkflowRun, 100)
					for i := range body.WorkflowRuns {
						body.WorkflowRuns[i] = WorkflowRun{ID: int64(i + 1)}
					}
				}

				raw, err := json.Marshal(body)
				if err != nil {
					t.Fatal(err)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(string(raw))),
					Header:     http.Header{},
					Request:    r,
				}, nil
			}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	runs, err := client.ListWorkflowRuns(context.Background(), Repository{Owner: "example-org", Name: "example-service"}, 101)
	if err != nil {
		t.Fatalf("ListWorkflowRuns() error = %v", err)
	}
	if len(runs) != 101 {
		t.Fatalf("len(runs) = %d", len(runs))
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
