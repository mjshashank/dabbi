package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupVMHandler(t *testing.T) (*VMHandler, *testutil.MockMultipassClient) {
	mockMP := new(testutil.MockMultipassClient)
	cfg := config.DefaultConfig()
	handler := NewVMHandler(mockMP, cfg)
	return handler, mockMP
}

func TestVMHandler_List(t *testing.T) {
	handler, mockMP := setupVMHandler(t)

	tests := []struct {
		name           string
		mockVMs        []multipass.ListInstance
		mockErr        error
		expectedStatus int
		expectedLen    int
	}{
		{
			name: "returns_list_of_vms",
			mockVMs: []multipass.ListInstance{
				{Name: "vm1", State: "Running", IPv4: []string{"192.168.1.1"}},
				{Name: "vm2", State: "Stopped", IPv4: []string{}},
			},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name:           "returns_empty_list",
			mockVMs:        []multipass.ListInstance{},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name:           "returns_error",
			mockVMs:        nil,
			mockErr:        errors.New("multipass error"),
			expectedStatus: http.StatusInternalServerError,
			expectedLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP.ExpectedCalls = nil
			mockMP.On("List").Return(tt.mockVMs, tt.mockErr)

			req := httptest.NewRequest(http.MethodGet, "/api/vms", nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var vms []multipass.ListInstance
				err := json.NewDecoder(rec.Body).Decode(&vms)
				require.NoError(t, err)
				assert.Len(t, vms, tt.expectedLen)
			}
			mockMP.AssertExpectations(t)
		})
	}
}

func TestVMHandler_Get(t *testing.T) {
	handler, mockMP := setupVMHandler(t)

	tests := []struct {
		name           string
		vmName         string
		mockInfo       *multipass.InstanceInfo
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "returns_vm_info",
			vmName:         "test-vm",
			mockInfo:       testutil.RunningVM("test-vm", "192.168.1.100"),
			mockErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "returns_not_found",
			vmName:         "nonexistent",
			mockInfo:       nil,
			mockErr:        errors.New("VM not found"),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP.ExpectedCalls = nil
			mockMP.On("Info", tt.vmName).Return(tt.mockInfo, tt.mockErr)

			req := httptest.NewRequest(http.MethodGet, "/api/vms/"+tt.vmName, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.vmName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Get(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestVMHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		request        CreateVMRequest
		mockSetup      func(*testutil.MockMultipassClient)
		expectedStatus int
	}{
		{
			name:    "successful_create",
			request: CreateVMRequest{Name: "new-vm"},
			mockSetup: func(m *testutil.MockMultipassClient) {
				m.On("Launch", mock.MatchedBy(func(opts multipass.LaunchOptions) bool {
					return opts.Name == "new-vm" && opts.CPUs == 2 && opts.Memory == "4G" && opts.Disk == "20G"
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "with_custom_specs",
			request: CreateVMRequest{
				Name:   "custom-vm",
				CPUs:   4,
				Memory: "8G",
				Disk:   "50G",
			},
			mockSetup: func(m *testutil.MockMultipassClient) {
				m.On("Launch", mock.MatchedBy(func(opts multipass.LaunchOptions) bool {
					return opts.Name == "custom-vm" && opts.CPUs == 4 && opts.Memory == "8G" && opts.Disk == "50G"
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing_name",
			request:        CreateVMRequest{CPUs: 4},
			mockSetup:      func(m *testutil.MockMultipassClient) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "launch_error",
			request: CreateVMRequest{Name: "error-vm"},
			mockSetup: func(m *testutil.MockMultipassClient) {
				m.On("Launch", mock.Anything).Return(errors.New("launch failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "invalid_network_config",
			request: CreateVMRequest{
				Name: "net-vm",
				Network: &multipass.NetworkConfig{
					Mode:  multipass.NetworkModeAllowlist,
					Rules: []multipass.NetworkRule{}, // Allowlist requires rules
				},
			},
			mockSetup:      func(m *testutil.MockMultipassClient) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP := new(testutil.MockMultipassClient)
			cfg := config.DefaultConfig()
			handler := NewVMHandler(mockMP, cfg)
			tt.mockSetup(mockMP)

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestVMHandler_Create_InvalidJSON(t *testing.T) {
	handler, _ := setupVMHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVMHandler_Delete(t *testing.T) {
	handler, mockMP := setupVMHandler(t)

	tests := []struct {
		name           string
		vmName         string
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "successful_delete",
			vmName:         "to-delete",
			mockErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "delete_error",
			vmName:         "error-vm",
			mockErr:        errors.New("delete failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP.ExpectedCalls = nil
			mockMP.On("Delete", tt.vmName, true).Return(tt.mockErr)

			req := httptest.NewRequest(http.MethodDelete, "/api/vms/"+tt.vmName, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.vmName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Delete(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestVMHandler_ChangeState(t *testing.T) {
	tests := []struct {
		name           string
		vmName         string
		action         string
		mockMethod     string
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "start_vm",
			vmName:         "test-vm",
			action:         "start",
			mockMethod:     "Start",
			mockErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "stop_vm",
			vmName:         "test-vm",
			action:         "stop",
			mockMethod:     "Stop",
			mockErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "restart_vm",
			vmName:         "test-vm",
			action:         "restart",
			mockMethod:     "Restart",
			mockErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid_action",
			vmName:         "test-vm",
			action:         "invalid",
			mockMethod:     "",
			mockErr:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "start_error",
			vmName:         "test-vm",
			action:         "start",
			mockMethod:     "Start",
			mockErr:        errors.New("start failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP := new(testutil.MockMultipassClient)
			cfg := config.DefaultConfig()
			handler := NewVMHandler(mockMP, cfg)

			if tt.mockMethod != "" {
				switch tt.mockMethod {
				case "Start":
					mockMP.On("Start", tt.vmName).Return(tt.mockErr)
				case "Stop":
					mockMP.On("Stop", tt.vmName).Return(tt.mockErr)
				case "Restart":
					mockMP.On("Restart", tt.vmName).Return(tt.mockErr)
				}
			}

			body, _ := json.Marshal(StateChangeRequest{Action: tt.action})
			req := httptest.NewRequest(http.MethodPost, "/api/vms/"+tt.vmName+"/state", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.vmName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.ChangeState(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestVMHandler_Clone(t *testing.T) {
	tests := []struct {
		name           string
		sourceName     string
		newName        string
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "successful_clone",
			sourceName:     "source-vm",
			newName:        "clone-vm",
			mockErr:        nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing_new_name",
			sourceName:     "source-vm",
			newName:        "",
			mockErr:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "clone_error",
			sourceName:     "source-vm",
			newName:        "clone-vm",
			mockErr:        errors.New("clone failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMP := new(testutil.MockMultipassClient)
			cfg := config.DefaultConfig()
			handler := NewVMHandler(mockMP, cfg)

			if tt.newName != "" {
				mockMP.On("Clone", tt.sourceName, tt.newName).Return(tt.mockErr)
			}

			body, _ := json.Marshal(CloneRequest{NewName: tt.newName})
			req := httptest.NewRequest(http.MethodPost, "/api/vms/"+tt.sourceName+"/clone", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.sourceName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Clone(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestVMHandler_Defaults(t *testing.T) {
	handler, _ := setupVMHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/vms/defaults", nil)
	rec := httptest.NewRecorder()

	handler.Defaults(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var defaults map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&defaults)
	require.NoError(t, err)

	assert.Equal(t, float64(2), defaults["cpu"])
	assert.Equal(t, "4G", defaults["mem"])
	assert.Equal(t, "20G", defaults["disk"])
}

func TestNewVMHandler(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	cfg := config.DefaultConfig()
	handler := NewVMHandler(mockMP, cfg)

	require.NotNil(t, handler)
	assert.Equal(t, mockMP, handler.mp)
	assert.Equal(t, cfg, handler.cfg)
}

func TestRespondJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	respondJSON(rec, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result map[string]string
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestRespondError(t *testing.T) {
	rec := httptest.NewRecorder()
	err := errors.New("test error")

	respondError(rec, http.StatusBadRequest, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result map[string]string
	decodeErr := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, decodeErr)
	assert.Equal(t, "test error", result["error"])
}
