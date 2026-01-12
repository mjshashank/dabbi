package daemon

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/mjshashank/dabbi/internal/agent"
	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/proxy"
	"github.com/mjshashank/dabbi/internal/tunnel"
	"github.com/mjshashank/dabbi/internal/watchdog"
	"golang.org/x/crypto/acme/autocert"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            int
	Domain          string
	Config          *config.Config
	MultipassClient multipass.Client
}

// Server represents the dabbi daemon
type Server struct {
	cfg      ServerConfig
	router   http.Handler
	watchdog *watchdog.Watchdog
	tunnels  *tunnel.Manager
	proxy    *proxy.Router
	agents   *agent.Manager
}

// NewServer creates a new daemon server
func NewServer(cfg ServerConfig) *Server {
	timeout := time.Duration(cfg.Config.ShutdownTimeoutMins) * time.Minute
	wd := watchdog.New(cfg.MultipassClient, timeout)
	tm := tunnel.NewManager(cfg.MultipassClient)
	pr := proxy.NewRouter(cfg.MultipassClient)
	am := agent.NewManager(cfg.MultipassClient)

	// Use TLS-aware router when domain is configured
	useTLS := cfg.Domain != ""
	router := SetupRouterWithTLS(cfg.Config, cfg.MultipassClient, tm, pr, am, useTLS)

	return &Server{
		cfg:      cfg,
		router:   router,
		watchdog: wd,
		tunnels:  tm,
		proxy:    pr,
		agents:   am,
	}
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)

	if s.cfg.Domain != "" {
		return s.listenTLS()
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv.ListenAndServe()
}

// listenTLS starts an HTTPS server with Let's Encrypt
func (s *Server) listenTLS() error {
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.cfg.Domain),
		Cache:      autocert.DirCache(".dabbi-certs"),
	}

	srv := &http.Server{
		Addr:    ":443",
		Handler: s.router,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
			MinVersion:     tls.VersionTLS12,
		},
	}

	// HTTP redirect server (also handles ACME challenges)
	go func() {
		httpSrv := &http.Server{
			Addr:    ":80",
			Handler: certManager.HTTPHandler(nil),
		}
		httpSrv.ListenAndServe()
	}()

	return srv.ListenAndServeTLS("", "")
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.watchdog.Stop()
	s.agents.StopAll()
	return nil
}
