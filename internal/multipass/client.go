package multipass

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// CommandExecutor interface for testability
type CommandExecutor interface {
	Execute(name string, args ...string) ([]byte, error)
}

// RealExecutor uses actual exec.Command
type RealExecutor struct{}

// Execute runs a command and returns stdout
func (e RealExecutor) Execute(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, &MultipassError{
			Command: strings.Join(append([]string{name}, args...), " "),
			Stderr:  stderr.String(),
			Err:     err,
		}
	}
	return stdout.Bytes(), nil
}

// MultipassError wraps exec errors with context
type MultipassError struct {
	Command string
	Stderr  string
	Err     error
}

func (e *MultipassError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("multipass command failed: %s\nstderr: %s", e.Command, strings.TrimSpace(e.Stderr))
	}
	return fmt.Sprintf("multipass command failed: %s: %v", e.Command, e.Err)
}

func (e *MultipassError) Unwrap() error {
	return e.Err
}

// Client interface for multipass operations
type Client interface {
	// VM Lifecycle
	List() ([]ListInstance, error)
	Info(name string) (*InstanceInfo, error)
	Launch(opts LaunchOptions) error
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	Delete(name string, purge bool) error

	// Clone
	Clone(source, dest string) error

	// Snapshots
	ListSnapshots(vmName string) (map[string]Snapshot, error)
	CreateSnapshot(vmName, snapshotName string) error
	RestoreSnapshot(vmName, snapshotName string, destructive bool) error
	DeleteSnapshot(vmName, snapshotName string) error

	// Files
	Transfer(src, dst string) error
	Exec(vmName string, cmd ...string) (string, error)

	// Mounts
	Mount(vmName, hostPath, vmPath string) error
	Unmount(vmName, path string) error
}

// client implements Client using multipass CLI
type client struct {
	exec CommandExecutor
}

// NewClient creates a new multipass client with the given executor
func NewClient(exec CommandExecutor) Client {
	return &client{exec: exec}
}

// NewRealClient creates a client that executes real multipass commands
func NewRealClient() Client {
	return &client{exec: RealExecutor{}}
}

// List returns all VMs
func (c *client) List() ([]ListInstance, error) {
	out, err := c.exec.Execute("multipass", "list", "--format", "json")
	if err != nil {
		return nil, err
	}

	var resp ListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse list output: %w", err)
	}
	return resp.List, nil
}

// Info returns detailed information about a VM
func (c *client) Info(name string) (*InstanceInfo, error) {
	out, err := c.exec.Execute("multipass", "info", name, "--format", "json")
	if err != nil {
		return nil, err
	}

	var resp InfoResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse info output: %w", err)
	}

	info, ok := resp.Info[name]
	if !ok {
		return nil, fmt.Errorf("vm not found: %s", name)
	}
	return &info, nil
}

// Launch creates and starts a new VM
func (c *client) Launch(opts LaunchOptions) error {
	args := []string{"launch", "--name", opts.Name}

	if opts.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%d", opts.CPUs))
	}
	if opts.Memory != "" {
		args = append(args, "--memory", opts.Memory)
	}
	if opts.Disk != "" {
		args = append(args, "--disk", opts.Disk)
	}
	if opts.CloudInit != "" {
		args = append(args, "--cloud-init", opts.CloudInit)
	}
	if opts.Image != "" {
		args = append(args, opts.Image)
	}

	_, err := c.exec.Execute("multipass", args...)
	return err
}

// Start starts a stopped VM
func (c *client) Start(name string) error {
	_, err := c.exec.Execute("multipass", "start", name)
	return err
}

// Stop stops a running VM
func (c *client) Stop(name string) error {
	_, err := c.exec.Execute("multipass", "stop", name)
	return err
}

// Restart restarts a VM
func (c *client) Restart(name string) error {
	_, err := c.exec.Execute("multipass", "restart", name)
	return err
}

// Delete removes a VM
func (c *client) Delete(name string, purge bool) error {
	args := []string{"delete", name}
	if purge {
		args = append(args, "--purge")
	}
	_, err := c.exec.Execute("multipass", args...)
	return err
}

// Clone creates a copy of a VM
func (c *client) Clone(source, dest string) error {
	_, err := c.exec.Execute("multipass", "clone", source, "-n", dest)
	return err
}

// ListSnapshots returns all snapshots for a VM
func (c *client) ListSnapshots(vmName string) (map[string]Snapshot, error) {
	out, err := c.exec.Execute("multipass", "list", "--snapshots", "--format", "json")
	if err != nil {
		return nil, err
	}

	var resp SnapshotsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse snapshots output: %w", err)
	}

	snapshots, ok := resp.Info[vmName]
	if !ok {
		return make(map[string]Snapshot), nil
	}
	return snapshots, nil
}

// CreateSnapshot creates a new snapshot (VM must be stopped)
func (c *client) CreateSnapshot(vmName, snapshotName string) error {
	args := []string{"snapshot", vmName}
	if snapshotName != "" {
		args = append(args, "--name", snapshotName)
	}
	_, err := c.exec.Execute("multipass", args...)
	return err
}

// RestoreSnapshot restores a VM to a previous snapshot
func (c *client) RestoreSnapshot(vmName, snapshotName string, destructive bool) error {
	args := []string{"restore", fmt.Sprintf("%s.%s", vmName, snapshotName)}
	if destructive {
		args = append(args, "--destructive")
	}
	_, err := c.exec.Execute("multipass", args...)
	return err
}

// DeleteSnapshot removes a snapshot
func (c *client) DeleteSnapshot(vmName, snapshotName string) error {
	_, err := c.exec.Execute("multipass", "delete", "--purge", fmt.Sprintf("%s.%s", vmName, snapshotName))
	return err
}

// Transfer copies files between host and VM
// Use vm_name:path syntax for VM paths
func (c *client) Transfer(src, dst string) error {
	_, err := c.exec.Execute("multipass", "transfer", src, dst)
	return err
}

// Exec runs a command in a VM and returns the output
func (c *client) Exec(vmName string, cmd ...string) (string, error) {
	args := append([]string{"exec", vmName, "--"}, cmd...)
	out, err := c.exec.Execute("multipass", args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Mount mounts a host directory to a VM
func (c *client) Mount(vmName, hostPath, vmPath string) error {
	target := fmt.Sprintf("%s:%s", vmName, vmPath)
	_, err := c.exec.Execute("multipass", "mount", hostPath, target)
	return err
}

// Unmount removes a mount from a VM
func (c *client) Unmount(vmName, path string) error {
	target := fmt.Sprintf("%s:%s", vmName, path)
	_, err := c.exec.Execute("multipass", "umount", target)
	return err
}
