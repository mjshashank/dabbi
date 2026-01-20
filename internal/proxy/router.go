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

const agentPort = 1234 // OpenCode port inside VM

// Router handles HTTP routing to VMs based on Host header
type Router struct {
	mp        multipass.Client
	authToken string
	waking    sync.Map // map[vmName]bool - tracks VMs currently waking
}

// NewRouter creates a new proxy router
func NewRouter(mp multipass.Client) *Router {
	return &Router{
		mp: mp,
	}
}

// SetAuthToken configures the auth token for protected ports
func (r *Router) SetAuthToken(token string) {
	r.authToken = token
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

const agentAuthCookie = "dabbi_agent_token"

// checkAgentAuth validates the auth token for agent requests
// Token can come from: query param, header, or cookie
// Sets a cookie on successful auth so subsequent requests (assets) work
func (r *Router) checkAgentAuth(w http.ResponseWriter, req *http.Request) bool {
	// Try query parameter first
	token := req.URL.Query().Get("token")

	// Try header
	if token == "" {
		token = req.Header.Get("X-Dabbi-Token")
	}

	// Try cookie
	if token == "" {
		if cookie, err := req.Cookie(agentAuthCookie); err == nil {
			token = cookie.Value
		}
	}

	if token != r.authToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	// Set cookie for subsequent requests (assets, etc.)
	// Only set if token came from query param or header (not already from cookie)
	if req.URL.Query().Get("token") != "" || req.Header.Get("X-Dabbi-Token") != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     agentAuthCookie,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})
	}

	return true
}

// handleVMRequest routes a request to the appropriate VM
func (r *Router) handleVMRequest(w http.ResponseWriter, req *http.Request, vmName string, port int) {
	// Auth check for agent port (1234)
	if port == agentPort && r.authToken != "" {
		if !r.checkAgentAuth(w, req) {
			return
		}
	}

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
	targetHost := fmt.Sprintf("%s:%d", vmIP, port)
	target, err := url.Parse(fmt.Sprintf("http://%s", targetHost))
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize director to handle WebSocket upgrades and preserve headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetHost
		// Preserve WebSocket upgrade headers
		if req.Header.Get("Upgrade") != "" {
			req.Header.Set("Connection", "Upgrade")
		}
		// Set forwarded headers
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, req)
}
