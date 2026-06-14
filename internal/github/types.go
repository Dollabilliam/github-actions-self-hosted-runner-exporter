package github

import "time"

type Repository struct {
	Owner string
	Name  string
}

type WorkflowRun struct {
	ID           int64      `json:"id"`
	RunNumber    int64      `json:"run_number"`
	RunAttempt   int64      `json:"run_attempt"`
	Name         string     `json:"name"`
	DisplayTitle string     `json:"display_title"`
	Event        string     `json:"event"`
	HeadBranch   string     `json:"head_branch"`
	Status       string     `json:"status"`
	Conclusion   string     `json:"conclusion"`
	CreatedAt    time.Time  `json:"created_at"`
	RunStartedAt *time.Time `json:"run_started_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type workflowRunsResponse struct {
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

type WorkflowJob struct {
	ID              int64      `json:"id"`
	RunID           int64      `json:"run_id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	Conclusion      string     `json:"conclusion"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	RunnerID        int64      `json:"runner_id"`
	RunnerName      string     `json:"runner_name"`
	RunnerGroupName string     `json:"runner_group_name"`
}

type workflowJobsResponse struct {
	Jobs []WorkflowJob `json:"jobs"`
}

type Runner struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	OS     string `json:"os"`
	Arch   string `json:"architecture"`
	Status string `json:"status"`
	Busy   bool   `json:"busy"`
}

type runnersResponse struct {
	Runners []Runner `json:"runners"`
}
