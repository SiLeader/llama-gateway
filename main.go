package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/sileader/llama-gateway/huggingface"
	"github.com/sileader/llama-gateway/llamaserver"
	"github.com/sileader/llama-gateway/model"
	"github.com/sileader/llama-gateway/revproxy"
	"sigs.k8s.io/yaml"
)

type config struct {
	Server      revproxy.ServerConfig `yaml:"server"`
	Models      []model.Info          `yaml:"models"`
	Backend     backend               `yaml:"backend"`
	Directories directories           `yaml:"directories"`
}

type backend struct {
	Port        int                `yaml:"port"`
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
			panic(err)
		}
	}

	configFile := flag.String("config", "/etc/llama-gateway/config.yaml", "config file path")

	flag.Parse()

	{
		lls := os.Getenv("LOG_LEVEL")
		logLevel := slog.LevelInfo
		if lls == "debug" {
			logLevel = slog.LevelDebug
		} else if lls == "warn" {
			logLevel = slog.LevelWarn
		} else if lls == "error" {
			logLevel = slog.LevelError
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	}

	config := loadGlobalConfig(*configFile)
	presetFile := fmt.Sprintf("%s/presets.ini", config.Directories.Config)

	llamaServerPort := config.Server.ListenPort() + 1
	if config.Backend.Port != 0 {
		llamaServerPort = config.Backend.Port
	}
	spawner := llamaserver.NewManager(config.Backend.LlamaServer, llamaServerPort, config.Directories.Models, presetFile)

	hfToken := loadHfConfig()
	hfClient := huggingface.NewClient(hfToken)
	downloader := model.NewDownloader(config.Models, config.Directories.Models, presetFile, hfClient, spawner)
	if err := downloader.DownloadAll(); err != nil {
		panic(err)
	}
	slog.Info("Downloaded all models")

	spawner.Start()
	slog.Info("Started llama server", "port", llamaServerPort)

	url := fmt.Sprintf("http://localhost:%d", llamaServerPort)
	proxy := revproxy.NewProxy(config.Server, url, downloader)

	if err := proxy.ListenAndServe(); err != nil {
		panic(err)
	}
	slog.Info("Bye!")
}

func loadGlobalConfig(path string) (config config) {
	slog.Info("Loading global config", "path", path)
	configContent, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(configContent, &config); err != nil {
		panic(err)
	}
	return
}

func loadHfConfig() (hfToken string) {
	slog.Info("Loading Hugging Face config")
	hfToken = os.Getenv("HF_TOKEN")

	return
}
