package collector

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/config"
	gh "github.com/Dollabilliam/github-actions-self-hosted-runner-exporter/internal/github"
	"github.com/prometheus/client_golang/prometheus"
)

type GitHubClient interface {
	ListWorkflowRuns(ctx context.Context, repo gh.Repository, limit int) ([]gh.WorkflowRun, error)
	ListWorkflowJobs(ctx context.Context, repo gh.Repository, runID int64) ([]gh.WorkflowJob, error)
	ListRunners(ctx context.Context, repo gh.Repository) ([]gh.Runner, error)
}

type Config struct {
	Client            GitHubClient
	Repositories      []config.Repository
	RefreshInterval   time.Duration
	RunsPerRepo       int
	Logger            *slog.Logger
	CollectionTimeout time.Duration
}

type Collector struct {
	client            GitHubClient
	repositories      []config.Repository
	refreshInterval   time.Duration
	runsPerRepo       int
	logger            *slog.Logger
	collectionTimeout time.Duration

	mu          sync.Mutex
	refreshMu   sync.Mutex
	snapshot    snapshot
	lastRefresh time.Time
	lastSuccess bool
	errorCount  float64

	exporterUp           *prometheus.Desc
	lastRefreshTimestamp *prometheus.Desc
	refreshErrors        *prometheus.Desc
	runnerOnline         *prometheus.Desc
	runnerBusy           *prometheus.Desc
	runnerInfo           *prometheus.Desc
	latestRunStatus      *prometheus.Desc
	latestRunInfo        *prometheus.Desc
	latestRunDuration    *prometheus.Desc
	recentRuns           *prometheus.Desc
	recentRunDuration    *prometheus.Desc
	recentRunQueue       *prometheus.Desc
	jobRunning           *prometheus.Desc
	jobDuration          *prometheus.Desc
	jobQueue             *prometheus.Desc
}

func New(cfg Config) *Collector {
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = time.Minute
	}
	if cfg.RunsPerRepo <= 0 {
		cfg.RunsPerRepo = 20
	}
	if cfg.CollectionTimeout <= 0 {
		cfg.CollectionTimeout = 2 * time.Minute
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Collector{
		client:            cfg.Client,
		repositories:      cfg.Repositories,
		refreshInterval:   cfg.RefreshInterval,
		runsPerRepo:       cfg.RunsPerRepo,
		logger:            cfg.Logger,
		collectionTimeout: cfg.CollectionTimeout,

		exporterUp: prometheus.NewDesc(
			"github_actions_exporter_up",
			"Whether the most recent GitHub Actions exporter refresh succeeded.",
			nil,
			nil,
		),
		lastRefreshTimestamp: prometheus.NewDesc(
			"github_actions_exporter_last_refresh_timestamp_seconds",
			"Unix timestamp of the last successful GitHub Actions exporter refresh.",
			nil,
			nil,
		),
		refreshErrors: prometheus.NewDesc(
			"github_actions_exporter_refresh_errors_total",
			"Total number of failed GitHub Actions exporter refreshes.",
			nil,
			nil,
		),
		runnerOnline: prometheus.NewDesc(
			"github_actions_runner_online",
			"Whether GitHub reports the self-hosted runner as online.",
			[]string{"owner", "repo", "runner"},
			nil,
		),
		runnerBusy: prometheus.NewDesc(
			"github_actions_runner_busy",
			"Whether GitHub reports the self-hosted runner as busy.",
			[]string{"owner", "repo", "runner"},
			nil,
		),
		runnerInfo: prometheus.NewDesc(
			"github_actions_runner_info",
			"Self-hosted runner attributes.",
			[]string{"owner", "repo", "runner", "os", "arch"},
			nil,
		),
		latestRunStatus: prometheus.NewDesc(
			"github_actions_workflow_run_latest_status",
			"Latest observed workflow run status and conclusion per repository and workflow.",
			[]string{"owner", "repo", "workflow", "status", "conclusion"},
			nil,
		),
		latestRunInfo: prometheus.NewDesc(
			"github_actions_workflow_run_latest_info",
			"Info-style metadata for the latest observed workflow run.",
			[]string{"owner", "repo", "workflow", "status", "conclusion", "run_id", "run_number", "run_attempt", "event", "head_branch"},
			nil,
		),
		latestRunDuration: prometheus.NewDesc(
			"github_actions_workflow_run_latest_duration_seconds",
			"Duration of the latest completed workflow run.",
			[]string{"owner", "repo", "workflow", "conclusion"},
			nil,
		),
		recentRuns: prometheus.NewDesc(
			"github_actions_workflow_runs_recent",
			"Count of recent workflow runs in the exporter sample.",
			[]string{"owner", "repo", "workflow", "status", "conclusion"},
			nil,
		),
		recentRunDuration: prometheus.NewDesc(
			"github_actions_workflow_run_recent_duration_seconds_avg",
			"Average completed workflow run duration across the recent exporter sample.",
			[]string{"owner", "repo", "workflow", "conclusion"},
			nil,
		),
		recentRunQueue: prometheus.NewDesc(
			"github_actions_workflow_run_recent_queue_seconds_avg",
			"Average workflow run queue time across the recent exporter sample.",
			[]string{"owner", "repo", "workflow"},
			nil,
		),
		jobRunning: prometheus.NewDesc(
			"github_actions_job_running",
			"Currently running GitHub Actions jobs.",
			[]string{"owner", "repo", "workflow", "job", "runner"},
			nil,
		),
		jobDuration: prometheus.NewDesc(
			"github_actions_job_recent_duration_seconds_avg",
			"Average completed GitHub Actions job duration across the recent exporter sample.",
			[]string{"owner", "repo", "workflow", "job", "conclusion"},
			nil,
		),
		jobQueue: prometheus.NewDesc(
			"github_actions_job_recent_queue_seconds_avg",
			"Average GitHub Actions job queue time across the recent exporter sample.",
			[]string{"owner", "repo", "workflow", "job"},
			nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.exporterUp
	ch <- c.lastRefreshTimestamp
	ch <- c.refreshErrors
	ch <- c.runnerOnline
	ch <- c.runnerBusy
	ch <- c.runnerInfo
	ch <- c.latestRunStatus
	ch <- c.latestRunInfo
	ch <- c.latestRunDuration
	ch <- c.recentRuns
	ch <- c.recentRunDuration
	ch <- c.recentRunQueue
	ch <- c.jobRunning
	ch <- c.jobDuration
	ch <- c.jobQueue
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	snapshot, up, lastRefresh, errorCount := c.currentSnapshot()

	ch <- prometheus.MustNewConstMetric(c.exporterUp, prometheus.GaugeValue, boolValue(up))
	ch <- prometheus.MustNewConstMetric(c.refreshErrors, prometheus.CounterValue, errorCount)
	if !lastRefresh.IsZero() {
		ch <- prometheus.MustNewConstMetric(c.lastRefreshTimestamp, prometheus.GaugeValue, float64(lastRefresh.Unix()))
	}

	emitSnapshot(ch, c, snapshot)
}

func (c *Collector) Start(ctx context.Context) {
	go func() {
		c.Refresh(ctx)

		ticker := time.NewTicker(c.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.Refresh(ctx)
			}
		}
	}()
}

func (c *Collector) Refresh(ctx context.Context) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	refreshCtx, cancel := context.WithTimeout(ctx, c.collectionTimeout)
	defer cancel()

	next, err := c.refresh(refreshCtx)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.errorCount++
		c.lastSuccess = false
		c.logger.Warn("refresh failed", "err", err)
		return
	}

	c.snapshot = next
	c.lastRefresh = next.refreshedAt
	c.lastSuccess = true
	c.logger.Info("refresh completed", "repositories", len(next.repositories))
}

func (c *Collector) currentSnapshot() (snapshot, bool, time.Time, float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.snapshot, c.lastSuccess, c.lastRefresh, c.errorCount
}

func (c *Collector) refresh(ctx context.Context) (snapshot, error) {
	next := snapshot{refreshedAt: time.Now()}
	var reposWithRuns int

	for _, configuredRepo := range c.repositories {
		repo := gh.Repository{Owner: configuredRepo.Owner, Name: configuredRepo.Name}
		repoSnapshot := repositorySnapshot{repo: repo}

		runners, err := c.client.ListRunners(ctx, repo)
		if err != nil {
			c.logger.Warn("list runners failed", "owner", repo.Owner, "repo", repo.Name, "err", err)
		} else {
			repoSnapshot.runners = runners
		}

		runs, err := c.client.ListWorkflowRuns(ctx, repo, c.runsPerRepo)
		if err != nil {
			c.logger.Warn("list workflow runs failed", "owner", repo.Owner, "repo", repo.Name, "err", err)
			next.repositories = append(next.repositories, repoSnapshot)
			continue
		}

		reposWithRuns++
		requiredJobs := requiredJobRunIDs(runs)
		for _, run := range runs {
			runSnapshot := runSnapshot{run: run}
			if requiredJobs[run.ID] {
				jobs, err := c.client.ListWorkflowJobs(ctx, repo, run.ID)
				if err != nil {
					c.logger.Warn("list workflow jobs failed", "owner", repo.Owner, "repo", repo.Name, "run_id", run.ID, "err", err)
				} else {
					runSnapshot.jobs = jobs
				}
			}
			repoSnapshot.runs = append(repoSnapshot.runs, runSnapshot)
		}

		next.repositories = append(next.repositories, repoSnapshot)
	}

	if reposWithRuns == 0 {
		return snapshot{}, errors.New("no repositories returned workflow runs")
	}

	return next, nil
}

func requiredJobRunIDs(runs []gh.WorkflowRun) map[int64]bool {
	required := map[int64]bool{}
	latestByWorkflow := map[string]bool{}
	latestCompletedByWorkflow := map[string]bool{}

	for _, run := range runs {
		workflow := workflowName(run)
		if !latestByWorkflow[workflow] {
			required[run.ID] = true
			latestByWorkflow[workflow] = true
		}
		if run.Status == "completed" && !latestCompletedByWorkflow[workflow] {
			required[run.ID] = true
			latestCompletedByWorkflow[workflow] = true
		}
		if run.Status != "" && run.Status != "completed" {
			required[run.ID] = true
		}
	}

	return required
}

func emitSnapshot(ch chan<- prometheus.Metric, c *Collector, snapshot snapshot) {
	for _, repoSnapshot := range snapshot.repositories {
		owner := repoSnapshot.repo.Owner
		repoName := repoSnapshot.repo.Name

		for _, runner := range repoSnapshot.runners {
			ch <- prometheus.MustNewConstMetric(c.runnerOnline, prometheus.GaugeValue, boolValue(runner.Status == "online"), owner, repoName, runner.Name)
			ch <- prometheus.MustNewConstMetric(c.runnerBusy, prometheus.GaugeValue, boolValue(runner.Busy), owner, repoName, runner.Name)
			ch <- prometheus.MustNewConstMetric(c.runnerInfo, prometheus.GaugeValue, 1, owner, repoName, runner.Name, runner.OS, runner.Arch)
		}

		emitWorkflowMetrics(ch, c, repoSnapshot)
		emitJobMetrics(ch, c, repoSnapshot)
	}
}

func emitWorkflowMetrics(ch chan<- prometheus.Metric, c *Collector, repoSnapshot repositorySnapshot) {
	owner := repoSnapshot.repo.Owner
	repoName := repoSnapshot.repo.Name

	latestByWorkflow := map[string]gh.WorkflowRun{}
	runCounts := map[workflowCountKey]float64{}
	durationGroups := map[workflowDurationKey]averager{}
	queueGroups := map[workflowQueueKey]averager{}

	for _, item := range repoSnapshot.runs {
		run := item.run
		workflow := workflowName(run)
		if _, exists := latestByWorkflow[workflow]; !exists {
			latestByWorkflow[workflow] = run
		}

		runCounts[workflowCountKey{
			workflow:   workflow,
			status:     valueOrUnknown(run.Status),
			conclusion: valueOrNone(run.Conclusion),
		}]++

		if duration, ok := workflowDuration(run); ok {
			durationGroups[workflowDurationKey{
				workflow:   workflow,
				conclusion: valueOrNone(run.Conclusion),
			}] = durationGroups[workflowDurationKey{
				workflow:   workflow,
				conclusion: valueOrNone(run.Conclusion),
			}].with(duration.Seconds())
		}

		if queue, ok := workflowQueue(run); ok {
			queueGroups[workflowQueueKey{workflow: workflow}] = queueGroups[workflowQueueKey{workflow: workflow}].with(queue.Seconds())
		}
	}

	for workflow, run := range latestByWorkflow {
		ch <- prometheus.MustNewConstMetric(
			c.latestRunStatus,
			prometheus.GaugeValue,
			1,
			owner,
			repoName,
			workflow,
			valueOrUnknown(run.Status),
			valueOrNone(run.Conclusion),
		)
		ch <- prometheus.MustNewConstMetric(
			c.latestRunInfo,
			prometheus.GaugeValue,
			1,
			owner,
			repoName,
			workflow,
			valueOrUnknown(run.Status),
			valueOrNone(run.Conclusion),
			strconv.FormatInt(run.ID, 10),
			strconv.FormatInt(run.RunNumber, 10),
			strconv.FormatInt(run.RunAttempt, 10),
			valueOrNone(run.Event),
			valueOrNone(run.HeadBranch),
		)
		if duration, ok := workflowDuration(run); ok {
			ch <- prometheus.MustNewConstMetric(c.latestRunDuration, prometheus.GaugeValue, duration.Seconds(), owner, repoName, workflow, valueOrNone(run.Conclusion))
		}
	}

	for _, key := range sortedWorkflowCountKeys(runCounts) {
		ch <- prometheus.MustNewConstMetric(c.recentRuns, prometheus.GaugeValue, runCounts[key], owner, repoName, key.workflow, key.status, key.conclusion)
	}

	for _, key := range sortedWorkflowDurationKeys(durationGroups) {
		ch <- prometheus.MustNewConstMetric(c.recentRunDuration, prometheus.GaugeValue, durationGroups[key].avg(), owner, repoName, key.workflow, key.conclusion)
	}

	for _, key := range sortedWorkflowQueueKeys(queueGroups) {
		ch <- prometheus.MustNewConstMetric(c.recentRunQueue, prometheus.GaugeValue, queueGroups[key].avg(), owner, repoName, key.workflow)
	}
}

func emitJobMetrics(ch chan<- prometheus.Metric, c *Collector, repoSnapshot repositorySnapshot) {
	owner := repoSnapshot.repo.Owner
	repoName := repoSnapshot.repo.Name
	durationGroups := map[jobDurationKey]averager{}
	queueGroups := map[jobQueueKey]averager{}

	for _, item := range repoSnapshot.runs {
		workflow := workflowName(item.run)

		for _, job := range item.jobs {
			jobName := valueOrUnknown(job.Name)
			if job.Status == "in_progress" {
				ch <- prometheus.MustNewConstMetric(c.jobRunning, prometheus.GaugeValue, 1, owner, repoName, workflow, jobName, valueOrNone(job.RunnerName))
			}

			if duration, ok := jobDuration(job); ok {
				durationGroups[jobDurationKey{
					workflow:   workflow,
					job:        jobName,
					conclusion: valueOrNone(job.Conclusion),
				}] = durationGroups[jobDurationKey{
					workflow:   workflow,
					job:        jobName,
					conclusion: valueOrNone(job.Conclusion),
				}].with(duration.Seconds())
			}

			if queue, ok := jobQueue(item.run, job); ok {
				queueGroups[jobQueueKey{
					workflow: workflow,
					job:      jobName,
				}] = queueGroups[jobQueueKey{
					workflow: workflow,
					job:      jobName,
				}].with(queue.Seconds())
			}
		}
	}

	for _, key := range sortedJobDurationKeys(durationGroups) {
		ch <- prometheus.MustNewConstMetric(c.jobDuration, prometheus.GaugeValue, durationGroups[key].avg(), owner, repoName, key.workflow, key.job, key.conclusion)
	}

	for _, key := range sortedJobQueueKeys(queueGroups) {
		ch <- prometheus.MustNewConstMetric(c.jobQueue, prometheus.GaugeValue, queueGroups[key].avg(), owner, repoName, key.workflow, key.job)
	}
}

type snapshot struct {
	refreshedAt  time.Time
	repositories []repositorySnapshot
}

type repositorySnapshot struct {
	repo    gh.Repository
	runners []gh.Runner
	runs    []runSnapshot
}

type runSnapshot struct {
	run  gh.WorkflowRun
	jobs []gh.WorkflowJob
}

type averager struct {
	sum   float64
	count float64
}

func (a averager) with(value float64) averager {
	a.sum += value
	a.count++
	return a
}

func (a averager) avg() float64 {
	if a.count == 0 {
		return 0
	}
	return a.sum / a.count
}

func workflowName(run gh.WorkflowRun) string {
	if run.Name != "" {
		return run.Name
	}
	if run.DisplayTitle != "" {
		return run.DisplayTitle
	}
	return "unknown"
}

func workflowDuration(run gh.WorkflowRun) (time.Duration, bool) {
	if run.RunStartedAt == nil || run.UpdatedAt.IsZero() || run.Status != "completed" {
		return 0, false
	}
	duration := run.UpdatedAt.Sub(*run.RunStartedAt)
	return duration, duration >= 0
}

func workflowQueue(run gh.WorkflowRun) (time.Duration, bool) {
	if run.RunStartedAt == nil || run.CreatedAt.IsZero() {
		return 0, false
	}
	queue := run.RunStartedAt.Sub(run.CreatedAt)
	return queue, queue >= 0
}

func jobDuration(job gh.WorkflowJob) (time.Duration, bool) {
	if job.StartedAt == nil || job.CompletedAt == nil {
		return 0, false
	}
	duration := job.CompletedAt.Sub(*job.StartedAt)
	return duration, duration >= 0
}

func jobQueue(run gh.WorkflowRun, job gh.WorkflowJob) (time.Duration, bool) {
	if job.StartedAt == nil || run.CreatedAt.IsZero() {
		return 0, false
	}
	queue := job.StartedAt.Sub(run.CreatedAt)
	return queue, queue >= 0
}

func boolValue(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func valueOrNone(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

type workflowCountKey struct {
	workflow   string
	status     string
	conclusion string
}

type workflowDurationKey struct {
	workflow   string
	conclusion string
}

type workflowQueueKey struct {
	workflow string
}

type jobDurationKey struct {
	workflow   string
	job        string
	conclusion string
}

type jobQueueKey struct {
	workflow string
	job      string
}

func sortedWorkflowCountKeys(values map[workflowCountKey]float64) []workflowCountKey {
	keys := make([]workflowCountKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].workflow+keys[i].status+keys[i].conclusion < keys[j].workflow+keys[j].status+keys[j].conclusion
	})
	return keys
}

func sortedWorkflowDurationKeys(values map[workflowDurationKey]averager) []workflowDurationKey {
	keys := make([]workflowDurationKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].workflow+keys[i].conclusion < keys[j].workflow+keys[j].conclusion
	})
	return keys
}

func sortedWorkflowQueueKeys(values map[workflowQueueKey]averager) []workflowQueueKey {
	keys := make([]workflowQueueKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].workflow < keys[j].workflow
	})
	return keys
}

func sortedJobDurationKeys(values map[jobDurationKey]averager) []jobDurationKey {
	keys := make([]jobDurationKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].workflow+keys[i].job+keys[i].conclusion < keys[j].workflow+keys[j].job+keys[j].conclusion
	})
	return keys
}

func sortedJobQueueKeys(values map[jobQueueKey]averager) []jobQueueKey {
	keys := make([]jobQueueKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].workflow+keys[i].job < keys[j].workflow+keys[j].job
	})
	return keys
}
