package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

type ClientConfig struct {
	BaseURL    string
	Token      string
	Timeout    time.Duration
	UserAgent  string
	HTTPClient *http.Client
}

type Client struct {
	baseURL   *url.URL
	token     string
	http      *http.Client
	userAgent string
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.github.com"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "github-actions-self-hosted-runner-exporter"
	}

	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &Client{
		baseURL:   baseURL,
		token:     cfg.Token,
		http:      httpClient,
		userAgent: cfg.UserAgent,
	}, nil
}

func (c *Client) ListWorkflowRuns(ctx context.Context, repo Repository, limit int) ([]WorkflowRun, error) {
	if limit <= 0 {
		limit = 20
	}

	var out []WorkflowRun
	page := 1
	for len(out) < limit {
		perPage := min(100, limit-len(out))
		var response workflowRunsResponse
		if err := c.get(ctx, repoPath(repo, "actions/runs"), pageValues(page, perPage), &response); err != nil {
			return nil, err
		}
		out = append(out, response.WorkflowRuns...)
		if len(response.WorkflowRuns) < perPage {
			break
		}
		page++
	}

	return out, nil
}

func (c *Client) ListWorkflowJobs(ctx context.Context, repo Repository, runID int64) ([]WorkflowJob, error) {
	var out []WorkflowJob
	page := 1
	for {
		var response workflowJobsResponse
		relativePath := repoPath(repo, "actions/runs/"+strconv.FormatInt(runID, 10)+"/jobs")
		if err := c.get(ctx, relativePath, pageValues(page, 100), &response); err != nil {
			return nil, err
		}
		out = append(out, response.Jobs...)
		if len(response.Jobs) < 100 {
			break
		}
		page++
	}

	return out, nil
}

func (c *Client) ListRunners(ctx context.Context, repo Repository) ([]Runner, error) {
	var out []Runner
	page := 1
	for {
		var response runnersResponse
		if err := c.get(ctx, repoPath(repo, "actions/runners"), pageValues(page, 100), &response); err != nil {
			return nil, err
		}
		out = append(out, response.Runners...)
		if len(response.Runners) < 100 {
			break
		}
		page++
	}

	return out, nil
}

func (c *Client) get(ctx context.Context, relativePath string, values url.Values, target any) error {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, relativePath)
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github api %s returned %s", endpoint.Path, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}

	return nil
}

func repoPath(repo Repository, suffix string) string {
	return path.Join("repos", repo.Owner, repo.Name, suffix)
}

func pageValues(page int, perPage int) url.Values {
	values := url.Values{}
	values.Set("page", strconv.Itoa(page))
	values.Set("per_page", strconv.Itoa(perPage))
	return values
}
