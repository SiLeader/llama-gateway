package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/sileader/llama-gateway/llamaserver"
	"github.com/sileader/llama-gateway/revproxy"
)

// UpstreamSwapper is implemented by revproxy.Proxy.
type UpstreamSwapper interface {
	SetUpstream(target *url.URL) revproxy.UpstreamHandle
}

// ServerController is the interface model.Downloader uses to trigger rollovers.
type ServerController interface {
	RestartServer(ctx context.Context) error
}

type instance struct {
	mgr     *llamaserver.Manager
	port    int
	runDone chan struct{}
}

// Orchestrator manages zero-downtime blue/green rollovers of llamaserver.
type Orchestrator struct {
	ports         []int
	llamaCfg      llamaserver.Config
	modelsDir     string
	presetFile    string
	proxy         UpstreamSwapper
	reloadConfig  func() (llamaserver.Config, error)
	healthTimeout time.Duration
	drainTimeout  time.Duration

	mu     sync.Mutex
	active *instance
}

// New creates an Orchestrator. reloadConfig may be nil when config reloading
// is not needed. ports must contain at least one element.
func New(
	ports []int,
	cfg llamaserver.Config,
	modelsDir string,
	presetFile string,
	proxy UpstreamSwapper,
	reloadConfig func() (llamaserver.Config, error),
) *Orchestrator {
	return &Orchestrator{
		ports:         ports,
		llamaCfg:      cfg,
		modelsDir:     modelsDir,
		presetFile:    presetFile,
		proxy:         proxy,
		reloadConfig:  reloadConfig,
		healthTimeout: 5 * time.Minute,
		drainTimeout:  30 * time.Second,
	}
}

// Start launches the first llama-server instance and waits for it to be ready.
func (o *Orchestrator) Start(ctx context.Context) error {
	inst, err := o.startInstance(ctx, o.ports[0], o.llamaCfg)
	if err != nil {
		return err
	}
	o.mu.Lock()
	o.active = inst
	o.mu.Unlock()

	newURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", inst.port))
	o.proxy.SetUpstream(newURL)
	return nil
}

// Reload reloads the llamaserver config and performs a rollover.
// For the single-port case it falls back to an in-place restart.
func (o *Orchestrator) Reload(ctx context.Context) error {
	var newCfg llamaserver.Config
	if o.reloadConfig != nil {
		cfg, err := o.reloadConfig()
		if err != nil {
			return fmt.Errorf("reload config: %w", err)
		}
		newCfg = cfg
	} else {
		newCfg = o.llamaCfg
	}
	return o.rollover(ctx, newCfg)
}

// RestartServer triggers a rollover without reloading config.
// Implements ServerController so model.Downloader can trigger rollovers.
func (o *Orchestrator) RestartServer(ctx context.Context) error {
	return o.rollover(ctx, o.llamaCfg)
}

// Close shuts down the active instance.
func (o *Orchestrator) Close() {
	o.mu.Lock()
	inst := o.active
	o.mu.Unlock()
	if inst != nil {
		inst.mgr.Close()
		<-inst.runDone
	}
}

func (o *Orchestrator) rollover(ctx context.Context, newCfg llamaserver.Config) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.ports) < 2 {
		slog.Warn("Zero-downtime rollover requires at least 2 ports; falling back to in-place restart")
		o.active.mgr.RestartServer()
		return nil
	}

	nextPort := o.nextPort()
	slog.Info("Starting new llama-server for rollover", "port", nextPort)

	newInst, err := o.startInstance(ctx, nextPort, newCfg)
	if err != nil {
		return fmt.Errorf("failed to start new instance on port %d: %w", nextPort, err)
	}

	// Atomically swap the proxy upstream.
	newURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", nextPort))
	oldUpstream := o.proxy.SetUpstream(newURL)
	slog.Info("Upstream swapped", "port", nextPort)

	// Drain in-flight requests on the old upstream.
	if oldUpstream != nil {
		drainCtx, cancel := context.WithTimeout(ctx, o.drainTimeout)
		defer cancel()
		if err := oldUpstream.Drain(drainCtx); err != nil {
			slog.Warn("Drain did not complete cleanly", "error", err)
		}
		oldUpstream.CloseIdleConnections()
	}

	// Retire the old instance.
	oldInst := o.active
	o.active = newInst
	o.llamaCfg = newCfg

	oldInst.mgr.Close()
	<-oldInst.runDone
	slog.Info("Old llama-server stopped", "port", oldInst.port)

	return nil
}

func (o *Orchestrator) nextPort() int {
	active := o.active.port
	for _, p := range o.ports {
		if p != active {
			return p
		}
	}
	return o.ports[0]
}

func (o *Orchestrator) startInstance(ctx context.Context, port int, cfg llamaserver.Config) (*instance, error) {
	mgr := llamaserver.NewManager(cfg, port, o.modelsDir, o.presetFile)
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		mgr.Run(ctx)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := llamaserver.WaitReady(ctx, baseURL, o.healthTimeout); err != nil {
		mgr.Close()
		<-runDone
		return nil, fmt.Errorf("health check failed on port %d: %w", port, err)
	}
	slog.Info("New llama-server ready", "port", port)
	return &instance{mgr: mgr, port: port, runDone: runDone}, nil
}
