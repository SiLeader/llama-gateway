package llamaserver

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"
)

type Manager struct {
	executable string
	modelsDir  string
	presetFile string
	args       []string
	port       string
	spawnChan  chan os.Signal
}

func NewManager(config Config, port int, modelsDir string, presetFile string) *Manager {
	return &Manager{
		executable: config.Executable,
		modelsDir:  modelsDir,
		presetFile: presetFile,
		args:       config.Args,
		port:       fmt.Sprintf("%d", port),
		spawnChan:  make(chan os.Signal, 1),
	}
}

func (m *Manager) ReloadServer() {
	m.spawnChan <- syscall.SIGHUP
}

func (m *Manager) Close() {
	close(m.spawnChan)
}

func (m *Manager) Start() {
	go m.run()
	m.spawnChan <- syscall.SIGHUP
}

func (m *Manager) run() {
	var cmd *exec.Cmd = nil
	retry := 0

	signal.Notify(m.spawnChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for sig := range m.spawnChan {
		stopProcess(cmd)

		if sig == syscall.SIGINT || sig == syscall.SIGTERM {
			slog.Info("Received signal to stop llama server", "signal", sig)
			return
		}

		cmd = exec.Command(m.executable, "--models-dir", m.modelsDir, "--models-preset", m.presetFile, "--port", m.port)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path.Dir(m.executable)
		cmd.Env = append(cmd.Env, fmt.Sprintf("LD_LIBRARY_PATH=%s", path.Dir(m.executable)))
		cmd.Args = append(cmd.Args, m.args...)

		slog.Debug("Starting llama server", "cmd", cmd)
		if err := cmd.Start(); err != nil {
			slog.Error("Failed to start llama server", "error", err)
			if retry < 5 {
				time.Sleep(2 * time.Second)
				retry++
				continue
			}
			slog.Error("Failed to start llama server after 5 retries", "error", err)
			return
		}
		slog.Info("Started llama server", "port", m.port)
		go waitProcess(cmd, m.spawnChan)
	}
	stopProcess(cmd)
}

func waitProcess(cmd *exec.Cmd, stopChan chan os.Signal) {
	if cmd != nil {
		if err := cmd.Wait(); err != nil {
			slog.Error("Failed to wait llama server", "error", err)
		}
		if cmd.ProcessState.ExitCode() != 0 {
			slog.Error("llama server exited with non-zero exit code", "exit_code", cmd.ProcessState.ExitCode())
		}
	}
	stopChan <- syscall.SIGHUP
}

func stopProcess(cmd *exec.Cmd) {
	if cmd != nil {
		if err := cmd.Process.Signal(os.Signal(syscall.SIGTERM)); err != nil {
			slog.Error("Send signal failed", "err", err)
		}
		if err := cmd.Wait(); err != nil {
			slog.Error("Failed to wait llama server", "error", err)
		}
	}
}
