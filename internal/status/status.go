package status

import (
	"context"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

var Version = "dev"

const (
	defaultCheckTimeout = 2 * time.Second
	defaultCacheTTL     = 2 * time.Second
)

type State string

const (
	StateOK       State = "ok"
	StateDegraded State = "degraded"
	StateDown     State = "down"
)

type Component struct {
	Name      string         `json:"name"`
	Status    State          `json:"status"`
	LatencyMs float64        `json:"latency_ms"`
	Error     string         `json:"error,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

type Checker interface {
	Name() string
	Check(ctx context.Context) Component
}

type ServiceInfo struct {
	Name          string    `json:"name"`
	APIVersion    string    `json:"api_version"`
	Version       string    `json:"version"`
	Commit        string    `json:"commit"`
	BuildTime     string    `json:"build_time"`
	GoVersion     string    `json:"go_version"`
	Env           string    `json:"env"`
	StartedAt     time.Time `json:"started_at"`
	UptimeSeconds int64     `json:"uptime_seconds"`
}

type Report struct {
	Status     State       `json:"status"`
	Service    ServiceInfo `json:"service"`
	Components []Component `json:"components"`
	CheckedAt  time.Time   `json:"checked_at"`
}

type Options struct {
	Name         string
	APIVersion   string
	Env          string
	Log          *slog.Logger
	CheckTimeout time.Duration
	CacheTTL     time.Duration
}

type Service struct {
	info         ServiceInfo
	checkers     []Checker
	log          *slog.Logger
	checkTimeout time.Duration
	cacheTTL     time.Duration
	started      time.Time
	now          func() time.Time

	mu       sync.Mutex
	cached   *Report
	cachedAt time.Time
}

func New(opts Options, checkers ...Checker) *Service {
	if opts.CheckTimeout <= 0 {
		opts.CheckTimeout = defaultCheckTimeout
	}
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = defaultCacheTTL
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}

	commit, buildTime := readBuildInfo()
	started := time.Now().UTC()

	return &Service{
		info: ServiceInfo{
			Name:       opts.Name,
			APIVersion: opts.APIVersion,
			Version:    Version,
			Commit:     commit,
			BuildTime:  buildTime,
			GoVersion:  runtime.Version(),
			Env:        opts.Env,
			StartedAt:  started,
		},
		checkers:     checkers,
		log:          opts.Log,
		checkTimeout: opts.CheckTimeout,
		cacheTTL:     opts.CacheTTL,
		started:      started,
		now:          time.Now,
	}
}

func (s *Service) Report(ctx context.Context) Report {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached == nil || s.now().Sub(s.cachedAt) >= s.cacheTTL {
		report := s.run(ctx)
		s.cached = &report
		s.cachedAt = s.now()
	}

	out := *s.cached
	out.Service.UptimeSeconds = int64(s.now().Sub(s.started).Seconds())
	return out
}

func (s *Service) run(ctx context.Context) Report {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.checkTimeout)
	defer cancel()

	components := make([]Component, len(s.checkers))

	var wg sync.WaitGroup
	for i, c := range s.checkers {
		wg.Go(func() {
			defer func() {
				if p := recover(); p != nil {
					s.log.ErrorContext(ctx, "status check panicked", slog.String("check", c.Name()), slog.Any("panic", p))
					components[i] = Component{Status: StateDown, Error: "check failed unexpectedly"}
				}
				components[i].Name = c.Name()
			}()
			components[i] = c.Check(ctx)
		})
	}
	wg.Wait()

	return Report{
		Status:     overall(components),
		Service:    s.info,
		Components: components,
		CheckedAt:  s.now().UTC(),
	}
}

func overall(components []Component) State {
	result := StateOK
	for _, c := range components {
		switch c.Status {
		case StateDown:
			return StateDown
		case StateDegraded:
			result = StateDegraded
		}
	}
	return result
}

func readBuildInfo() (commit, buildTime string) {
	commit, buildTime = "unknown", "unknown"

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return commit, buildTime
	}

	modified := false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
			if len(commit) > 12 {
				commit = commit[:12]
			}
		case "vcs.time":
			buildTime = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if modified && commit != "unknown" {
		commit += "-dirty" // built with uncommitted changes
	}
	return commit, buildTime
}
