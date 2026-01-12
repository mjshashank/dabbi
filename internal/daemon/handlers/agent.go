package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/agent"
)

// AgentHandler handles agent URL requests
type AgentHandler struct {
	am *agent.Manager
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(am *agent.Manager) *AgentHandler {
	return &AgentHandler{am: am}
}

// GetURL returns the URL to access the agent for a VM
// It starts the agent listener if not already running
func (h *AgentHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	if vmName == "" {
		http.Error(w, "VM name required", http.StatusBadRequest)
		return
	}

	url, err := h.am.GetURL(vmName, r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": url,
	})
}
