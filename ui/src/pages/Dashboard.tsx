import { useState, useEffect, useMemo, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { api, VM } from "../api/client";
import CreateVMModal from "../components/CreateVMModal";
import ConfirmModal from "../components/ConfirmModal";

// Icons
const SearchIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="11" cy="11" r="8" />
    <line x1="21" y1="21" x2="16.65" y2="16.65" />
  </svg>
);

const PlusIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="12" y1="5" x2="12" y2="19" />
    <line x1="5" y1="12" x2="19" y2="12" />
  </svg>
);

const PlayIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polygon points="5,3 19,12 5,21" fill="currentColor" stroke="none" />
  </svg>
);

const StopIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="6" y="6" width="12" height="12" rx="1" fill="currentColor" stroke="none" />
  </svg>
);

const TerminalIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="4,17 10,11 4,5" />
    <line x1="12" y1="19" x2="20" y2="19" />
  </svg>
);

const FolderIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M22,19a2,2,0,0,1-2,2H4a2,2,0,0,1-2-2V5A2,2,0,0,1,4,3H9l2,3h9a2,2,0,0,1,2,2Z" />
  </svg>
);

const AgentIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M15 4V2M15 16v-2M8 9h2M20 9h2M17.8 11.8L19 13M17.8 6.2L19 5M3 21l9-9M12.2 6.2L11 5" />
    <circle cx="15" cy="9" r="3" />
  </svg>
);

const ChevronRightIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="9,18 15,12 9,6" />
  </svg>
);

interface ConfirmState {
  type: "stop" | null;
  vmName: string;
}

export default function Dashboard() {
  const navigate = useNavigate();
  const [vms, setVMs] = useState<VM[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [pendingVMName, setPendingVMName] = useState<string | null>(null);
  const [confirmState, setConfirmState] = useState<ConfirmState>({ type: null, vmName: "" });
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const loadVMs = useCallback(async () => {
    try {
      const data = await api.listVMs();
      setVMs(data || []);
      setError("");
    } catch {
      // Don't show error during transitions - VM might not be ready yet
      setError("Failed to load VMs");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadVMs();
    const interval = setInterval(loadVMs, 5000);
    return () => clearInterval(interval);
  }, [loadVMs]);

  // Poll faster when waiting for a new VM to appear, and close modal when it does
  useEffect(() => {
    if (!pendingVMName) return;

    const pollForVM = async () => {
      try {
        const data = await api.listVMs();
        setVMs(data || []);
        const found = (data || []).some((vm) => vm.name === pendingVMName);
        if (found) {
          setPendingVMName(null);
          setShowCreateModal(false);
        }
      } catch {
        // Continue polling
      }
    };

    // Poll immediately and then every 500ms
    pollForVM();
    const interval = setInterval(pollForVM, 500);
    return () => clearInterval(interval);
  }, [pendingVMName]);

  const filteredVMs = useMemo(() => {
    if (!searchQuery.trim()) return vms;
    const query = searchQuery.toLowerCase();
    return vms.filter(
      (vm) =>
        vm.name.toLowerCase().includes(query) ||
        vm.state.toLowerCase().includes(query)
    );
  }, [vms, searchQuery]);

  const handleVMCreated = (vmName: string) => {
    // Start polling for the VM to appear - modal will close automatically
    setPendingVMName(vmName);
  };

  const handleAction = async (name: string, action: string, e: React.MouseEvent) => {
    e.stopPropagation();

    if (action === "stop") {
      setConfirmState({ type: "stop", vmName: name });
      return;
    }

    setActionLoading(name);
    try {
      if (action === "start") {
        await api.startVM(name);
      }
      loadVMs();
    } catch (err) {
      alert(`Failed to ${action} VM: ${err}`);
    } finally {
      setActionLoading(null);
    }
  };

  const handleConfirm = async () => {
    const { type, vmName } = confirmState;
    if (!type || !vmName) return;

    setActionLoading(vmName);
    setConfirmState({ type: null, vmName: "" });

    try {
      await api.stopVM(vmName);
      loadVMs();
    } catch (err) {
      alert(`Failed to stop VM: ${err}`);
    } finally {
      setActionLoading(null);
    }
  };

  const openTerminal = (name: string, e: React.MouseEvent) => {
    e.stopPropagation();
    window.open(`/vm/${name}/terminal`, "_blank");
  };

  const openFiles = (name: string, e: React.MouseEvent) => {
    e.stopPropagation();
    window.open(`/vm/${name}/files`, "_blank");
  };

  const openAgent = async (name: string, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      const { url } = await api.getAgentURL(name);
      window.open(url, '_blank');
    } catch (err) {
      alert(`Failed to get agent URL: ${err}`);
    }
  };

  const handleRowClick = (name: string) => {
    navigate(`/vm/${name}`);
  };

  const isTransitioning = (state: string) =>
    ["Starting", "Restarting", "Unknown", "Suspending"].includes(state);

  const getStatusColor = (state: string) => {
    if (state === "Running") return "var(--success)";
    if (isTransitioning(state)) return "var(--warning)";
    if (state === "Deleted" || state === "Error") return "var(--error)";
    return "var(--text-tertiary)";
  };

  const getTransitionLabel = (state: string) => {
    if (state === "Unknown") return "Starting";
    return state;
  };

  return (
    <div className="dashboard">
      {/* Header */}
      <div className="dashboard-header">
        <div className="header-title">
          <h1>machines</h1>
          <button
            className="btn-action btn-action--primary"
            onClick={() => setShowCreateModal(true)}
          >
            <PlusIcon />
            <span>New</span>
          </button>
        </div>
        <div className="search-wrapper">
          <SearchIcon />
          <input
            type="text"
            className="search-input"
            placeholder="Search..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button className="search-clear" onClick={() => setSearchQuery("")}>
              &times;
            </button>
          )}
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="error-banner">
          {error}
          <button onClick={loadVMs}>Retry</button>
        </div>
      )}

      {/* VM List */}
      <div className="vm-list">
        {loading ? (
          <div className="empty-state">Loading...</div>
        ) : filteredVMs.length === 0 ? (
          <div className="empty-state">
            {searchQuery ? (
              <>No machines match "{searchQuery}"</>
            ) : (
              <>No machines yet. Create one to get started.</>
            )}
          </div>
        ) : (
          filteredVMs.map((vm) => {
            const isLoading = actionLoading === vm.name;
            const transitioning = isTransitioning(vm.state);
            const isRunning = vm.state === "Running";
            const isStopped = vm.state === "Stopped" || vm.state === "Suspended";

            return (
              <div
                key={vm.name}
                className={`vm-card ${isLoading ? "loading" : ""}`}
                onClick={() => handleRowClick(vm.name)}
              >
                <div className="vm-info">
                  <span
                    className={`status-dot ${transitioning ? "pulsing" : ""}`}
                    style={{ background: getStatusColor(vm.state) }}
                  />
                  <span className="vm-name">{vm.name}</span>
                  {vm.ipv4?.[0] && (
                    <span className="vm-ip">{vm.ipv4[0]}</span>
                  )}
                </div>

                <div className="vm-actions" onClick={(e) => e.stopPropagation()}>
                  {transitioning ? (
                    <span className="inline-state">
                      <span className="spinner-sm" />
                      <span>{getTransitionLabel(vm.state)}</span>
                    </span>
                  ) : isRunning ? (
                    <>
                      <button
                        className="btn-action btn-action--icon"
                        onClick={(e) => openAgent(vm.name, e)}
                        title="Agent"
                      >
                        <AgentIcon />
                      </button>
                      <button
                        className="btn-action btn-action--icon"
                        onClick={(e) => openFiles(vm.name, e)}
                        title="Files"
                      >
                        <FolderIcon />
                      </button>
                      <button
                        className="btn-action btn-action--icon"
                        onClick={(e) => openTerminal(vm.name, e)}
                        title="Terminal"
                      >
                        <TerminalIcon />
                      </button>
                      <button
                        className="btn-action btn-action--icon btn-action--warning"
                        onClick={(e) => handleAction(vm.name, "stop", e)}
                        title="Stop"
                        disabled={isLoading}
                      >
                        <StopIcon />
                      </button>
                    </>
                  ) : isStopped ? (
                    <button
                      className="btn-action btn-action--icon btn-action--success"
                      onClick={(e) => handleAction(vm.name, "start", e)}
                      title="Start"
                      disabled={isLoading}
                    >
                      <PlayIcon />
                    </button>
                  ) : null}
                  <span className="chevron">
                    <ChevronRightIcon />
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Stats */}
      {vms.length > 0 && (
        <div className="stats">
          {vms.length} machine{vms.length !== 1 ? "s" : ""} ·{" "}
          {vms.filter((v) => v.state === "Running").length} running
        </div>
      )}

      {/* Modals */}
      {showCreateModal && (
        <CreateVMModal
          onClose={() => {
            setShowCreateModal(false);
            setPendingVMName(null); // Stop fast polling if user closes modal
          }}
          onCreated={handleVMCreated}
        />
      )}

      {confirmState.type === "stop" && (
        <ConfirmModal
          title="Stop Machine"
          message={`Stop "${confirmState.vmName}"? Unsaved work may be lost.`}
          confirmText="Stop"
          variant="warning"
          onConfirm={handleConfirm}
          onCancel={() => setConfirmState({ type: null, vmName: "" })}
        />
      )}

      <style>{`
        .dashboard {
          max-width: 720px;
          margin: 0 auto;
          padding: var(--space-lg);
        }

        .dashboard-header {
          display: flex;
          flex-direction: column;
          gap: var(--space-md);
          margin-bottom: var(--space-xl);
        }

        .header-title {
          display: flex;
          align-items: center;
          justify-content: space-between;
        }

        .header-title h1 {
          font-size: var(--text-xl);
          font-weight: 600;
        }

        .search-wrapper {
          position: relative;
        }

        .search-wrapper svg {
          position: absolute;
          left: var(--space-md);
          top: 50%;
          transform: translateY(-50%);
          color: var(--text-tertiary);
          pointer-events: none;
        }

        .search-input {
          width: 100%;
          padding: var(--space-sm) var(--space-md);
          padding-left: 36px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          font-size: var(--text-base);
          color: var(--text-primary);
        }

        .search-input::placeholder {
          color: var(--text-tertiary);
        }

        .search-input:focus {
          outline: none;
          border-color: var(--accent);
        }

        .search-clear {
          position: absolute;
          right: var(--space-sm);
          top: 50%;
          transform: translateY(-50%);
          width: 24px;
          height: 24px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: var(--text-tertiary);
          font-size: 18px;
          border-radius: 50%;
        }

        .search-clear:hover {
          background: var(--bg-hover);
          color: var(--text-primary);
        }

        .error-banner {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-md);
          background: var(--error-bg);
          border: 1px solid var(--error);
          border-radius: var(--radius-md);
          color: var(--error);
          margin-bottom: var(--space-lg);
          font-size: var(--text-sm);
        }

        .error-banner button {
          padding: var(--space-xs) var(--space-sm);
          background: var(--error);
          color: white;
          border-radius: var(--radius-sm);
          font-size: var(--text-xs);
        }

        .vm-list {
          display: flex;
          flex-direction: column;
          gap: var(--space-sm);
        }

        .empty-state {
          padding: var(--space-2xl);
          text-align: center;
          color: var(--text-secondary);
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
        }

        .vm-card {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-md) var(--space-lg);
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all 0.15s ease;
          min-height: 56px;
        }

        .vm-card:hover {
          border-color: var(--accent);
          background: var(--bg-hover);
        }

        .vm-card.loading {
          opacity: 0.6;
          pointer-events: none;
        }

        .vm-info {
          display: flex;
          align-items: center;
          gap: var(--space-md);
          min-width: 0;
        }

        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          flex-shrink: 0;
        }

        .status-dot.pulsing {
          animation: pulse 1.5s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }

        .vm-name {
          font-family: var(--font-mono);
          font-weight: 500;
          font-size: var(--text-base);
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .vm-ip {
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .vm-actions {
          display: flex;
          align-items: center;
          gap: var(--space-xs);
          flex-shrink: 0;
        }

        .inline-state {
          display: flex;
          align-items: center;
          gap: var(--space-xs);
          font-size: var(--text-xs);
          color: var(--warning);
          padding: 0 var(--space-sm);
        }

        .spinner-sm {
          width: 12px;
          height: 12px;
          border: 2px solid var(--border);
          border-top-color: var(--warning);
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        .chevron {
          color: var(--text-tertiary);
          margin-left: var(--space-sm);
        }

        /* Action Buttons */
        .btn-action {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          gap: var(--space-xs);
          padding: 8px 14px;
          font-size: var(--text-sm);
          font-weight: 500;
          color: var(--text-primary);
          background: transparent;
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all 0.15s ease;
          white-space: nowrap;
        }

        .btn-action:hover:not(:disabled) {
          border-color: var(--text-tertiary);
          background: var(--bg-hover);
        }

        .btn-action:active:not(:disabled) {
          transform: scale(0.97);
        }

        .btn-action:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .btn-action--icon {
          width: 36px;
          height: 36px;
          padding: 0;
        }

        .btn-action--primary {
          border-color: var(--accent);
          color: var(--accent);
        }

        .btn-action--primary:hover:not(:disabled) {
          background: var(--accent);
          border-color: var(--accent);
          color: white;
        }

        .btn-action--success {
          border-color: var(--success);
          color: var(--success);
        }

        .btn-action--success:hover:not(:disabled) {
          background: var(--success);
          border-color: var(--success);
          color: white;
        }

        .btn-action--warning {
          border-color: var(--warning);
          color: var(--warning);
        }

        .btn-action--warning:hover:not(:disabled) {
          background: var(--warning);
          border-color: var(--warning);
          color: white;
        }

        .stats {
          margin-top: var(--space-xl);
          text-align: center;
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        @media (max-width: 480px) {
          .dashboard {
            padding: var(--space-md);
          }

          .vm-card {
            padding: var(--space-sm) var(--space-md);
          }

          .vm-ip {
            display: none;
          }

          .btn-action--icon {
            width: 32px;
            height: 32px;
          }

          .header-title h1 {
            font-size: var(--text-lg);
          }
        }
      `}</style>
    </div>
  );
}
