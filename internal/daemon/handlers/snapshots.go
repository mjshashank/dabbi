package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/multipass"
)

// SnapshotHandler handles snapshot-related API requests
type SnapshotHandler struct {
	mp multipass.Client
}

// NewSnapshotHandler creates a new snapshot handler
func NewSnapshotHandler(mp multipass.Client) *SnapshotHandler {
	return &SnapshotHandler{mp: mp}
}

// List returns all snapshots for a VM
func (h *SnapshotHandler) List(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")

	snapshots, err := h.mp.ListSnapshots(vmName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, snapshots)
}

// CreateSnapshotRequest represents a snapshot creation request
type CreateSnapshotRequest struct {
	Name string `json:"name,omitempty"`
}

// Create creates a new snapshot
func (h *SnapshotHandler) Create(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")

	var req CreateSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.mp.CreateSnapshot(vmName, req.Name); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// RestoreSnapshotRequest represents a snapshot restore request
type RestoreSnapshotRequest struct {
	SnapshotName string `json:"snapshot_name"`
	Destructive  bool   `json:"destructive,omitempty"`
}

// Restore restores a snapshot
func (h *SnapshotHandler) Restore(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")

	var req RestoreSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if req.SnapshotName == "" {
		http.Error(w, `{"error": "snapshot_name is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.mp.RestoreSnapshot(vmName, req.SnapshotName, req.Destructive); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

// Delete removes a snapshot
func (h *SnapshotHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	snapName := chi.URLParam(r, "snap")

	if err := h.mp.DeleteSnapshot(vmName, snapName); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
