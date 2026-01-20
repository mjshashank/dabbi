package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/agent"
)

const agentPort = 1234 // OpenCode port inside VM

// AgentHandler handles agent URL requests
type AgentHandler struct {
	am        *agent.Manager
	domain    string
	authToken string
	useTLS    bool
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(am *agent.Manager, domain, authToken string, useTLS bool) *AgentHandler {
	return &AgentHandler{
		am:        am,
		domain:    domain,
		authToken: authToken,
		useTLS:    useTLS,
	}
}

// GetURL returns the URL to access the agent for a VM
// When TLS is enabled, returns subdomain-based HTTPS URL to avoid WebSocket bugs
func (h *AgentHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	if vmName == "" {
		http.Error(w, "VM name required", http.StatusBadRequest)
		return
	}

	// Verify VM exists and is running
	if err := h.am.VerifyVM(vmName); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	var agentURL string
	if h.useTLS && h.domain != "" {
		// Subdomain-based HTTPS URL: https://<vm>-1234.<domain>?token=xxx
		agentURL = fmt.Sprintf("https://%s-%d.%s/?token=%s",
			vmName, agentPort, h.domain, url.QueryEscape(h.authToken))
	} else {
		// Fallback: use the old port-based HTTP URL
		var err error
		agentURL, err = h.am.GetURL(vmName, r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": agentURL,
	})
}
