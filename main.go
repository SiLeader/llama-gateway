package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/sileader/llama-gateway/huggingface"
	"github.com/sileader/llama-gateway/llamaserver"
	"github.com/sileader/llama-gateway/model"
	"github.com/sileader/llama-gateway/revproxy"
	"sigs.k8s.io/yaml"
)

type config struct {
	Listen      listen       `yaml:"listen"`
	Models      []model.Info `yaml:"mapping"`
	Backend     backend      `yaml:"backend"`
	Directories directories  `yaml:"directories"`
}

type listen struct {
	Host string `yaml:"host" default:"0.0.0.0"`
	Port int    `yaml:"port" default:"8080"`
}

type backend struct {
	LlamaServer llamaserver.Config `yaml:"llamaServer"`
}

type directories struct {
	Models string `yaml:"models" default:"/models"`
	Config string `yaml:"config" default:"/config"`
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

	{
		hfToken := loadHfConfig()
		hfClient := huggingface.NewClient(hfToken)
		downloader := NewModelDownloader(config.Models, config.Directories.Models, hfClient)
		downloader.DownloadAll(presetFile)
		slog.Info("Downloaded all models")
	}

	mapper := model.NewModelMapper(config.Models, config.Directories.Models)

	llamaServerPort := config.Listen.Port + 1
	spawner := llamaserver.NewManager(config.Backend.LlamaServer, llamaServerPort, config.Directories.Models, presetFile)
	spawner.Start()
	slog.Info("Started llama server", "port", llamaServerPort)

	url := fmt.Sprintf("http://localhost:%d", llamaServerPort)
	proxy := revproxy.NewProxy(url, mapper)

	slog.Info("Starting reverse proxy", "url", url, "port", config.Listen.Port, "host", config.Listen.Host)
	http.ListenAndServe(fmt.Sprintf("%s:%d", config.Listen.Host, config.Listen.Port), accessLog(proxy))
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
