package proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHost(t *testing.T) {
	r := NewRouter(nil)

	tests := []struct {
		host     string
		wantVM   string
		wantPort int
		wantOK   bool
	}{
		// Valid patterns
		{"myvm-8080.localhost", "myvm", 8080, true},
		{"myvm-8080.localhost:3000", "myvm", 8080, true},
		{"my-vm-8080.example.com", "my-vm", 8080, true},
		{"vm123-443.localhost", "vm123", 443, true},
		{"dev-vm-3000.mydomain.io", "dev-vm", 3000, true},
		{"test-8080.localhost:80", "test", 8080, true},
		{"a-1.localhost", "a", 1, true},
		{"vm-65535.example.com", "vm", 65535, true},
		{"my-multi-dash-vm-8080.localhost", "my-multi-dash-vm", 8080, true},
		{"VM1-8080.localhost", "VM1", 8080, true}, // uppercase allowed

		// Invalid patterns
		{"localhost:8080", "", 0, false},           // No VM pattern
		{"myvm.localhost", "", 0, false},           // No port in name
		{"myvm-abc.localhost", "", 0, false},       // Invalid port
		{"-8080.localhost", "", 0, false},          // Empty VM name (starts with dash)
		{"myvm-8080", "", 0, false},                // No domain
		{"8080.localhost", "", 0, false},           // No VM name before port
		{"myvm-8080.", "", 0, false},               // Trailing dot only
		{"", "", 0, false},                         // Empty
		{"myvm.8080.localhost", "", 0, false},      // Wrong format (. instead of -)
		{"myvm-8080-extra.localhost", "", 0, false}, // Extra suffix after port
		{"myvm--8080.localhost", "myvm-", 8080, true}, // Double dash allowed (VM name ends with dash)
		{".myvm-8080.localhost", "", 0, false},     // Leading dot
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			vm, port, ok := r.parseHost(tt.host)
			assert.Equal(t, tt.wantOK, ok, "parseHost(%q) ok mismatch", tt.host)
			if ok {
				assert.Equal(t, tt.wantVM, vm, "parseHost(%q) vm mismatch", tt.host)
				assert.Equal(t, tt.wantPort, port, "parseHost(%q) port mismatch", tt.host)
			}
		})
	}
}

func TestRouter_Middleware_PassesThrough(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	r := NewRouter(mockMP)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Request without VM pattern in host
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()

	r.Middleware(next).ServeHTTP(rec, req)

	assert.True(t, nextCalled, "Next handler should be called for non-VM requests")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_Middleware_VMNotFound(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "nonexistent").Return(nil, errors.New("VM not found"))

	r := NewRouter(mockMP)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next should not be called for VM requests")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "nonexistent-8080.localhost"
	rec := httptest.NewRecorder()

	r.Middleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "not found")
	mockMP.AssertExpectations(t)
}

func TestRouter_HandleVMRequest_Running(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "test-vm").Return(testutil.RunningVM("test-vm", "192.168.64.5"), nil)

	r := NewRouter(mockMP)

	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	// Note: We can't easily test the actual proxy without a real backend
	// This test verifies the flow up to the proxy call
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "test-vm-8080.localhost"
	rec := httptest.NewRecorder()

	// The actual proxy will fail since we're not running a real backend at the VM IP
	// But we can verify the mock was called correctly
	r.handleVMRequest(rec, req, "test-vm", 8080)

	// Expect BadGateway since there's no real backend at the IP
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	mockMP.AssertExpectations(t)
}

func TestRouter_HandleVMRequest_Stopped(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "stopped-vm").Return(testutil.StoppedVM("stopped-vm"), nil)
	// Wake-on-request will call Start in a goroutine
	mockMP.On("Start", "stopped-vm").Return(nil).Maybe()
	// After starting, it may check Info again
	mockMP.On("Info", "stopped-vm").Return(testutil.RunningVM("stopped-vm", "192.168.64.5"), nil).Maybe()

	r := NewRouter(mockMP)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "stopped-vm-8080.localhost"
	rec := httptest.NewRecorder()

	// Stopped VM should trigger wake-on-request
	// The wake handler returns a "Waking up" page
	r.handleVMRequest(rec, req, "stopped-vm", 8080)

	// Should show waking page
	// Note: This is a simplified test - the actual wake process is async
}

func TestRouter_HandleVMRequest_NoIP(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)

	// Return running VM with no IP
	vmInfo := &multipass.InstanceInfo{
		State: multipass.StateRunning,
		IPv4:  []string{}, // No IP
	}
	mockMP.On("Info", "no-ip-vm").Return(vmInfo, nil)

	r := NewRouter(mockMP)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r.handleVMRequest(rec, req, "no-ip-vm", 8080)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "no IP address")
	mockMP.AssertExpectations(t)
}

func TestRouter_HandleVMRequest_UnexpectedState(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)

	vmInfo := &multipass.InstanceInfo{
		State: "Starting", // Not a standard state we handle
		IPv4:  []string{"192.168.64.5"},
	}
	mockMP.On("Info", "starting-vm").Return(vmInfo, nil)

	r := NewRouter(mockMP)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r.handleVMRequest(rec, req, "starting-vm", 8080)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "unexpected state")
	mockMP.AssertExpectations(t)
}

func TestNewRouter(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	r := NewRouter(mockMP)

	require.NotNil(t, r)
	assert.Equal(t, mockMP, r.mp)
}

func TestRouter_HostPatternExamples(t *testing.T) {
	// Real-world examples of host patterns
	r := NewRouter(nil)

	examples := []struct {
		desc   string
		host   string
		wantVM string
		wantOK bool
	}{
		{"React dev server", "myapp-3000.localhost", "myapp", true},
		{"PostgreSQL", "dbvm-5432.localhost", "dbvm", true},
		{"MySQL", "mysql-3306.myhost.com:80", "mysql", true},
		{"Web server", "web-80.localhost", "web", true},
		{"HTTPS port", "secure-443.localhost", "secure", true},
		{"High port", "node-49152.localhost", "node", true},
		{"With domain", "api-8080.dev.example.com", "api", true},
	}

	for _, ex := range examples {
		t.Run(ex.desc, func(t *testing.T) {
			vm, _, ok := r.parseHost(ex.host)
			assert.Equal(t, ex.wantOK, ok)
			if ok {
				assert.Equal(t, ex.wantVM, vm)
			}
		})
	}
}
