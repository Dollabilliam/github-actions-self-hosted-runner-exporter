# Security

## Reporting

Please do not include tokens, private repository names, runner hostnames, or log
excerpts with secrets in public issues. If you believe you found a vulnerability,
open a private vulnerability report for this repository if available, or contact
the repository owner privately before publishing details.

## Operational Notes

The exporter does not require write access to GitHub. Use the least-privileged
token that can read Actions workflow, job, and self-hosted runner metadata for
the repositories you configure.

Keep `/metrics` and `/healthz` on an internal network. Exported labels can reveal
repository names, workflow names, runner names, build status, queue time, and
runtime information.

Runner diagnostic logs are not collected by this exporter. If you ship runner
logs to Loki or another log backend, scrub secrets at the collector or logging
pipeline and keep that backend access-controlled.
