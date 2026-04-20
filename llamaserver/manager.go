package llamaserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sync/atomic"
	"syscall"
	"time"
)

type serverEvent int

const (
	serverEventRestart serverEvent = iota
	serverEventStopped
)

type managedCmd struct {
	cmd      *exec.Cmd
	done     chan struct{}
	stopping atomic.Bool
}

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
		spawnChan:  make(chan serverEvent, 64),
	}
}

func (m *Manager) RestartServer() {
	m.spawnChan <- serverEventRestart
}

func (m *Manager) Close() {
	m.spawnChan <- serverEventStopped
}

func (m *Manager) Run(ctx context.Context) {
	var mc *managedCmd
	retry := 0

	m.spawnChan <- serverEventRestart
	for sig := range m.spawnChan {
		stopCmd(mc)

		if sig == serverEventStopped {
			slog.Info("Received signal to stop llama server")
			return
		}

		cmd := exec.CommandContext(
			ctx,
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
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return nil
			}
			return cmd.Process.Signal(syscall.SIGTERM)
		}
		cmd.WaitDelay = 5 * time.Second

		slog.Debug("Starting llama server", "cmd", cmd)
		if err := cmd.Start(); err != nil {
			slog.Error("Failed to start llama server", "error", err)
			if retry < 5 {
				time.Sleep(2 * time.Second)
				retry++
				m.spawnChan <- serverEventRestart
				continue
			}
			slog.Error("Failed to start llama server after 5 retries")
			return
		}
		retry = 0
		slog.Info("Started llama server", "port", m.port)

		mc = &managedCmd{cmd: cmd, done: make(chan struct{})}
		go func(mc *managedCmd) {
			defer close(mc.done)
			err := mc.cmd.Wait()
			if err != nil {
				slog.Error("llama-server exited with error", "error", err)
			} else if ps := mc.cmd.ProcessState; ps != nil && ps.ExitCode() != 0 {
				slog.Error("llama-server non-zero exit", "exit_code", ps.ExitCode())
			} else {
				slog.Info("llama-server exited cleanly")
			}
			if !mc.stopping.Load() {
				m.spawnChan <- serverEventRestart
			}
		}(mc)
	}
	stopCmd(mc)
	slog.Info("Stopped llama server manager")
}

func stopCmd(mc *managedCmd) {
	if mc == nil {
		return
	}
	select {
	case <-mc.done:
		return
	default:
	}
	mc.stopping.Store(true)
	if err := mc.cmd.Process.Signal(syscall.SIGTERM); err != nil &&
		!errors.Is(err, os.ErrProcessDone) {
		slog.Warn("Failed to send SIGTERM to llama-server", "error", err)
	}
	select {
	case <-mc.done:
	case <-time.After(5 * time.Second):
		_ = mc.cmd.Process.Kill()
		select {
		case <-mc.done:
		case <-time.After(5 * time.Second):
			slog.Error("llama-server failed to exit after SIGKILL")
		}
	}
}
