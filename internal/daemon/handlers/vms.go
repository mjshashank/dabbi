package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/network"
)

// VMHandler handles VM-related API requests
type VMHandler struct {
	mp  multipass.Client
	cfg *config.Config
}

// NewVMHandler creates a new VM handler
func NewVMHandler(mp multipass.Client, cfg *config.Config) *VMHandler {
	return &VMHandler{mp: mp, cfg: cfg}
}

// Defaults returns the default VM configuration values
func (h *VMHandler) Defaults(w http.ResponseWriter, r *http.Request) {
	cpu := h.cfg.Defaults.CPU
	if cpu == 0 {
		cpu = 2
	}
	mem := h.cfg.Defaults.Mem
	if mem == "" {
		mem = "4G"
	}
	disk := h.cfg.Defaults.Disk
	if disk == "" {
		disk = "20G"
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cpu":  cpu,
		"mem":  mem,
		"disk": disk,
	})
}

// List returns all VMs
func (h *VMHandler) List(w http.ResponseWriter, r *http.Request) {
	vms, err := h.mp.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, vms)
}

// Get returns details for a single VM
func (h *VMHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	info, err := h.mp.Info(name)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	respondJSON(w, http.StatusOK, info)
}

// CreateVMRequest represents a VM creation request
type CreateVMRequest struct {
	Name      string                   `json:"name"`
	CPUs      int                      `json:"cpu,omitempty"`
	Memory    string                   `json:"mem,omitempty"`
	Disk      string                   `json:"disk,omitempty"`
	CloudInit string                   `json:"cloud_init,omitempty"`
	Image     string                   `json:"image,omitempty"`
	Network   *multipass.NetworkConfig `json:"network,omitempty"`
}

// Create creates a new VM
func (h *VMHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error": "name is required"}`, http.StatusBadRequest)
		return
	}

	// Set defaults if not provided
	if req.CPUs == 0 {
		req.CPUs = h.cfg.Defaults.CPU
		if req.CPUs == 0 {
			req.CPUs = 2
		}
	}
	if req.Memory == "" {
		req.Memory = h.cfg.Defaults.Mem
		if req.Memory == "" {
			req.Memory = "4G"
		}
	}
	if req.Disk == "" {
		req.Disk = h.cfg.Defaults.Disk
		if req.Disk == "" {
			req.Disk = "20G"
		}
	}

	// Resolve cloud-init path (explicit > config default > ~/.dabbi/cloud-init.yaml)
	resolvedCloudInit := h.cfg.GetCloudInitPath(req.CloudInit)

	// Handle network config
	netConfig := req.Network
	if netConfig == nil && h.cfg.Defaults.NetworkConfig != nil && h.cfg.Defaults.NetworkConfig.Mode != multipass.NetworkModeNone {
		netConfig = h.cfg.Defaults.NetworkConfig
	}

	// Validate network config if provided
	if netConfig != nil && netConfig.Mode != multipass.NetworkModeNone {
		if err := network.ValidateConfig(netConfig); err != nil {
			http.Error(w, `{"error": "invalid network config: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	}

	// Read base cloud-init content
	var baseContent string
	if resolvedCloudInit != "" {
		data, err := os.ReadFile(resolvedCloudInit)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		baseContent = string(data)
	} else {
		baseContent = config.DefaultCloudInit
	}

	// Inject auth token into cloud-init (replaces __DABBI_AUTH_TOKEN__ placeholder)
	modifiedContent := config.GenerateCloudInitWithAuthToken(baseContent, h.cfg.AuthToken)

	// Generate cloud-init with network config if needed
	if netConfig != nil && netConfig.Mode != multipass.NetworkModeNone {
		var err error
		modifiedContent, err = config.GenerateCloudInitWithNetwork(modifiedContent, netConfig)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
	}

	// Write to temp file
	tmpDir, err := os.MkdirTemp("", "dabbi-cloudinit-*")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	tempCloudInitFile := filepath.Join(tmpDir, "cloud-init.yaml")
	if err := os.WriteFile(tempCloudInitFile, []byte(modifiedContent), 0644); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	finalCloudInit := tempCloudInitFile

	opts := multipass.LaunchOptions{
		Name:          req.Name,
		CPUs:          req.CPUs,
		Memory:        req.Memory,
		Disk:          req.Disk,
		CloudInit:     finalCloudInit,
		Image:         req.Image,
		NetworkConfig: netConfig,
	}

	// Launch VM synchronously so we can return errors to the user
	if err := h.mp.Launch(opts); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"status": "created",
		"name":   req.Name,
	})
}

// Delete removes a VM
func (h *VMHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.mp.Delete(name, true); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// StateChangeRequest represents a state change request
type StateChangeRequest struct {
	Action string `json:"action"` // "start" or "stop"
}

// ChangeState changes the state of a VM
func (h *VMHandler) ChangeState(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req StateChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = h.mp.Start(name)
	case "stop":
		err = h.mp.Stop(name)
	case "restart":
		err = h.mp.Restart(name)
	default:
		http.Error(w, `{"error": "invalid action, must be 'start', 'stop', or 'restart'"}`, http.StatusBadRequest)
		return
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": req.Action + "ed"})
}

// CloneRequest represents a clone request
type CloneRequest struct {
	NewName string `json:"new_name"`
}

// Clone creates a copy of a VM
func (h *VMHandler) Clone(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req CloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if req.NewName == "" {
		http.Error(w, `{"error": "new_name is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.mp.Clone(name, req.NewName); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"status": "cloned",
		"name":   req.NewName,
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
