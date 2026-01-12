package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/multipass"
)

// MountHandler handles mount-related API requests
type MountHandler struct {
	mp multipass.Client
}

// NewMountHandler creates a new mount handler
func NewMountHandler(mp multipass.Client) *MountHandler {
	return &MountHandler{mp: mp}
}

// MountEntry represents a mount point
type MountEntry struct {
	HostPath string `json:"host_path"`
	VMPath   string `json:"vm_path"`
}

// List returns all mounts for a VM
func (h *MountHandler) List(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")

	info, err := h.mp.Info(vmName)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	var mounts []MountEntry
	for vmPath, mount := range info.Mounts {
		mounts = append(mounts, MountEntry{
			HostPath: mount.SourcePath,
			VMPath:   vmPath,
		})
	}

	respondJSON(w, http.StatusOK, mounts)
}

// AddMountRequest represents a mount add request
type AddMountRequest struct {
	HostPath string `json:"host_path"`
	VMPath   string `json:"vm_path"`
}

// Add creates a new mount
func (h *MountHandler) Add(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")

	var req AddMountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if req.HostPath == "" || req.VMPath == "" {
		http.Error(w, `{"error": "host_path and vm_path are required"}`, http.StatusBadRequest)
		return
	}

	// Check VM is running
	info, err := h.mp.Info(vmName)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}
	if info.State != multipass.StateRunning {
		http.Error(w, `{"error": "VM is not running"}`, http.StatusBadRequest)
		return
	}

	if err := h.mp.Mount(vmName, req.HostPath, req.VMPath); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"status": "mounted"})
}

// Remove removes a mount
func (h *MountHandler) Remove(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	vmPath := r.URL.Query().Get("path")

	if vmPath == "" {
		http.Error(w, `{"error": "path query parameter is required"}`, http.StatusBadRequest)
		return
	}

	// Check VM is running
	info, err := h.mp.Info(vmName)
	if err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}
	if info.State != multipass.StateRunning {
		http.Error(w, `{"error": "VM is not running"}`, http.StatusBadRequest)
		return
	}

	if err := h.mp.Unmount(vmName, vmPath); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "unmounted"})
}
