package llamaserver

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"
)

type serverEvent int

const (
	serverEventRestart serverEvent = iota
	serverEventStopped
)

type Manager struct {
	executable string
	modelsDir  string
	presetFile string
	args       []string
	port       string
	threads    int
	spawnChan  chan serverEvent
}

func NewManager(config Config, port int, modelsDir string, presetFile string) *Manager {
	var threads int
	if config.Threads != nil {
		threads = *config.Threads
	} else {
		threads = getCpuThreads()
	}
	slog.Info("Detected CPU threads", "threads", threads)

	return &Manager{
		executable: config.Executable,
		modelsDir:  modelsDir,
		presetFile: presetFile,
		args:       config.Args,
		port:       fmt.Sprintf("%d", port),
		threads:    threads,
		spawnChan:  make(chan serverEvent, 1),
	}
}

func (m *Manager) RestartServer() {
	m.spawnChan <- serverEventRestart
}

func (m *Manager) Close() {
	close(m.spawnChan)
}

func (m *Manager) Start() {
	go m.run()
	m.spawnChan <- serverEventRestart
}

func (m *Manager) run() {
	var cmd *exec.Cmd = nil
	retry := 0

	for sig := range m.spawnChan {
		stopProcess(cmd)

		if sig == serverEventStopped {
			slog.Info("Received signal to stop llama server", "signal", sig)
			return
		}

		cmd = exec.Command(
			m.executable,
			"--models-dir", m.modelsDir,
			"--models-preset", m.presetFile,
			"--port", m.port,
			"--threads", fmt.Sprintf("%d", m.threads),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path.Dir(m.executable)
		cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", path.Dir(m.executable)))
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
		retry = 0
		slog.Info("Started llama server", "port", m.port)
		go waitProcess(cmd, m.spawnChan)
	}
	stopProcess(cmd)
	slog.Info("Stopped llama server manager")
}

func waitProcess(cmd *exec.Cmd, stopChan chan serverEvent) {
	if cmd != nil && cmd.ProcessState == nil {
		if err := cmd.Wait(); err != nil {
			slog.Error("Failed to wait llama server", "error", err)
		}
		if cmd.ProcessState == nil {
			slog.Error("llama server exited with nil ProcessState")
		} else if cmd.ProcessState.ExitCode() != 0 {
			slog.Error("llama server exited with non-zero exit code", "exit_code", cmd.ProcessState.ExitCode())
		} else {
			slog.Info("llama server exited")
		}
	}
	stopChan <- serverEventRestart
}

func stopProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.ProcessState == nil {
		slog.Info("Exiting llama server with SIGTERM")
		if err := cmd.Process.Signal(os.Signal(syscall.SIGTERM)); err != nil {
			slog.Error("Send signal failed", "err", err)
		}
		if err := cmd.Wait(); err != nil {
			slog.Error("Failed to wait llama server", "error", err)
		}
		slog.Info("llama server exited")
	}
}
