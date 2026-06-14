package collector

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/config"
	gh "github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCollectorEmitsDashboardMetrics(t *testing.T) {
	createdAt := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	startedAt := createdAt.Add(30 * time.Second)
	updatedAt := startedAt.Add(2 * time.Minute)
	jobStartedAt := startedAt.Add(10 * time.Second)
	jobCompletedAt := jobStartedAt.Add(90 * time.Second)

	client := &fakeClient{
		runners: []gh.Runner{
			{Name: "runner-linux-1", OS: "linux", Arch: "X64", Status: "online", Busy: true},
		},
		runs: []gh.WorkflowRun{
			{
				ID:           42,
				RunNumber:    7,
				RunAttempt:   1,
				Name:         "lint",
				Event:        "push",
				HeadBranch:   "main",
				Status:       "completed",
				Conclusion:   "success",
				CreatedAt:    createdAt,
				RunStartedAt: &startedAt,
				UpdatedAt:    updatedAt,
			},
		},
		jobs: map[int64][]gh.WorkflowJob{
			42: {
				{
					ID:          100,
					RunID:       42,
					Name:        "go-test",
					Status:      "completed",
					Conclusion:  "success",
					StartedAt:   &jobStartedAt,
					CompletedAt: &jobCompletedAt,
					RunnerName:  "runner-linux-1",
				},
			},
		},
	}

	collector := New(Config{
		Client: client,
		Repositories: []config.Repository{
			{Owner: "example-org", Name: "example-service"},
		},
		RefreshInterval:   time.Hour,
		RunsPerRepo:       20,
		CollectionTimeout: time.Second,
	})
	collector.Refresh(context.Background())

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	expected := `
# HELP github_actions_exporter_up Whether the most recent GitHub Actions exporter refresh succeeded.
# TYPE github_actions_exporter_up gauge
github_actions_exporter_up 1
# HELP github_actions_job_recent_duration_seconds_avg Average completed GitHub Actions job duration across the recent exporter sample.
# TYPE github_actions_job_recent_duration_seconds_avg gauge
github_actions_job_recent_duration_seconds_avg{conclusion="success",job="go-test",owner="example-org",repo="example-service",workflow="lint"} 90
# HELP github_actions_job_recent_queue_seconds_avg Average GitHub Actions job queue time across the recent exporter sample.
# TYPE github_actions_job_recent_queue_seconds_avg gauge
github_actions_job_recent_queue_seconds_avg{job="go-test",owner="example-org",repo="example-service",workflow="lint"} 40
# HELP github_actions_runner_busy Whether GitHub reports the self-hosted runner as busy.
# TYPE github_actions_runner_busy gauge
github_actions_runner_busy{owner="example-org",repo="example-service",runner="runner-linux-1"} 1
# HELP github_actions_runner_online Whether GitHub reports the self-hosted runner as online.
# TYPE github_actions_runner_online gauge
github_actions_runner_online{owner="example-org",repo="example-service",runner="runner-linux-1"} 1
# HELP github_actions_workflow_run_latest_duration_seconds Duration of the latest completed workflow run.
# TYPE github_actions_workflow_run_latest_duration_seconds gauge
github_actions_workflow_run_latest_duration_seconds{conclusion="success",owner="example-org",repo="example-service",workflow="lint"} 120
# HELP github_actions_workflow_run_latest_info Info-style metadata for the latest observed workflow run.
# TYPE github_actions_workflow_run_latest_info gauge
github_actions_workflow_run_latest_info{conclusion="success",event="push",head_branch="main",owner="example-org",repo="example-service",run_attempt="1",run_id="42",run_number="7",status="completed",workflow="lint"} 1
# HELP github_actions_workflow_run_latest_status Latest observed workflow run status and conclusion per repository and workflow.
# TYPE github_actions_workflow_run_latest_status gauge
github_actions_workflow_run_latest_status{conclusion="success",owner="example-org",repo="example-service",status="completed",workflow="lint"} 1
# HELP github_actions_workflow_run_recent_duration_seconds_avg Average completed workflow run duration across the recent exporter sample.
# TYPE github_actions_workflow_run_recent_duration_seconds_avg gauge
github_actions_workflow_run_recent_duration_seconds_avg{conclusion="success",owner="example-org",repo="example-service",workflow="lint"} 120
# HELP github_actions_workflow_run_recent_queue_seconds_avg Average workflow run queue time across the recent exporter sample.
# TYPE github_actions_workflow_run_recent_queue_seconds_avg gauge
github_actions_workflow_run_recent_queue_seconds_avg{owner="example-org",repo="example-service",workflow="lint"} 30
# HELP github_actions_workflow_runs_recent Count of recent workflow runs in the exporter sample.
# TYPE github_actions_workflow_runs_recent gauge
github_actions_workflow_runs_recent{conclusion="success",owner="example-org",repo="example-service",status="completed",workflow="lint"} 1
`

	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected),
		"github_actions_exporter_up",
		"github_actions_job_recent_duration_seconds_avg",
		"github_actions_job_recent_queue_seconds_avg",
		"github_actions_runner_busy",
		"github_actions_runner_online",
		"github_actions_workflow_run_latest_duration_seconds",
		"github_actions_workflow_run_latest_info",
		"github_actions_workflow_run_latest_status",
		"github_actions_workflow_run_recent_duration_seconds_avg",
		"github_actions_workflow_run_recent_queue_seconds_avg",
		"github_actions_workflow_runs_recent",
	); err != nil {
		t.Fatal(err)
	}
}

type fakeClient struct {
	runs    []gh.WorkflowRun
	jobs    map[int64][]gh.WorkflowJob
	runners []gh.Runner
}

func (f *fakeClient) ListWorkflowRuns(context.Context, gh.Repository, int) ([]gh.WorkflowRun, error) {
	return f.runs, nil
}

func (f *fakeClient) ListWorkflowJobs(_ context.Context, _ gh.Repository, runID int64) ([]gh.WorkflowJob, error) {
	return f.jobs[runID], nil
}

func (f *fakeClient) ListRunners(context.Context, gh.Repository) ([]gh.Runner, error) {
	return f.runners, nil
}
