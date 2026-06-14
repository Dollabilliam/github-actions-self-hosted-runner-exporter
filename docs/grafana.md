# Grafana Dashboard

This repository includes an importable Grafana dashboard for the exporter:

`examples/grafana/dashboards/github-actions-overview.json`

The dashboard is intentionally datasource-agnostic. On import, choose your
Prometheus datasource. Choose a Loki datasource only if you also ship
self-hosted runner diagnostic logs.

## Prometheus Scrape

Scrape the exporter from Prometheus, Grafana Alloy, or another Prometheus
compatible collector:

```yaml
scrape_configs:
  - job_name: github-actions-exporter
    static_configs:
      - targets:
          - github-actions-exporter:9176
```

The dashboard uses these labels when present:

- `owner`
- `repo`
- `workflow`
- `runner`
- `host`

The `host` label is usually added by your scrape configuration rather than by
the exporter itself.

## Dashboard Semantics

The latest-status panels use `github_actions_workflow_run_latest_status`, so
they show one current/latest outcome per workflow.

Metrics with `recent` in their name, such as
`github_actions_workflow_run_recent_duration_seconds_avg`, are calculated from
the exporter's configured recent sample. By default this is the latest 20
workflow runs per configured repository.

The latest workflow runs table uses `github_actions_workflow_run_latest_info`
and adds a Grafana data link from `run_id` to the matching GitHub Actions run.

## Optional Loki Logs Panel

The logs panel expects runner diagnostic logs to be shipped separately to Loki.
Use labels like:

- `job="github-actions-runner"`
- `repo="<repository name>"`
- `runner="<runner name>"`
- `host="<runner host>"`

If you do not use Loki, the Prometheus panels still work; remove or ignore the
logs panel after importing the dashboard.

## Provisioning Example

`examples/grafana/provisioning/dashboards.yml` is a minimal Grafana
provisioning provider. Mount the dashboard JSON into the path configured there,
or adjust the path for your Grafana deployment.
