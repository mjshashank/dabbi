package daemon

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mjshashank/dabbi/internal/agent"
	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/daemon/handlers"
	authMw "github.com/mjshashank/dabbi/internal/daemon/mw"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/proxy"
	"github.com/mjshashank/dabbi/internal/tunnel"
	"github.com/mjshashank/dabbi/internal/ui"
)

// SetupRouter configures and returns the HTTP router
func SetupRouter(
	cfg *config.Config,
	mp multipass.Client,
	tm *tunnel.Manager,
	pr *proxy.Router,
	am *agent.Manager,
) http.Handler {
	return SetupRouterWithTLS(cfg, mp, tm, pr, am, false, "")
}

// SetupRouterWithTLS configures and returns the HTTP router with TLS awareness
func SetupRouterWithTLS(
	cfg *config.Config,
	mp multipass.Client,
	tm *tunnel.Manager,
	pr *proxy.Router,
	am *agent.Manager,
	useTLS bool,
	domain string,
) http.Handler {
	r := chi.NewRouter()

	// Configure proxy router with auth token for protected ports
	pr.SetAuthToken(cfg.AuthToken)

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Proxy router handles VM traffic based on Host header
	// This MUST be first to intercept VM requests before API routes
	r.Use(pr.Middleware)

	// Auth endpoints (not protected)
	r.Post("/api/auth/login", authMw.LoginHandler(cfg.AuthToken, useTLS))
	r.Post("/api/auth/logout", authMw.LogoutHandler())

	// API routes (protected by auth)
	r.Route("/api", func(r chi.Router) {
		r.Use(authMw.BearerAuth(cfg.AuthToken))

		// VMs
		vmHandler := handlers.NewVMHandler(mp, cfg)
		r.Get("/defaults", vmHandler.Defaults)
		r.Get("/vms", vmHandler.List)
		r.Post("/vms", vmHandler.Create)
		r.Get("/vms/{name}", vmHandler.Get)
		r.Delete("/vms/{name}", vmHandler.Delete)
		r.Post("/vms/{name}/state", vmHandler.ChangeState)
		r.Post("/vms/{name}/clone", vmHandler.Clone)

		// Snapshots
		snapHandler := handlers.NewSnapshotHandler(mp)
		r.Get("/vms/{name}/snapshots", snapHandler.List)
		r.Post("/vms/{name}/snapshots", snapHandler.Create)
		r.Post("/vms/{name}/snapshots/restore", snapHandler.Restore)
		r.Delete("/vms/{name}/snapshots/{snap}", snapHandler.Delete)

		// Files
		fileHandler := handlers.NewFileHandler(mp)
		r.Get("/vms/{name}/files", fileHandler.Browse)
		r.Post("/vms/{name}/files", fileHandler.Upload)
		r.Get("/vms/{name}/files/download", fileHandler.Download)

		// Mounts
		mountHandler := handlers.NewMountHandler(mp)
		r.Get("/vms/{name}/mounts", mountHandler.List)
		r.Post("/vms/{name}/mounts", mountHandler.Add)
		r.Delete("/vms/{name}/mounts", mountHandler.Remove)

		// Tunnels
		tunnelHandler := handlers.NewTunnelHandler(tm)
		r.Get("/tunnels", tunnelHandler.List)
		r.Post("/tunnels", tunnelHandler.Create)
		r.Delete("/tunnels/{port}", tunnelHandler.Delete)

		// Network configuration
		networkHandler := handlers.NewNetworkHandler(mp, cfg)
		r.Get("/vms/{name}/network", networkHandler.Get)
		r.Put("/vms/{name}/network", networkHandler.Update)
		r.Delete("/vms/{name}/network", networkHandler.Remove)
		r.Post("/vms/{name}/network/apply", networkHandler.Apply)
		r.Get("/network/defaults", networkHandler.GetDefaults)
		r.Put("/network/defaults", networkHandler.SetDefaults)

		// Shell (WebSocket)
		shellHandler := handlers.NewShellHandler(mp)
		r.Get("/vms/{name}/shell", shellHandler.Handle)

		// Agent (opencode) - returns URL to access agent via subdomain proxy
		agentHandler := handlers.NewAgentHandler(am, domain, cfg.AuthToken, useTLS)
		r.Get("/vms/{name}/agent-url", agentHandler.GetURL)
	})

	// Health check (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Embedded UI (fallback for all other routes)
	r.Handle("/*", ui.Handler())

	return r
}
