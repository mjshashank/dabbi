package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/multipass"
)

// FileHandler handles file-related API requests
type FileHandler struct {
	mp multipass.Client
}

// NewFileHandler creates a new file handler
func NewFileHandler(mp multipass.Client) *FileHandler {
	return &FileHandler{mp: mp}
}

// FileEntry represents a file or directory in the browser
type FileEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
	Mode  string `json:"mode"`
}

// Browse lists files in a directory or returns file content
func (h *FileHandler) Browse(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	path := r.URL.Query().Get("path")

	if path == "" {
		path = "/home/ubuntu"
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

	// List directory contents using exec
	output, err := h.mp.Exec(vmName, "ls", "-la", path)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	// Parse ls output
	entries := parseLsOutput(output)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"path":    path,
		"entries": entries,
	})
}

// parseLsOutput parses the output of ls -la
func parseLsOutput(output string) []FileEntry {
	var entries []FileEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		name := strings.Join(fields[8:], " ")
		if name == "." || name == ".." {
			continue
		}

		entry := FileEntry{
			Name:  name,
			IsDir: fields[0][0] == 'd',
			Mode:  fields[0],
		}

		// Parse size (field 4)
		var size int64
		fmt.Sscanf(fields[4], "%d", &size)
		entry.Size = size

		entries = append(entries, entry)
	}

	return entries
}

// Upload handles file uploads to a VM
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	targetPath := r.URL.Query().Get("path")

	if targetPath == "" {
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

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	// Create temp file on host
	tmpFile, err := os.CreateTemp("", "dabbi-upload-*")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy uploaded file to temp
	if _, err := io.Copy(tmpFile, file); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	tmpFile.Close()

	// Determine full target path
	fullPath := targetPath
	if strings.HasSuffix(targetPath, "/") {
		fullPath = filepath.Join(targetPath, header.Filename)
	}

	// Transfer to VM
	vmPath := fmt.Sprintf("%s:%s", vmName, fullPath)
	if err := h.mp.Transfer(tmpFile.Name(), vmPath); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "uploaded",
		"path":   fullPath,
	})
}

// Download handles file downloads from a VM
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	vmName := chi.URLParam(r, "name")
	filePath := r.URL.Query().Get("path")

	if filePath == "" {
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

	// Create temp file on host
	tmpFile, err := os.CreateTemp("", "dabbi-download-*")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Transfer from VM to host
	vmPath := fmt.Sprintf("%s:%s", vmName, filePath)
	if err := h.mp.Transfer(vmPath, tmpFile.Name()); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	// Read the downloaded file
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	// Set headers for download
	filename := filepath.Base(filePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Write(content)
}
