package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/tunnel"
)

// TunnelHandler handles tunnel-related API requests
type TunnelHandler struct {
	tm *tunnel.Manager
}

// NewTunnelHandler creates a new tunnel handler
func NewTunnelHandler(tm *tunnel.Manager) *TunnelHandler {
	return &TunnelHandler{tm: tm}
}

// TunnelInfo represents tunnel information in API responses
type TunnelInfo struct {
	HostPort int    `json:"host_port"`
	VMName   string `json:"vm_name"`
	VMPort   int    `json:"vm_port"`
}

// List returns all active tunnels
func (h *TunnelHandler) List(w http.ResponseWriter, r *http.Request) {
	tunnels := h.tm.List()

	var info []TunnelInfo
	for _, t := range tunnels {
		info = append(info, TunnelInfo{
			HostPort: t.HostPort,
			VMName:   t.VMName,
			VMPort:   t.VMPort,
		})
	}

	respondJSON(w, http.StatusOK, info)
}

// CreateTunnelRequest represents a tunnel creation request
type CreateTunnelRequest struct {
	VMName string `json:"vm_name"`
	VMPort int    `json:"vm_port"`
}

// Create creates a new tunnel
func (h *TunnelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	if req.VMName == "" || req.VMPort == 0 {
		http.Error(w, `{"error": "vm_name and vm_port are required"}`, http.StatusBadRequest)
		return
	}

	t, err := h.tm.Create(req.VMName, req.VMPort)
	if err != nil {
		// Return 400 for user errors like VM not running
		if strings.Contains(err.Error(), "not running") {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusCreated, TunnelInfo{
		HostPort: t.HostPort,
		VMName:   t.VMName,
		VMPort:   t.VMPort,
	})
}

// Delete closes a tunnel
func (h *TunnelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	portStr := chi.URLParam(r, "port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, `{"error": "invalid port"}`, http.StatusBadRequest)
		return
	}

	if err := h.tm.Delete(port); err != nil {
		respondError(w, http.StatusNotFound, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}
