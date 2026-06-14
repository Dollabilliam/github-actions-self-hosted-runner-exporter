# Metric Contract

All metric names use the `github_actions` namespace and base units where units
exist. Labels are intentionally low-cardinality so the exporter works well with
Prometheus and Grafana.

## Dashboard Mapping

| Dashboard section | Primary metrics |
| --- | --- |
| Now running | `github_actions_workflow_run_latest_status`, `github_actions_job_running` |
| Last build | `github_actions_workflow_run_latest_status`, `github_actions_workflow_run_latest_info`, `github_actions_workflow_run_latest_duration_seconds` |
| Build health | `github_actions_workflow_runs_recent` |
| Runtime | `github_actions_workflow_run_recent_duration_seconds_avg`, `github_actions_job_recent_duration_seconds_avg` |
| Queue pressure | `github_actions_workflow_run_recent_queue_seconds_avg`, `github_actions_job_recent_queue_seconds_avg` |
| Runner fleet | `github_actions_runner_online`, `github_actions_runner_busy`, `github_actions_runner_info` |
| Exporter health | `github_actions_exporter_up`, `github_actions_exporter_last_refresh_timestamp_seconds`, `github_actions_exporter_refresh_errors_total` |

## Exporter Health

### `github_actions_exporter_up`

Gauge set to `1` when the most recent refresh succeeded and `0` when it failed.

Labels: none.

### `github_actions_exporter_last_refresh_timestamp_seconds`

Unix timestamp for the last successful refresh.

Labels: none.

### `github_actions_exporter_refresh_errors_total`

Counter of refresh failures.

Labels: none.

## Workflow Runs

### `github_actions_workflow_run_latest_status`

Gauge set to `1` for the latest observed workflow run status/conclusion per
repository and workflow.

Labels:

- `owner`
- `repo`
- `workflow`
- `status`
- `conclusion`

### `github_actions_workflow_run_latest_duration_seconds`

Gauge containing the latest completed workflow run duration.

Labels:

- `owner`
- `repo`
- `workflow`
- `conclusion`

### `github_actions_workflow_run_latest_info`

Info-style gauge set to `1` for the latest observed workflow run metadata. Use
`owner`, `repo`, and `run_id` to build dashboard links back to GitHub Actions.
This metric is intended for latest-run lookup tables, not historical
aggregation.

Labels:

- `owner`
- `repo`
- `workflow`
- `status`
- `conclusion`
- `run_id`
- `run_number`
- `run_attempt`
- `event`
- `head_branch`

### `github_actions_workflow_runs_recent`

Gauge count of recent workflow runs in the exporter sample.

Labels:

- `owner`
- `repo`
- `workflow`
- `status`
- `conclusion`

### `github_actions_workflow_run_recent_duration_seconds_avg`

Gauge containing average completed workflow run duration across the recent
sample.

Labels:

- `owner`
- `repo`
- `workflow`
- `conclusion`

### `github_actions_workflow_run_recent_queue_seconds_avg`

Gauge containing average workflow run queue time across the recent sample.

Labels:

- `owner`
- `repo`
- `workflow`

## Jobs

### `github_actions_job_running`

Gauge count of currently running jobs by repository, workflow, job name, and
runner.

Labels:

- `owner`
- `repo`
- `workflow`
- `job`
- `runner`

### `github_actions_job_recent_duration_seconds_avg`

Gauge containing average completed job duration across the recent sample.

Labels:

- `owner`
- `repo`
- `workflow`
- `job`
- `conclusion`

### `github_actions_job_recent_queue_seconds_avg`

Gauge containing average job queue time across the recent sample.

Labels:

- `owner`
- `repo`
- `workflow`
- `job`

## Runners

### `github_actions_runner_online`

Gauge set to `1` when GitHub reports the runner as online, otherwise `0`.

Labels:

- `owner`
- `repo`
- `runner`

### `github_actions_runner_busy`

Gauge set to `1` when GitHub reports the runner as busy, otherwise `0`.

Labels:

- `owner`
- `repo`
- `runner`

### `github_actions_runner_info`

Info-style gauge set to `1` with runner attributes.

Labels:

- `owner`
- `repo`
- `runner`
- `os`
- `arch`

## Deliberately Excluded From Prometheus Labels

These values are useful in logs or links, but should not become first-pass
Prometheus labels:

- job ID
- commit SHA
- actor
- pull request number
- full runner label set

They can be added later behind explicit config switches if a dashboard really
needs them.
