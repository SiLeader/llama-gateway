package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/sileader/llama-gateway/huggingface"
	"github.com/sileader/llama-gateway/llamaserver"
	"github.com/sileader/llama-gateway/model"
	"github.com/sileader/llama-gateway/orchestrator"
	"github.com/sileader/llama-gateway/revproxy"
	"gopkg.in/yaml.v3"
)

type config struct {
	Server      revproxy.ServerConfig `yaml:"server"`
	Models      []model.Info          `yaml:"models"`
	Backend     backend               `yaml:"backend"`
	Directories directories           `yaml:"directories"`
}

type backend struct {
	Port        int                `yaml:"port,omitempty"`
	Ports       []int              `yaml:"ports,omitempty"`
	LlamaServer llamaserver.Config `yaml:"llamaServer"`
}

type directories struct {
	Models string `yaml:"models"`
	Config string `yaml:"config"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalln("Failed to load .env file", err)
		}
	}

	configFile := flag.String("config", "/etc/llama-gateway/config.yaml", "config file path")
	flag.Parse()

	{
		lls := os.Getenv("LOG_LEVEL")
		logLevel := slog.LevelInfo
		if lls != "" {
			if err := logLevel.UnmarshalText([]byte(lls)); err != nil {
				log.Println("warn: Failed to parse LOG_LEVEL", err)
			}
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	}

	shutdownCtx, shutdownCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer shutdownCancel()

	cfg := loadGlobalConfig(*configFile)
	presetFile := fmt.Sprintf("%s/presets.ini", cfg.Directories.Config)

	ports := resolveBackendPorts(cfg.Backend, cfg.Server.ListenPort())
	validatePorts(ports, cfg.Server.ListenPort())

	reloadConfigFn := func() (llamaserver.Config, error) {
		return loadGlobalConfig(*configFile).Backend.LlamaServer, nil
	}

	// Proxy is created before the orchestrator so we can pass it as UpstreamSwapper.
	// The first SetUpstream call happens inside orchestrator.Start.
	url := fmt.Sprintf("http://localhost:%d", ports[0])
	proxy, err := revproxy.NewProxy(cfg.Server, url, nil, nil)
	if err != nil {
		log.Fatalln("Failed to create proxy instance", err)
	}

	orch := orchestrator.New(ports, cfg.Backend.LlamaServer, cfg.Directories.Models, presetFile, proxy, reloadConfigFn)

	hfToken := loadHfConfig()
	hfClient := huggingface.NewClient(hfToken)
	downloader, err := model.NewDownloader(cfg.Models, cfg.Directories.Models, presetFile, hfClient, orch)
	if err != nil {
		log.Fatalln("Failed to create downloader", err)
	}
	if err := downloader.DownloadAll(shutdownCtx); err != nil {
		log.Fatalln("Failed to download all models", "error", err)
	}
	slog.Info("Downloaded all models")

	// Wire the reloader and downloader into the proxy now that orchestrator exists.
	proxy.SetReloader(orch)
	proxy.SetDownloader(downloader)

	if err := orch.Start(shutdownCtx); err != nil {
		log.Fatalln("Failed to start llama server", "error", err)
	}
	slog.Info("llama-server ready", "ports", ports)

	// SIGHUP: reload config and trigger rollover.
	sighupCh := make(chan os.Signal, 1)
	signal.Notify(sighupCh, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-sighupCh:
				slog.Info("SIGHUP received — reloading")
				if err := orch.Reload(context.Background()); err != nil {
					slog.Error("Reload on SIGHUP failed", "error", err)
				}
			case <-shutdownCtx.Done():
				return
			}
		}
	}()

	// fsnotify: watch config.yaml for changes.
	go watchConfig(shutdownCtx, *configFile, orch)

	if err := proxy.ListenAndServe(shutdownCtx); err != nil {
		slog.Error("Reverse proxy error", "error", err)
	}
	orch.Close()
	slog.Info("Bye!")
}

// watchConfig watches the config file and triggers a reload on changes.
func watchConfig(ctx context.Context, configFile string, orch *orchestrator.Orchestrator) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("Failed to create fsnotify watcher", "error", err)
		return
	}
	defer watcher.Close()

	// Watch the directory to catch atomic-save patterns (rename/create).
	dir := filepath.Dir(configFile)
	if err := watcher.Add(dir); err != nil {
		slog.Warn("Failed to watch config directory", "dir", dir, "error", err)
		return
	}
	slog.Info("Watching config file for changes", "path", configFile)

	var debounce <-chan time.Time
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Clean(event.Name) != filepath.Clean(configFile) {
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				debounce = time.After(500 * time.Millisecond)
			}
		case <-debounce:
			debounce = nil
			slog.Info("Config file changed — reloading")
			if err := orch.Reload(context.Background()); err != nil {
				slog.Error("Reload on config change failed", "error", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("fsnotify error", "error", err)
		case <-ctx.Done():
			return
		}
	}
}

// resolveBackendPorts returns the list of llama-server ports from config.
// Priority: backend.ports > backend.port > listen.port+1.
func resolveBackendPorts(b backend, listenPort int) []int {
	if len(b.Ports) > 0 {
		return b.Ports
	}
	if b.Port != 0 {
		return []int{b.Port}
	}
	return []int{listenPort + 1}
}

func validatePorts(ports []int, listenPort int) {
	seen := map[int]bool{}
	for _, p := range ports {
		if p == listenPort {
			log.Fatalf("backend port %d conflicts with gateway listen port", p)
		}
		if seen[p] {
			log.Fatalf("duplicate backend port %d", p)
		}
		seen[p] = true
	}
}

func loadGlobalConfig(path string) (c config) {
	slog.Info("Loading global config", "path", path)
	configContent, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln("Failed to read config file", err)
	}
	if err := yaml.Unmarshal(configContent, &c); err != nil {
		log.Fatalln("Failed to parse config file", err)
	}
	return
}

func loadHfConfig() (hfToken string) {
	slog.Info("Loading Hugging Face config")
	hfToken = os.Getenv("HF_TOKEN")
	return
}
