package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		host       string
		shouldPass bool
	}{
		// Cases that should ALLOW
		{
			name:       "no_origin_allows",
			origin:     "",
			host:       "localhost:8080",
			shouldPass: true,
		},
		{
			name:       "localhost_allowed",
			origin:     "http://localhost:3000",
			host:       "example.com:8080",
			shouldPass: true,
		},
		{
			name:       "localhost_no_port_allowed",
			origin:     "http://localhost",
			host:       "example.com:8080",
			shouldPass: true,
		},
		{
			name:       "127.0.0.1_allowed",
			origin:     "http://127.0.0.1:3000",
			host:       "example.com:8080",
			shouldPass: true,
		},
		{
			name:       "127.0.0.1_no_port_allowed",
			origin:     "http://127.0.0.1",
			host:       "example.com:8080",
			shouldPass: true,
		},
		{
			name:       "ipv6_localhost_allowed",
			origin:     "http://[::1]:3000",
			host:       "example.com:8080",
			shouldPass: true,
		},
		{
			name:       "same_origin_allowed",
			origin:     "http://dabbi.local:8080",
			host:       "dabbi.local:8080",
			shouldPass: true,
		},
		{
			name:       "same_origin_different_port_allowed",
			origin:     "http://dabbi.local:3000",
			host:       "dabbi.local:8080",
			shouldPass: true,
		},
		{
			name:       "same_origin_no_host_port",
			origin:     "http://myapp.example.com",
			host:       "myapp.example.com",
			shouldPass: true,
		},
		{
			name:       "https_localhost_allowed",
			origin:     "https://localhost:443",
			host:       "example.com",
			shouldPass: true,
		},

		// Cases that should BLOCK
		{
			name:       "different_origin_blocked",
			origin:     "http://evil.com",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "different_origin_with_port_blocked",
			origin:     "http://evil.com:3000",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "subdomain_blocked",
			origin:     "http://sub.localhost",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "localhost_in_domain_blocked",
			origin:     "http://notlocalhost.com",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "localhost_suffix_blocked",
			origin:     "http://evil-localhost",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "invalid_origin_url_blocked",
			origin:     "not-a-url",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "empty_hostname_in_origin_blocked",
			origin:     "http://",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "attacker_localhost_subdomain_blocked",
			origin:     "http://localhost.attacker.com",
			host:       "localhost:8080",
			shouldPass: false,
		},
		{
			name:       "different_host_blocked",
			origin:     "http://other.example.com",
			host:       "myapp.example.com:8080",
			shouldPass: false,
		},
		{
			name:       "origin_127_0_0_2_blocked",
			origin:     "http://127.0.0.2",
			host:       "localhost:8080",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/vms/test/shell", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			req.Host = tt.host

			result := checkOrigin(req)
			assert.Equal(t, tt.shouldPass, result, "checkOrigin should return %v for origin=%q, host=%q", tt.shouldPass, tt.origin, tt.host)
		})
	}
}

func TestShellHandler_VMNotFound(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "nonexistent-vm").Return(nil, errors.New("VM not found"))

	handler := NewShellHandler(mockMP)

	req := httptest.NewRequest(http.MethodGet, "/api/vms/nonexistent-vm/shell", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "nonexistent-vm")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "VM not found")
	mockMP.AssertExpectations(t)
}

func TestShellHandler_VMNotRunning(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "stopped-vm").Return(testutil.StoppedVM("stopped-vm"), nil)

	handler := NewShellHandler(mockMP)

	req := httptest.NewRequest(http.MethodGet, "/api/vms/stopped-vm/shell", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "stopped-vm")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "not running")
	mockMP.AssertExpectations(t)
}

func TestShellHandler_ParsesInitialSize(t *testing.T) {
	// This test verifies the handler correctly parses query params
	// We can't fully test WebSocket upgrade without a real server,
	// but we can test the VM validation part

	tests := []struct {
		name   string
		cols   string
		rows   string
		vmName string
	}{
		{"default_size", "", "", "test-vm"},
		{"custom_size", "120", "40", "test-vm"},
		{"invalid_cols", "abc", "24", "test-vm"},
		{"invalid_rows", "80", "xyz", "test-vm"},
		{"zero_cols", "0", "24", "test-vm"},
		{"negative_rows", "80", "-10", "test-vm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP := new(testutil.MockMultipassClient)
			mockMP.On("Info", tt.vmName).Return(testutil.StoppedVM(tt.vmName), nil)

			handler := NewShellHandler(mockMP)

			url := "/api/vms/" + tt.vmName + "/shell"
			if tt.cols != "" || tt.rows != "" {
				url += "?"
				if tt.cols != "" {
					url += "cols=" + tt.cols + "&"
				}
				if tt.rows != "" {
					url += "rows=" + tt.rows
				}
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.vmName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			// VM is stopped so it should return error (which is expected)
			// We're just testing that the handler handles various query params without crashing
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestNewShellHandler(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	handler := NewShellHandler(mockMP)

	require.NotNil(t, handler)
	assert.Equal(t, mockMP, handler.mp)
}
