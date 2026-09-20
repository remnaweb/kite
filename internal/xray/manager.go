package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	Bin        string
	ConfigPath string
	LogPath    string
	APIPort    int

	mu      sync.Mutex
	cmd     *exec.Cmd
	running bool
	lastErr string
}

func NewManager(bin, configPath, logPath string, apiPort int) *Manager {
	return &Manager{
		Bin:        bin,
		ConfigPath: configPath,
		LogPath:    logPath,
		APIPort:    apiPort,
	}
}

func (m *Manager) BinaryExists() bool {
	st, err := os.Stat(m.Bin)
	return err == nil && !st.IsDir()
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.aliveLocked()
}

func (m *Manager) LastError() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastErr
}

func (m *Manager) Status() map[string]any {
	return map[string]any{
		"running":      m.Running(),
		"binary":       m.Bin,
		"binaryExists": m.BinaryExists(),
		"apiPort":      m.APIPort,
		"lastError":    m.LastError(),
	}
}

func (m *Manager) WriteConfig(cfg map[string]any) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.ConfigPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.ConfigPath, raw, 0o600)
}

func (m *Manager) Restart() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.stopLocked(); err != nil {
		m.lastErr = err.Error()
		return err
	}
	if err := m.startLocked(); err != nil {
		m.lastErr = err.Error()
		return err
	}
	m.lastErr = ""
	return nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startLocked()
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

func (m *Manager) startLocked() error {
	if m.aliveLocked() {
		return nil
	}
	if !m.BinaryExists() {
		return fmt.Errorf("xray binary not found at %s", m.Bin)
	}
	if _, err := os.Stat(m.ConfigPath); err != nil {
		return fmt.Errorf("xray config missing: %w", err)
	}
	logFile, err := os.OpenFile(m.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.Bin, "run", "-c", m.ConfigPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	m.cmd = cmd
	m.running = true
	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.cmd == cmd {
			m.running = false
			m.cmd = nil
			if err != nil && m.lastErr == "" {
				m.lastErr = err.Error()
			}
		}
	}()
	time.Sleep(250 * time.Millisecond)
	if !m.aliveLocked() {
		return fmt.Errorf("xray exited immediately, see %s", m.LogPath)
	}
	return nil
}

func (m *Manager) stopLocked() error {
	if m.cmd == nil || m.cmd.Process == nil {
		m.running = false
		return nil
	}
	_ = m.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_, _ = m.cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = m.cmd.Process.Kill()
	}
	m.cmd = nil
	m.running = false
	return nil
}

func (m *Manager) aliveLocked() bool {
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	if err := m.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		m.running = false
		return false
	}
	return true
}

