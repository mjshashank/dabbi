package agent

import (
	"context"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/mjshashank/dabbi/internal/multipass"
)

const (
	agentPort     = 1234  // opencode port inside VM
	basePort      = 11000 // start of agent port range
	portRange     = 1000  // number of ports in range
	startupTimeout = 30 * time.Second
)

// Manager manages HTTP reverse proxy listeners for VM agents
type Manager struct {
	mp        multipass.Client
	listeners sync.Map // vmName -> *listener
}

type listener struct {
	server   *http.Server
	listener net.Listener
	vmName   string
	port     int
}

// NewManager creates a new agent manager
func NewManager(mp multipass.Client) *Manager {
	return &Manager{mp: mp}
}

// PortForVM returns the deterministic port for a VM based on its name
func PortForVM(vmName string) int {
	h := fnv.New32a()
	h.Write([]byte(vmName))
	return basePort + int(h.Sum32()%portRange)
}

// Start starts the agent proxy listener for a VM
func (m *Manager) Start(vmName string) error {
	// Check if already running
	if _, exists := m.listeners.Load(vmName); exists {
		return nil
	}

	// Get VM info to verify it exists and get IP
	info, err := m.mp.Info(vmName)
	if err != nil {
		return fmt.Errorf("VM '%s' not found: %w", vmName, err)
	}

	if info.State != multipass.StateRunning {
		return fmt.Errorf("VM '%s' is not running (state: %s)", vmName, info.State)
	}

	if len(info.IPv4) == 0 {
		return fmt.Errorf("VM '%s' has no IP address", vmName)
	}

	vmIP := info.IPv4[0]
	port := PortForVM(vmName)

	// Create listener on the determined port
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	// Create reverse proxy to VM
	target, _ := url.Parse(fmt.Sprintf("http://%s:%d", vmIP, agentPort))
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom director to set headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("Agent proxy error: %v", err), http.StatusBadGateway)
	}

	// Create HTTP server
	server := &http.Server{
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	l := &listener{
		server:   server,
		listener: ln,
		vmName:   vmName,
		port:     port,
	}

	m.listeners.Store(vmName, l)

	// Start serving in background
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash - listener might have been stopped
		}
		m.listeners.Delete(vmName)
	}()

	return nil
}

// Stop stops the agent proxy listener for a VM
func (m *Manager) Stop(vmName string) {
	if val, exists := m.listeners.LoadAndDelete(vmName); exists {
		l := val.(*listener)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		l.server.Shutdown(ctx)
	}
}

// GetURL returns the agent URL for a VM, starting the listener if needed
func (m *Manager) GetURL(vmName, host string) (string, error) {
	// Start listener if not running
	if err := m.Start(vmName); err != nil {
		return "", err
	}

	port := PortForVM(vmName)

	// Extract hostname without port from the host header
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	return fmt.Sprintf("http://%s:%d/", hostname, port), nil
}

// StopAll stops all agent listeners
func (m *Manager) StopAll() {
	m.listeners.Range(func(key, value any) bool {
		vmName := key.(string)
		m.Stop(vmName)
		return true
	})
}

// IsRunning checks if a listener is running for a VM
func (m *Manager) IsRunning(vmName string) bool {
	_, exists := m.listeners.Load(vmName)
	return exists
}

// VerifyVM checks if a VM exists and is running (without starting a listener)
func (m *Manager) VerifyVM(vmName string) error {
	info, err := m.mp.Info(vmName)
	if err != nil {
		return fmt.Errorf("VM '%s' not found: %w", vmName, err)
	}

	if info.State != multipass.StateRunning {
		return fmt.Errorf("VM '%s' is not running (state: %s)", vmName, info.State)
	}

	if len(info.IPv4) == 0 {
		return fmt.Errorf("VM '%s' has no IP address", vmName)
	}

	return nil
}
