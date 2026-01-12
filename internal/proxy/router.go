package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/mjshashank/dabbi/internal/multipass"
)

// Pattern: <vm_name>-<port>.localhost[:port] or <vm_name>-<port>.<domain>[:port]
var hostPattern = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9-]*)-(\d+)\.(localhost|[a-zA-Z0-9.-]+)(:\d+)?$`)

// Router handles HTTP routing to VMs based on Host header
type Router struct {
	mp     multipass.Client
	waking sync.Map // map[vmName]bool - tracks VMs currently waking
}

// NewRouter creates a new proxy router
func NewRouter(mp multipass.Client) *Router {
	return &Router{
		mp: mp,
	}
}

// Middleware returns middleware that routes requests to VMs based on Host header
func (r *Router) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vmName, port, ok := r.parseHost(req.Host)
		if !ok {
			// Not a VM request, pass through to next handler
			next.ServeHTTP(w, req)
			return
		}

		r.handleVMRequest(w, req, vmName, port)
	})
}

// parseHost extracts VM name and port from Host header
func (r *Router) parseHost(host string) (vmName string, port int, ok bool) {
	matches := hostPattern.FindStringSubmatch(host)
	if matches == nil {
		return "", 0, false
	}

	vmName = matches[1]
	port, _ = strconv.Atoi(matches[2])
	return vmName, port, true
}

// handleVMRequest routes a request to the appropriate VM
func (r *Router) handleVMRequest(w http.ResponseWriter, req *http.Request, vmName string, port int) {
	// Get VM info
	info, err := r.mp.Info(vmName)
	if err != nil {
		http.Error(w, fmt.Sprintf("VM '%s' not found", vmName), http.StatusNotFound)
		return
	}

	// Check state and handle accordingly
	switch info.State {
	case multipass.StateStopped, multipass.StateSuspended:
		r.handleWakeOnRequest(w, req, vmName, port)
		return

	case multipass.StateRunning:
		// Get IP
		if len(info.IPv4) == 0 {
			http.Error(w, "VM has no IP address", http.StatusServiceUnavailable)
			return
		}
		r.proxyRequest(w, req, info.IPv4[0], port)

	default:
		http.Error(w, fmt.Sprintf("VM in unexpected state: %s", info.State), http.StatusServiceUnavailable)
	}
}

// proxyRequest forwards the request to the VM using httputil.ReverseProxy
func (r *Router) proxyRequest(w http.ResponseWriter, req *http.Request, vmIP string, port int) {
	target, err := url.Parse(fmt.Sprintf("http://%s:%d", vmIP, port))
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize director to preserve Host header for virtual hosting
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Optionally preserve original host for apps that need it
		// req.Header.Set("X-Forwarded-Host", req.Host)
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, req)
}
