package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/network"
)

// NetworkHandler handles network configuration API requests
type NetworkHandler struct {
	mp      multipass.Client
	cfg     *config.Config
	applier *network.Applier
}

// NewNetworkHandler creates a new network handler
func NewNetworkHandler(mp multipass.Client, cfg *config.Config) *NetworkHandler {
	return &NetworkHandler{
		mp:      mp,
		cfg:     cfg,
		applier: network.NewApplier(mp),
	}
}

// NetworkConfigRequest represents a network configuration update request
type NetworkConfigRequest struct {
	Mode  string                  `json:"mode"`  // "none", "allowlist", "blocklist", "isolated"
	Rules []multipass.NetworkRule `json:"rules"` // Rules (ignored for "isolated" and "none")
}

// NetworkConfigResponse represents the current network configuration
type NetworkConfigResponse struct {
	Mode  string                  `json:"mode"`
	Rules []multipass.NetworkRule `json:"rules,omitempty"`
}

// Get returns the current network configuration for a VM
// GET /api/vms/{name}/network
func (h *NetworkHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Verify VM exists and is running
	info, err := h.mp.Info(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	if info.State != multipass.StateRunning {
		http.Error(w, `{"error": "VM must be running to query network config"}`, http.StatusBadRequest)
		return
	}

	// Query the VM for current config
	cfg, err := h.applier.GetCurrentConfig(name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if cfg == nil {
		// No config = no restrictions
		respondJSON(w, http.StatusOK, NetworkConfigResponse{
			Mode:  string(multipass.NetworkModeNone),
			Rules: nil,
		})
		return
	}

	respondJSON(w, http.StatusOK, NetworkConfigResponse{
		Mode:  string(cfg.Mode),
		Rules: cfg.Rules,
	})
}

// Update sets or updates the network configuration for a VM
// PUT /api/vms/{name}/network
func (h *NetworkHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req NetworkConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// Verify VM exists and is running
	info, err := h.mp.Info(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	if info.State != multipass.StateRunning {
		http.Error(w, `{"error": "VM must be running to update network config"}`, http.StatusBadRequest)
		return
	}

	// Build config
	cfg := &multipass.NetworkConfig{
		Mode:  multipass.NetworkMode(req.Mode),
		Rules: req.Rules,
	}

	// Validate
	if err := network.ValidateConfig(cfg); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Apply to VM
	if err := h.applier.ApplyToVM(name, cfg); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "applied",
		"mode":   req.Mode,
	})
}

// Remove removes all network restrictions from a VM
// DELETE /api/vms/{name}/network
func (h *NetworkHandler) Remove(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Verify VM exists and is running
	info, err := h.mp.Info(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	if info.State != multipass.StateRunning {
		http.Error(w, `{"error": "VM must be running to remove network config"}`, http.StatusBadRequest)
		return
	}

	// Apply "none" mode
	if err := h.applier.RemoveFromVM(name); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "removed",
	})
}

// Apply re-applies the current network configuration
// POST /api/vms/{name}/network/apply
func (h *NetworkHandler) Apply(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Verify VM exists and is running
	info, err := h.mp.Info(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	if info.State != multipass.StateRunning {
		http.Error(w, `{"error": "VM must be running to apply network config"}`, http.StatusBadRequest)
		return
	}

	// Get current config from VM
	cfg, err := h.applier.GetCurrentConfig(name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if cfg == nil {
		http.Error(w, `{"error": "no network config to apply"}`, http.StatusBadRequest)
		return
	}

	// Re-apply
	if err := h.applier.ApplyToVM(name, cfg); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "applied",
		"mode":   string(cfg.Mode),
	})
}

// GetDefaults returns the global default network configuration
// GET /api/network/defaults
func (h *NetworkHandler) GetDefaults(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfg.Defaults.NetworkConfig
	if cfg == nil {
		respondJSON(w, http.StatusOK, NetworkConfigResponse{
			Mode:  string(multipass.NetworkModeNone),
			Rules: nil,
		})
		return
	}

	respondJSON(w, http.StatusOK, NetworkConfigResponse{
		Mode:  string(cfg.Mode),
		Rules: cfg.Rules,
	})
}

// SetDefaults updates the global default network configuration
// PUT /api/network/defaults
func (h *NetworkHandler) SetDefaults(w http.ResponseWriter, r *http.Request) {
	var req NetworkConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// Build config
	cfg := &multipass.NetworkConfig{
		Mode:  multipass.NetworkMode(req.Mode),
		Rules: req.Rules,
	}

	// Validate
	if err := network.ValidateConfig(cfg); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Update config
	h.cfg.Defaults.NetworkConfig = cfg

	// Save to disk
	if err := h.cfg.Save(); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "saved",
		"mode":   req.Mode,
	})
}
