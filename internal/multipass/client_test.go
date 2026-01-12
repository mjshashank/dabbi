package multipass

import (
	"errors"
	"testing"
)

// MockExecutor for testing
type MockExecutor struct {
	responses map[string][]byte
	errors    map[string]error
	calls     []string
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		responses: make(map[string][]byte),
		errors:    make(map[string]error),
		calls:     make([]string, 0),
	}
}

// SetResponse sets a mock response for a command
func (m *MockExecutor) SetResponse(key string, response []byte) {
	m.responses[key] = response
}

// SetError sets a mock error for a command
func (m *MockExecutor) SetError(key string, err error) {
	m.errors[key] = err
}

// Execute mocks command execution
func (m *MockExecutor) Execute(name string, args ...string) ([]byte, error) {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	m.calls = append(m.calls, key)

	if err, ok := m.errors[key]; ok {
		return nil, err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return nil, errors.New("unexpected command: " + key)
}

// GetCalls returns all commands that were executed
func (m *MockExecutor) GetCalls() []string {
	return m.calls
}

func TestClient_List(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass list --format json", []byte(`{
		"list": [
			{
				"ipv4": ["192.168.2.3"],
				"name": "test-vm",
				"release": "Ubuntu 24.04 LTS",
				"state": "Running"
			},
			{
				"ipv4": [],
				"name": "stopped-vm",
				"release": "Ubuntu 22.04 LTS",
				"state": "Stopped"
			}
		]
	}`))

	client := NewClient(mock)
	vms, err := client.List()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("expected 2 VMs, got %d", len(vms))
	}
	if vms[0].Name != "test-vm" {
		t.Errorf("expected name 'test-vm', got '%s'", vms[0].Name)
	}
	if vms[0].State != "Running" {
		t.Errorf("expected state 'Running', got '%s'", vms[0].State)
	}
	if len(vms[0].IPv4) != 1 || vms[0].IPv4[0] != "192.168.2.3" {
		t.Errorf("expected IPv4 '192.168.2.3', got %v", vms[0].IPv4)
	}
}

func TestClient_Info(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass info test-vm --format json", []byte(`{
		"errors": [],
		"info": {
			"test-vm": {
				"cpu_count": "2",
				"disks": {
					"sda1": {"total": "4081515520", "used": "2184845824"}
				},
				"image_hash": "abc123",
				"image_release": "24.04 LTS",
				"ipv4": ["192.168.2.3"],
				"load": [0.1, 0.05, 0.01],
				"memory": {"total": 472784896, "used": 180793344},
				"mounts": {},
				"release": "Ubuntu 24.04.3 LTS",
				"snapshot_count": "2",
				"state": "Running"
			}
		}
	}`))

	client := NewClient(mock)
	info, err := client.Info("test-vm")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CPUCount != "2" {
		t.Errorf("expected cpu_count '2', got '%s'", info.CPUCount)
	}
	if info.SnapshotCount != "2" {
		t.Errorf("expected snapshot_count '2', got '%s'", info.SnapshotCount)
	}
	if info.Memory.Total != 472784896 {
		t.Errorf("expected memory total 472784896, got %d", info.Memory.Total)
	}
	if info.State != "Running" {
		t.Errorf("expected state 'Running', got '%s'", info.State)
	}
	if len(info.Load) != 3 {
		t.Errorf("expected 3 load values, got %d", len(info.Load))
	}
}

func TestClient_Launch(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass launch --name test-vm --cpus 2 --memory 4G --disk 20G", []byte(""))

	client := NewClient(mock)
	err := client.Launch(LaunchOptions{
		Name:   "test-vm",
		CPUs:   2,
		Memory: "4G",
		Disk:   "20G",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
}

func TestClient_LaunchWithCloudInit(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass launch --name test-vm --cpus 2 --memory 4G --disk 20G --cloud-init /tmp/init.yaml jammy", []byte(""))

	client := NewClient(mock)
	err := client.Launch(LaunchOptions{
		Name:      "test-vm",
		CPUs:      2,
		Memory:    "4G",
		Disk:      "20G",
		CloudInit: "/tmp/init.yaml",
		Image:     "jammy",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_StartStopRestart(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass start test-vm", []byte(""))
	mock.SetResponse("multipass stop test-vm", []byte(""))
	mock.SetResponse("multipass restart test-vm", []byte(""))

	client := NewClient(mock)

	if err := client.Start("test-vm"); err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if err := client.Stop("test-vm"); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	if err := client.Restart("test-vm"); err != nil {
		t.Errorf("Restart failed: %v", err)
	}
}

func TestClient_Delete(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass delete test-vm", []byte(""))
	mock.SetResponse("multipass delete test-vm --purge", []byte(""))

	client := NewClient(mock)

	if err := client.Delete("test-vm", false); err != nil {
		t.Errorf("Delete without purge failed: %v", err)
	}
	if err := client.Delete("test-vm", true); err != nil {
		t.Errorf("Delete with purge failed: %v", err)
	}
}

func TestClient_Clone(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass clone source-vm -n dest-vm", []byte(""))

	client := NewClient(mock)
	err := client.Clone("source-vm", "dest-vm")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_ListSnapshots(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass list --snapshots --format json", []byte(`{
		"errors": [],
		"info": {
			"test-vm": {
				"snap1": {
					"comment": "First snapshot",
					"parent": ""
				},
				"snap2": {
					"comment": "Second snapshot",
					"parent": "snap1"
				}
			}
		}
	}`))

	client := NewClient(mock)
	snapshots, err := client.ListSnapshots("test-vm")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots["snap1"].Comment != "First snapshot" {
		t.Errorf("expected comment 'First snapshot', got '%s'", snapshots["snap1"].Comment)
	}
	if snapshots["snap2"].Parent != "snap1" {
		t.Errorf("expected parent 'snap1', got '%s'", snapshots["snap2"].Parent)
	}
}

func TestClient_ListSnapshots_NoSnapshots(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass list --snapshots --format json", []byte(`{
		"errors": [],
		"info": {}
	}`))

	client := NewClient(mock)
	snapshots, err := client.ListSnapshots("nonexistent-vm")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestClient_SnapshotOperations(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass snapshot test-vm --name snap1", []byte(""))
	mock.SetResponse("multipass restore test-vm.snap1", []byte(""))
	mock.SetResponse("multipass restore test-vm.snap1 --destructive", []byte(""))
	mock.SetResponse("multipass delete --purge test-vm.snap1", []byte(""))

	client := NewClient(mock)

	if err := client.CreateSnapshot("test-vm", "snap1"); err != nil {
		t.Errorf("CreateSnapshot failed: %v", err)
	}
	if err := client.RestoreSnapshot("test-vm", "snap1", false); err != nil {
		t.Errorf("RestoreSnapshot failed: %v", err)
	}
	if err := client.RestoreSnapshot("test-vm", "snap1", true); err != nil {
		t.Errorf("RestoreSnapshot with destructive failed: %v", err)
	}
	if err := client.DeleteSnapshot("test-vm", "snap1"); err != nil {
		t.Errorf("DeleteSnapshot failed: %v", err)
	}
}

func TestClient_Transfer(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass transfer ./local.txt test-vm:/home/ubuntu/remote.txt", []byte(""))

	client := NewClient(mock)
	err := client.Transfer("./local.txt", "test-vm:/home/ubuntu/remote.txt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Exec(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass exec test-vm -- ls -la", []byte("total 0\ndrwxr-xr-x  2 ubuntu ubuntu 40 Jan 10 12:00 .\n"))

	client := NewClient(mock)
	output, err := client.Exec("test-vm", "ls", "-la")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestClient_Mount(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass mount /tmp/shared test-vm:/home/ubuntu/shared", []byte(""))

	client := NewClient(mock)
	err := client.Mount("test-vm", "/tmp/shared", "/home/ubuntu/shared")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Unmount(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("multipass umount test-vm:/home/ubuntu/shared", []byte(""))

	client := NewClient(mock)
	err := client.Unmount("test-vm", "/home/ubuntu/shared")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Error(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetError("multipass list --format json", &MultipassError{
		Command: "multipass list --format json",
		Stderr:  "multipass not found",
		Err:     errors.New("exit status 1"),
	})

	client := NewClient(mock)
	_, err := client.List()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var mpErr *MultipassError
	if !errors.As(err, &mpErr) {
		t.Errorf("expected MultipassError, got %T", err)
	}
}
