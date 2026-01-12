package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/mjshashank/dabbi/internal/multipass"
)

// Manager manages TCP tunnels to VMs
type Manager struct {
	mu      sync.RWMutex
	tunnels map[int]*Tunnel
	mp      multipass.Client
}

// Tunnel represents an active TCP tunnel
type Tunnel struct {
	HostPort int
	VMName   string
	VMPort   int
	vmIP     string
	listener net.Listener
	done     chan struct{}
}

// NewManager creates a new tunnel manager
func NewManager(mp multipass.Client) *Manager {
	return &Manager{
		tunnels: make(map[int]*Tunnel),
		mp:      mp,
	}
}

// Create creates a new tunnel to a VM port
func (m *Manager) Create(vmName string, vmPort int) (*Tunnel, error) {
	// Ensure VM is running
	info, err := m.mp.Info(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	if info.State != multipass.StateRunning {
		return nil, fmt.Errorf("VM %q is not running (state: %s)", vmName, info.State)
	}

	if len(info.IPv4) == 0 {
		return nil, fmt.Errorf("VM has no IP address")
	}
	vmIP := info.IPv4[0]

	// Find free port on host
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	hostPort := listener.Addr().(*net.TCPAddr).Port

	tunnel := &Tunnel{
		HostPort: hostPort,
		VMName:   vmName,
		VMPort:   vmPort,
		vmIP:     vmIP,
		listener: listener,
		done:     make(chan struct{}),
	}

	go tunnel.serve()

	m.mu.Lock()
	m.tunnels[hostPort] = tunnel
	m.mu.Unlock()

	return tunnel, nil
}

// Delete closes a tunnel
func (m *Manager) Delete(hostPort int) error {
	m.mu.Lock()
	tunnel, ok := m.tunnels[hostPort]
	if ok {
		delete(m.tunnels, hostPort)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("tunnel not found: %d", hostPort)
	}

	close(tunnel.done)
	tunnel.listener.Close()
	return nil
}

// List returns all active tunnels
func (m *Manager) List() []*Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]*Tunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		tunnels = append(tunnels, t)
	}
	return tunnels
}

// serve accepts connections and proxies them to the VM
func (t *Tunnel) serve() {
	for {
		select {
		case <-t.done:
			return
		default:
			// Set a deadline so we can check done periodically
			if tcpListener, ok := t.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			}

			conn, err := t.listener.Accept()
			if err != nil {
				// Check if we're done
				select {
				case <-t.done:
					return
				default:
					continue
				}
			}
			go t.handleConnection(conn)
		}
	}
}

// handleConnection proxies a single connection to the VM
func (t *Tunnel) handleConnection(client net.Conn) {
	defer client.Close()

	// Connect to VM
	target, err := net.Dial("tcp", fmt.Sprintf("%s:%d", t.vmIP, t.VMPort))
	if err != nil {
		return
	}
	defer target.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(target, client)
		target.(*net.TCPConn).CloseWrite()
	}()

	go func() {
		defer wg.Done()
		io.Copy(client, target)
		client.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
}
