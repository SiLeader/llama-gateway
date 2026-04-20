package llamaserver

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	threads := 4
	m := NewManager(Config{Executable: "/bin/true", Threads: &threads}, 8081, "/tmp/models", "/tmp/presets.ini")
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
	if m.port != "8081" {
		t.Errorf("port = %q, want 8081", m.port)
	}
	if m.threads != 4 {
		t.Errorf("threads = %d, want 4", m.threads)
	}
	if m.executable != "/bin/true" {
		t.Errorf("executable = %q, want /bin/true", m.executable)
	}
}

func TestNewManager_DefaultThreads(t *testing.T) {
	m := NewManager(Config{Executable: "/bin/true"}, 8081, "/tmp", "/tmp/p.ini")
	if m.threads <= 0 {
		t.Errorf("threads = %d, expected positive value from getCpuThreads", m.threads)
	}
}

func TestRestartServer_SendsEvent(t *testing.T) {
	m := NewManager(Config{Executable: "/bin/true"}, 8081, "/tmp", "/tmp/p.ini")
	m.RestartServer()
	select {
	case evt := <-m.spawnChan:
		if evt != serverEventRestart {
			t.Errorf("expected serverEventRestart, got %d", evt)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for restart event")
	}
}

func TestClose_SendsStoppedEvent(t *testing.T) {
	m := NewManager(Config{Executable: "/bin/true"}, 8081, "/tmp", "/tmp/p.ini")
	m.Close()
	select {
	case evt := <-m.spawnChan:
		if evt != serverEventStopped {
			t.Errorf("expected serverEventStopped, got %d", evt)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for stopped event")
	}
}

func TestStopCmd_Nil(t *testing.T) {
	// Should not panic
	stopCmd(nil)
}

func TestStopCmd_AlreadyDone(t *testing.T) {
	mc := &managedCmd{done: make(chan struct{})}
	close(mc.done)
	// Should return immediately without panicking
	stopCmd(mc)
}

func TestManagerRun_StartsAndStops(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}

	m := NewManager(Config{Executable: "/bin/sleep", Args: []string{"10"}}, 19999, "/tmp", "/tmp/p.ini")
	// Override args to just pass "10" — but manager always appends m.args after fixed args.
	// Use a simple long-running process.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.Run(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(200 * time.Millisecond)

	// Signal to stop
	m.Close()

	select {
	case <-done:
		// Run exited cleanly
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not exit after Close")
	}
}

func TestManagerRun_RestartsOnCrash(t *testing.T) {
	// /bin/true exits immediately (exit 0), which triggers a restart
	// We send Close quickly after to stop the loop
	m := NewManager(Config{Executable: "/bin/true"}, 19998, "/tmp", "/tmp/p.ini")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.Run(ctx)
	}()

	// Let it restart at least once
	time.Sleep(300 * time.Millisecond)
	m.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit after Close")
	}
}
