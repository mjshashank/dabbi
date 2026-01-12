package proxy

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"time"
)

const loadingHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Starting {{.VMName}}...</title>
    <meta http-equiv="refresh" content="2">
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #eee;
        }
        .container {
            text-align: center;
            padding: 40px;
        }
        .spinner {
            width: 60px;
            height: 60px;
            border: 4px solid rgba(255,255,255,0.1);
            border-top-color: #00d4ff;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin: 0 auto 30px;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        h1 {
            font-size: 28px;
            margin-bottom: 10px;
            font-weight: 500;
        }
        p {
            color: #888;
            margin: 5px 0;
        }
        .vm-name {
            color: #00d4ff;
            font-family: monospace;
            font-size: 20px;
        }
        .info {
            margin-top: 30px;
            padding: 20px;
            background: rgba(255,255,255,0.05);
            border-radius: 8px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="spinner"></div>
        <h1>Starting VM</h1>
        <p class="vm-name">{{.VMName}}</p>
        <p>Waiting for port {{.Port}} to become available...</p>
        <div class="info">
            <p>This page will refresh automatically.</p>
            <p>The VM is being started and may take a moment.</p>
        </div>
    </div>
</body>
</html>`

var loadingTmpl = template.Must(template.New("loading").Parse(loadingHTML))

// handleWakeOnRequest starts a stopped VM and serves a loading page
func (r *Router) handleWakeOnRequest(w http.ResponseWriter, req *http.Request, vmName string, port int) {
	// Check if already waking this VM
	if _, waking := r.waking.LoadOrStore(vmName, true); waking {
		// Already waking, just serve loading page
		r.serveLoadingPage(w, vmName, port)
		return
	}

	// Start waking in background
	go func() {
		defer r.waking.Delete(vmName)

		// Start the VM
		if err := r.mp.Start(vmName); err != nil {
			// Log error but don't block
			return
		}

		// Wait for port to be ready
		r.waitForPort(vmName, port, 90*time.Second)
	}()

	// Serve loading page immediately
	r.serveLoadingPage(w, vmName, port)
}

// serveLoadingPage renders the loading page
func (r *Router) serveLoadingPage(w http.ResponseWriter, vmName string, port int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	loadingTmpl.Execute(w, map[string]interface{}{
		"VMName": vmName,
		"Port":   port,
	})
}

// waitForPort polls until the VM port is accepting connections
func (r *Router) waitForPort(vmName string, port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Get current VM info (IP might change after start)
		info, err := r.mp.Info(vmName)
		if err != nil || len(info.IPv4) == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		// Try to connect to the port
		addr := fmt.Sprintf("%s:%d", info.IPv4[0], port)
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}

		time.Sleep(1 * time.Second)
	}

	return false
}
