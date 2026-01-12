package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/mjshashank/dabbi/internal/multipass"
)

const (
	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer at this interval (must be less than pongWait)
	pingPeriod = 30 * time.Second

	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     checkOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// checkOrigin validates WebSocket connection origins to prevent CSRF attacks.
// Allows: no origin (non-browser clients), localhost, and same-origin requests.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// No origin header = non-browser client (curl, CLI tools) - allow
	if origin == "" {
		return true
	}

	// Parse origin
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := u.Hostname()

	// Allow localhost variants
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Allow requests where origin matches the Host header (same-origin)
	reqHost := r.Host
	if h, _, err := net.SplitHostPort(reqHost); err == nil {
		reqHost = h
	}
	if host == reqHost {
		return true
	}

	return false
}

// ShellHandler handles WebSocket shell sessions
type ShellHandler struct {
	mp multipass.Client
}

// NewShellHandler creates a new shell handler
func NewShellHandler(mp multipass.Client) *ShellHandler {
	return &ShellHandler{mp: mp}
}

// ResizeMessage represents a terminal resize message
type ResizeMessage struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// Handle upgrades to WebSocket and provides shell access
func (h *ShellHandler) Handle(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")

	// Ensure VM exists and is running
	info, err := h.mp.Info(vmName)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if info.State != multipass.StateRunning {
		http.Error(w, "VM is not running", http.StatusBadRequest)
		return
	}

	// Get initial terminal size from query params
	// This ensures the PTY starts with correct dimensions from the very beginning
	initialCols := 80
	initialRows := 24
	if cols := r.URL.Query().Get("cols"); cols != "" {
		if c, err := strconv.Atoi(cols); err == nil && c > 0 {
			initialCols = c
		}
	}
	if rows := r.URL.Query().Get("rows"); rows != "" {
		if r, err := strconv.Atoi(rows); err == nil && r > 0 {
			initialRows = r
		}
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Start multipass shell with PTY at the correct initial size
	// CRITICAL: Using StartWithSize ensures the shell starts with correct dimensions
	// This fixes TUI applications like Claude Code that read terminal size at startup
	cmd := exec.Command("multipass", "shell", vmName)

	// Set environment variables for proper terminal behavior
	cmd.Env = append(cmd.Environ(),
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(initialRows),
		Cols: uint16(initialCols),
	})
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to start shell: "+err.Error()))
		return
	}

	// Ensure PTY and process are cleaned up on any exit
	defer func() {
		ptmx.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait() // Reap the zombie process
		}
	}()

	// Channel to signal all goroutines to stop
	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() {
		closeOnce.Do(func() {
			close(done)
		})
	}

	// Mutex to synchronize WebSocket writes (ping + PTY output)
	var writeMu sync.Mutex

	// Set up WebSocket ping/pong for dead connection detection
	// This is critical for detecting when browser tabs are closed abruptly
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Goroutine 1: Send pings periodically to detect dead connections
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				writeMu.Lock()
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				writeMu.Unlock()
				if err != nil {
					closeDone()
					return
				}
			}
		}
	}()

	// Goroutine 2: Read from PTY and send to WebSocket
	go func() {
		defer closeDone()
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
			}

			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}

			writeMu.Lock()
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			err = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	// Main goroutine: Read from WebSocket and write to PTY
	for {
		select {
		case <-done:
			return
		default:
		}

		// ReadMessage will return error when read deadline expires (no pong received)
		// or when the connection is closed
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			closeDone()
			return
		}

		// Check for resize message (JSON with type: "resize")
		if msgType == websocket.TextMessage && len(data) > 0 && data[0] == '{' {
			var resize ResizeMessage
			if err := json.Unmarshal(data, &resize); err == nil && resize.Type == "resize" {
				pty.Setsize(ptmx, &pty.Winsize{
					Rows: resize.Rows,
					Cols: resize.Cols,
				})
				continue
			}
		}

		// Write to PTY
		if _, err := ptmx.Write(data); err != nil {
			closeDone()
			return
		}
	}
}
