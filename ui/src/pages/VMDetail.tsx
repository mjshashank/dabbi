import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { api, VMInfo } from '../api/client'
import SnapshotPanel from '../components/SnapshotPanel'
import TunnelsPanel from '../components/TunnelsPanel'
import MountsPanel from '../components/MountsPanel'
import NetworkPanel from '../components/NetworkPanel'
import CloneVMModal from '../components/CloneVMModal'
import ConfirmModal from '../components/ConfirmModal'
import Tooltip from '../components/Tooltip'

type Tab = 'snapshots' | 'tunnels' | 'mounts' | 'network'

const validTabs: Tab[] = ['snapshots', 'tunnels', 'mounts', 'network']

// Icons
const TerminalIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="4 17 10 11 4 5" />
    <line x1="12" y1="19" x2="20" y2="19" />
  </svg>
)

const FolderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
  </svg>
)

const CopyIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </svg>
)

const PlayIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
    <polygon points="5 3 19 12 5 21 5 3" />
  </svg>
)

const StopIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
    <rect x="6" y="6" width="12" height="12" rx="1" />
  </svg>
)

const RestartIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="23 4 23 10 17 10" />
    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
  </svg>
)

const TrashIcon = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="3 6 5 6 21 6" />
    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
  </svg>
)

const ExternalLinkIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
    <polyline points="15 3 21 3 21 9" />
    <line x1="10" y1="14" x2="21" y2="3" />
  </svg>
)

const AgentIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M15 4V2M15 16v-2M8 9h2M20 9h2M17.8 11.8L19 13M17.8 6.2L19 5M3 21l9-9M12.2 6.2L11 5" />
    <circle cx="15" cy="9" r="3" />
  </svg>
)

const CpuIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="4" y="4" width="16" height="16" rx="2" />
    <rect x="9" y="9" width="6" height="6" />
    <line x1="9" y1="1" x2="9" y2="4" />
    <line x1="15" y1="1" x2="15" y2="4" />
    <line x1="9" y1="20" x2="9" y2="23" />
    <line x1="15" y1="20" x2="15" y2="23" />
    <line x1="20" y1="9" x2="23" y2="9" />
    <line x1="20" y1="14" x2="23" y2="14" />
    <line x1="1" y1="9" x2="4" y2="9" />
    <line x1="1" y1="14" x2="4" y2="14" />
  </svg>
)

const MemoryIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="2" y="6" width="20" height="12" rx="2" />
    <line x1="6" y1="6" x2="6" y2="2" />
    <line x1="10" y1="6" x2="10" y2="2" />
    <line x1="14" y1="6" x2="14" y2="2" />
    <line x1="18" y1="6" x2="18" y2="2" />
    <rect x="5" y="9" width="4" height="6" rx="0.5" />
    <rect x="11" y="9" width="4" height="6" rx="0.5" />
  </svg>
)

const DiskIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <ellipse cx="12" cy="5" rx="9" ry="3" />
    <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
    <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
  </svg>
)

const InfoIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <circle cx="12" cy="12" r="10" />
    <line x1="12" y1="16" x2="12" y2="12" />
    <line x1="12" y1="8" x2="12.01" y2="8" />
  </svg>
)

const SpinnerIcon = ({ size = 16 }: { size?: number }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="spinner-icon">
    <circle cx="12" cy="12" r="10" strokeOpacity="0.25" />
    <path d="M12 2a10 10 0 0 1 10 10" strokeLinecap="round" />
  </svg>
)

export default function VMDetail() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [vm, setVM] = useState<VMInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [cloneModalOpen, setCloneModalOpen] = useState(false)
  const [confirmDialog, setConfirmDialog] = useState<{
    type: 'stop' | 'restart' | 'delete' | null
  }>({ type: null })

  // Counts for tabs
  const [tunnelCount, setTunnelCount] = useState(0)
  const [mountCount, setMountCount] = useState(0)

  // Get initial tab from URL query param
  const urlTab = searchParams.get('tab') as Tab | null
  const initialTab = urlTab && validTabs.includes(urlTab) ? urlTab : 'snapshots'
  const [activeTab, setActiveTab] = useState<Tab>(initialTab)

  const loadVM = useCallback(async () => {
    if (!name) return
    try {
      const data = await api.getVM(name)
      setVM(data)
      setError('')

      // Get mount count from VM info
      setMountCount(Object.keys(data.mounts || {}).length)
    } catch {
      // Only show error if we don't already have VM data
      setVM(prev => {
        if (!prev) setError('Failed to load VM details')
        return prev
      })
    } finally {
      setLoading(false)
    }
  }, [name])

  const loadTunnelCount = useCallback(async () => {
    if (!name) return
    try {
      const tunnels = await api.listTunnels()
      const vmTunnels = (tunnels || []).filter(t => t.vm_name === name)
      setTunnelCount(vmTunnels.length)
    } catch {
      // Ignore tunnel count errors
    }
  }, [name])

  useEffect(() => {
    loadVM()
    loadTunnelCount()
    const interval = setInterval(() => {
      loadVM()
      loadTunnelCount()
    }, 5000)
    return () => clearInterval(interval)
  }, [loadVM, loadTunnelCount])

  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab)
    setSearchParams({ tab })
  }

  // Poll until VM state changes after an action
  const pollUntilStateChange = async (expectedAction: string) => {
    if (!name) return
    const targetState = expectedAction === 'start' ? 'Running' : 'Stopped'
    for (let i = 0; i < 30; i++) {
      await new Promise(r => setTimeout(r, 1000))
      try {
        const data = await api.getVM(name)
        setVM(data)
        if (data.state === targetState) break
      } catch {
        // Continue polling
      }
    }
  }

  const handleAction = async (action: string) => {
    if (!name) return
    setActionLoading(action)
    try {
      switch (action) {
        case 'start':
          await api.startVM(name)
          await pollUntilStateChange('start')
          break
        case 'stop':
          await api.stopVM(name)
          await pollUntilStateChange('stop')
          break
        case 'restart':
          await api.restartVM(name)
          // Poll for restart - first goes to stopped, then running
          await pollUntilStateChange('stop')
          await pollUntilStateChange('start')
          break
        case 'delete':
          await api.deleteVM(name)
          navigate('/')
          return
      }
    } catch (err) {
      setError(`Failed to ${action} VM: ${err}`)
    } finally {
      setActionLoading(null)
    }
  }

  const handleConfirmedAction = (action: 'stop' | 'restart' | 'delete') => {
    setConfirmDialog({ type: null })
    handleAction(action)
  }

  const openTerminal = () => {
    window.open(`/vm/${name}/terminal`, '_blank')
  }

  const openFiles = () => {
    window.open(`/vm/${name}/files`, '_blank')
  }

  const openAgent = async () => {
    try {
      const { url } = await api.getAgentURL(name!)
      window.open(url, '_blank')
    } catch (err) {
      setError(`Failed to get agent URL: ${err}`)
    }
  }

  if (loading) {
    return (
      <div className="loading-state container">
        <SpinnerIcon size={24} />
        <span>Loading VM details...</span>
        <style>{`
          .loading-state {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 12px;
            padding: 80px 20px;
            color: var(--text-secondary);
          }
          .spinner-icon {
            animation: spin 0.8s linear infinite;
          }
          @keyframes spin {
            to { transform: rotate(360deg); }
          }
        `}</style>
      </div>
    )
  }

  if (error && !vm) {
    return (
      <div className="error-state container">
        <p>{error}</p>
        <button className="btn btn-primary" onClick={() => navigate('/')}>
          Back to Dashboard
        </button>
        <style>{`
          .error-state {
            text-align: center;
            padding: 80px 20px;
            color: var(--text-secondary);
          }
          .error-state button {
            margin-top: 20px;
          }
        `}</style>
      </div>
    )
  }

  if (!vm) return null

  const formatBytes = (bytes: number) => {
    const units = ['B', 'KB', 'MB', 'GB']
    let i = 0
    let value = bytes
    while (value >= 1024 && i < units.length - 1) {
      value /= 1024
      i++
    }
    return `${value.toFixed(1)} ${units[i]}`
  }

  const memoryPercent = vm.memory.total > 0
    ? Math.round((vm.memory.used / vm.memory.total) * 100)
    : 0

  const diskEntries = Object.values(vm.disks || {})
  const diskInfo = diskEntries[0] || { total: '0', used: '0' }
  const diskTotalFromAPI = parseFloat(diskInfo.total) || 0
  const diskUsedBytes = parseFloat(diskInfo.used) || 0

  // Cache disk total in localStorage when available (VM running)
  // Show cached value when VM is stopped and API returns 0
  const diskCacheKey = `dabbi_disk_total_${name}`
  let diskTotalBytes = diskTotalFromAPI
  if (diskTotalFromAPI > 0) {
    localStorage.setItem(diskCacheKey, String(diskTotalFromAPI))
  } else {
    const cached = localStorage.getItem(diskCacheKey)
    if (cached) {
      diskTotalBytes = parseFloat(cached) || 0
    }
  }

  const diskPercent = diskTotalBytes > 0
    ? Math.round((diskUsedBytes / diskTotalBytes) * 100)
    : 0

  const isRunning = vm.state === 'Running'
  const snapshotCount = parseInt(vm.snapshot_count) || 0

  // State helpers - match Dashboard logic
  const isTransitioningState = (state: string) =>
    ['Starting', 'Restarting', 'Unknown', 'Suspending'].includes(state)

  const getStateLabel = (state: string) => {
    if (state === 'Unknown') return 'Starting'
    return state
  }

  const vmTransitioning = isTransitioningState(vm.state)
  const isTransitioning = actionLoading !== null || vmTransitioning

  return (
    <div className="vm-detail container">
      {/* Header Row - VM name on left, actions on right */}
      <div className="header-row">
        <div className="vm-name-section">
          <h1>{name}</h1>
          <span className={`state-badge state-${vm.state.toLowerCase()} ${vmTransitioning ? 'transitioning' : ''}`}>
            {actionLoading === 'start' || actionLoading === 'stop' || actionLoading === 'restart' ? (
              <>
                <SpinnerIcon size={12} />
                {actionLoading === 'start' ? 'Starting' : actionLoading === 'stop' ? 'Stopping' : 'Restarting'}
              </>
            ) : vmTransitioning ? (
              <>
                <SpinnerIcon size={12} />
                {getStateLabel(vm.state)}
              </>
            ) : (
              getStateLabel(vm.state)
            )}
          </span>
        </div>

        <div className="header-actions">
          {isRunning ? (
            <>
              <Tooltip text="Wait for current action to complete" show={isTransitioning}>
                <button
                  className="action-btn"
                  onClick={() => setConfirmDialog({ type: 'restart' })}
                  disabled={isTransitioning}
                >
                  {actionLoading === 'restart' ? <SpinnerIcon size={15} /> : <RestartIcon />}
                  Restart
                </button>
              </Tooltip>
              <Tooltip text="Wait for current action to complete" show={isTransitioning}>
                <button
                  className="action-btn warning-btn"
                  onClick={() => setConfirmDialog({ type: 'stop' })}
                  disabled={isTransitioning}
                >
                  {actionLoading === 'stop' ? <SpinnerIcon size={14} /> : <StopIcon />}
                  Stop
                </button>
              </Tooltip>
            </>
          ) : (
            <>
              <button
                className="action-btn clone-btn"
                onClick={() => setCloneModalOpen(true)}
              >
                <CopyIcon />
                Clone
              </button>
              <Tooltip text="Wait for current action to complete" show={isTransitioning}>
                <button
                  className="action-btn success-btn"
                  onClick={() => handleAction('start')}
                  disabled={isTransitioning}
                >
                  {actionLoading === 'start' ? <SpinnerIcon size={14} /> : <PlayIcon />}
                  Start
                </button>
              </Tooltip>
            </>
          )}
          <Tooltip text="Wait for current action to complete" show={isTransitioning}>
            <button
              className="action-btn danger-btn"
              onClick={() => setConfirmDialog({ type: 'delete' })}
              disabled={isTransitioning}
            >
              {actionLoading === 'delete' ? <SpinnerIcon size={15} /> : <TrashIcon />}
              Delete
            </button>
          </Tooltip>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="error-banner">
          {error}
          <button onClick={() => setError('')}>&times;</button>
        </div>
      )}

      {/* Quick Access Buttons - Agent, Files, Terminal */}
      <div className="quick-access">
        <Tooltip text="Start the VM to access agent" show={!isRunning}>
          <button
            className="quick-btn agent-btn"
            onClick={openAgent}
            disabled={!isRunning}
          >
            <AgentIcon />
            <span className="quick-btn-label">Agent</span>
            <ExternalLinkIcon />
          </button>
        </Tooltip>
        <Tooltip text="Start the VM to browse files" show={!isRunning}>
          <button
            className="quick-btn files-btn"
            onClick={openFiles}
            disabled={!isRunning}
          >
            <FolderIcon />
            <span className="quick-btn-label">Files</span>
            <ExternalLinkIcon />
          </button>
        </Tooltip>
        <Tooltip text="Start the VM to access terminal" show={!isRunning}>
          <button
            className="quick-btn terminal-btn"
            onClick={openTerminal}
            disabled={!isRunning}
          >
            <TerminalIcon />
            <span className="quick-btn-label">Terminal</span>
            <ExternalLinkIcon />
          </button>
        </Tooltip>
      </div>

      {/* Resource Cards - 4 cards */}
      <div className="resource-cards">
        {/* Info - combined Network + Release */}
        <div className="resource-card info-card">
          <div className="resource-header">
            <InfoIcon />
            <span>Info</span>
          </div>
          <div className="info-row">
            <span className="info-label">IP</span>
            <span className="info-value">{vm.ipv4?.[0] || '—'}</span>
          </div>
          <div className="info-row">
            <span className="info-label">Release</span>
            <span className="info-value">{vm.release}</span>
          </div>
        </div>

        {/* CPU - no progress bar */}
        <div className="resource-card cpu-card">
          <div className="resource-header">
            <CpuIcon />
            <span>CPU</span>
          </div>
          <div className="resource-value">{vm.cpu_count}</div>
          <div className="resource-label">{vm.cpu_count === '1' ? 'core' : 'cores'}</div>
        </div>

        {/* Memory - with progress bar */}
        <div className="resource-card">
          <div className="resource-header">
            <MemoryIcon />
            <span>Memory</span>
          </div>
          <div className="resource-value">{formatBytes(vm.memory.used)}</div>
          <div className="resource-bar">
            <div className="resource-fill memory-fill" style={{ width: `${memoryPercent}%` }} />
          </div>
          <div className="resource-label">of {formatBytes(vm.memory.total)} ({memoryPercent}%)</div>
        </div>

        {/* Disk - always show total, usage when available */}
        <div className="resource-card">
          <div className="resource-header">
            <DiskIcon />
            <span>Disk</span>
          </div>
          <div className="resource-value">
            {diskUsedBytes > 0 ? formatBytes(diskUsedBytes) : (diskTotalBytes > 0 ? formatBytes(diskTotalBytes) : '—')}
          </div>
          <div className="resource-bar">
            <div className="resource-fill disk-fill" style={{ width: `${diskPercent}%` }} />
          </div>
          <div className="resource-label">
            {diskUsedBytes > 0
              ? `of ${formatBytes(diskTotalBytes)} (${diskPercent}%)`
              : diskTotalBytes > 0
                ? 'total allocated'
                : 'Start VM to see usage'}
          </div>
        </div>
      </div>

      {/* Tabs with counts */}
      <div className="tabs">
        <button
          className={activeTab === 'snapshots' ? 'active' : ''}
          onClick={() => handleTabChange('snapshots')}
        >
          Snapshots
          <span className="tab-count">{snapshotCount}</span>
        </button>
        <button
          className={activeTab === 'tunnels' ? 'active' : ''}
          onClick={() => handleTabChange('tunnels')}
        >
          Tunnels
          <span className="tab-count">{tunnelCount}</span>
        </button>
        <button
          className={activeTab === 'mounts' ? 'active' : ''}
          onClick={() => handleTabChange('mounts')}
        >
          Mounts
          <span className="tab-count">{mountCount}</span>
        </button>
        <button
          className={activeTab === 'network' ? 'active' : ''}
          onClick={() => handleTabChange('network')}
        >
          Network
        </button>
      </div>

      {/* Tab Content */}
      <div className="tab-content">
        {activeTab === 'snapshots' && name && (
          <SnapshotPanel vmName={name} isRunning={isRunning} onRefresh={loadVM} />
        )}
        {activeTab === 'tunnels' && name && (
          <TunnelsPanel vmName={name} isRunning={isRunning} />
        )}
        {activeTab === 'mounts' && name && (
          <MountsPanel vmName={name} onRefresh={loadVM} />
        )}
        {activeTab === 'network' && name && (
          <NetworkPanel vmName={name} isRunning={isRunning} />
        )}
      </div>

      {/* Clone Modal */}
      {cloneModalOpen && name && (
        <CloneVMModal
          sourceName={name}
          onClose={() => setCloneModalOpen(false)}
          onCloned={() => {
            setCloneModalOpen(false)
            loadVM()
          }}
        />
      )}

      {/* Confirmation Dialogs */}
      {confirmDialog.type === 'stop' && (
        <ConfirmModal
          title="Stop VM"
          message={`Are you sure you want to stop "${name}"? Any unsaved work inside the VM may be lost.`}
          confirmText="Stop VM"
          variant="warning"
          onConfirm={() => handleConfirmedAction('stop')}
          onCancel={() => setConfirmDialog({ type: null })}
        />
      )}

      {confirmDialog.type === 'restart' && (
        <ConfirmModal
          title="Restart VM"
          message={`Are you sure you want to restart "${name}"? The VM will be briefly unavailable during restart.`}
          confirmText="Restart VM"
          variant="warning"
          onConfirm={() => handleConfirmedAction('restart')}
          onCancel={() => setConfirmDialog({ type: null })}
        />
      )}

      {confirmDialog.type === 'delete' && (
        <ConfirmModal
          title="Delete VM"
          message={`Are you sure you want to delete "${name}"? This action cannot be undone and all data will be permanently lost.`}
          confirmText="Delete VM"
          variant="danger"
          onConfirm={() => handleConfirmedAction('delete')}
          onCancel={() => setConfirmDialog({ type: null })}
        />
      )}

      <style>{`
        .vm-detail {
          padding-bottom: 60px;
        }

        /* Header */
        .header-row {
          display: flex;
          justify-content: space-between;
          align-items: center;
          gap: 24px;
          margin-bottom: 24px;
        }

        .vm-name-section {
          display: flex;
          align-items: center;
          gap: 14px;
          min-width: 0;
        }

        .vm-name-section h1 {
          font-size: 28px;
          font-weight: 600;
          letter-spacing: -0.5px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .state-badge {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 5px 12px;
          border-radius: 16px;
          font-size: 11px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          flex-shrink: 0;
        }

        .state-badge .spinner-icon {
          width: 12px;
          height: 12px;
        }

        .state-running {
          background: rgba(76, 175, 80, 0.15);
          color: var(--success);
        }

        .state-stopped {
          background: rgba(255, 152, 0, 0.15);
          color: var(--warning);
        }

        .state-suspended {
          background: rgba(33, 150, 243, 0.15);
          color: #2196f3;
        }

        .state-unknown,
        .state-starting,
        .state-restarting,
        .state-suspending {
          background: rgba(255, 152, 0, 0.15);
          color: var(--warning);
        }

        .state-badge.transitioning {
          background: rgba(255, 152, 0, 0.15);
          color: var(--warning);
        }

        /* Header Actions */
        .header-actions {
          display: flex;
          gap: 8px;
          flex-shrink: 0;
        }

        .action-btn {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 8px 14px;
          border: 1px solid var(--border);
          border-radius: 6px;
          background: var(--bg-secondary);
          color: var(--text-primary);
          font-size: 13px;
          font-weight: 500;
          transition: all 0.15s;
          white-space: nowrap;
        }

        .action-btn:hover:not(:disabled) {
          border-color: var(--accent);
          color: var(--accent);
        }

        .action-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .action-btn.clone-btn:hover:not(:disabled) {
          border-color: #9c27b0;
          color: #9c27b0;
        }

        .action-btn.success-btn {
          background: rgba(76, 175, 80, 0.1);
          border-color: rgba(76, 175, 80, 0.3);
          color: var(--success);
        }

        .action-btn.success-btn:hover:not(:disabled) {
          background: var(--success);
          border-color: var(--success);
          color: white;
        }

        .action-btn.warning-btn {
          background: rgba(255, 152, 0, 0.08);
          border-color: rgba(255, 152, 0, 0.25);
          color: var(--warning);
        }

        .action-btn.warning-btn:hover:not(:disabled) {
          background: rgba(255, 152, 0, 0.15);
          border-color: var(--warning);
          color: var(--warning);
        }

        .action-btn.danger-btn {
          background: rgba(244, 67, 54, 0.08);
          border-color: rgba(244, 67, 54, 0.25);
          color: var(--error);
        }

        .action-btn.danger-btn:hover:not(:disabled) {
          background: rgba(244, 67, 54, 0.15);
          border-color: var(--error);
          color: var(--error);
        }

        /* Error Banner */
        .error-banner {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 12px 16px;
          margin-bottom: 20px;
          background: rgba(244, 67, 54, 0.1);
          border: 1px solid rgba(244, 67, 54, 0.3);
          border-radius: 8px;
          color: var(--error);
          font-size: 14px;
        }

        .error-banner button {
          background: none;
          border: none;
          color: var(--error);
          font-size: 20px;
          cursor: pointer;
          padding: 0 4px;
        }

        /* Quick Access Buttons */
        .quick-access {
          display: flex;
          gap: 12px;
          margin-bottom: 24px;
        }

        .quick-access .tooltip-wrapper {
          flex: 1;
        }

        .quick-btn {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 14px 20px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 10px;
          color: var(--text-primary);
          font-size: 14px;
          font-weight: 500;
          transition: all 0.2s;
          width: 100%;
        }

        .quick-btn:hover:not(:disabled) {
          border-color: var(--accent);
          background: var(--bg-tertiary);
        }

        .quick-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .quick-btn-label {
          flex: 1;
        }

        .quick-btn svg:last-child {
          opacity: 0.4;
        }

        .quick-btn:hover:not(:disabled) svg:last-child {
          opacity: 0.7;
        }

        .quick-btn.terminal-btn:hover:not(:disabled) {
          border-color: var(--success);
          color: var(--success);
        }

        .quick-btn.files-btn:hover:not(:disabled) {
          border-color: #2196f3;
          color: #2196f3;
        }

        .quick-btn.agent-btn:hover:not(:disabled) {
          border-color: #9c27b0;
          color: #9c27b0;
        }

        /* Resource Cards - 4 columns */
        .resource-cards {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 14px;
          margin-bottom: 28px;
        }

        .resource-card {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 10px;
          padding: 16px;
        }

        .resource-header {
          display: flex;
          align-items: center;
          gap: 6px;
          color: var(--text-tertiary);
          font-size: 11px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          margin-bottom: 10px;
        }

        .resource-header svg {
          opacity: 0.6;
        }

        .resource-value {
          font-size: 22px;
          font-weight: 600;
          color: var(--text-primary);
          margin-bottom: 10px;
          font-family: var(--font-mono);
          line-height: 1;
        }

        /* CPU card - no bar */
        .cpu-card .resource-value {
          margin-bottom: 6px;
        }

        .resource-bar {
          height: 4px;
          background: var(--bg-tertiary);
          border-radius: 2px;
          overflow: hidden;
          margin-bottom: 8px;
        }

        .resource-fill {
          height: 100%;
          border-radius: 2px;
          transition: width 0.5s ease;
        }

        .memory-fill {
          background: linear-gradient(90deg, #9c27b0, #e040fb);
        }

        .disk-fill {
          background: linear-gradient(90deg, #ff9800, #ffb74d);
        }

        .resource-label {
          font-size: 11px;
          color: var(--text-tertiary);
        }

        /* Info card - combined IP and release */
        .info-card {
          display: flex;
          flex-direction: column;
        }

        .info-card .resource-header {
          margin-bottom: 12px;
        }

        .info-row {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 6px 0;
        }

        .info-row:first-of-type {
          border-bottom: 1px solid var(--border);
          margin-bottom: 2px;
          padding-bottom: 8px;
        }

        .info-label {
          font-size: 11px;
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.3px;
        }

        .info-value {
          font-size: 13px;
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        /* Tabs */
        .tabs {
          display: flex;
          gap: 4px;
          border-bottom: 1px solid var(--border);
          margin-bottom: 24px;
        }

        .tabs button {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 12px 20px;
          background: transparent;
          border: none;
          color: var(--text-secondary);
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          border-bottom: 2px solid transparent;
          margin-bottom: -1px;
          transition: all 0.2s;
        }

        .tabs button:hover {
          color: var(--text-primary);
        }

        .tabs button.active {
          color: var(--accent);
          border-bottom-color: var(--accent);
        }

        .tab-count {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          min-width: 20px;
          height: 20px;
          padding: 0 6px;
          background: var(--bg-tertiary);
          border-radius: 10px;
          font-size: 11px;
          font-weight: 600;
          color: var(--text-tertiary);
        }

        .tabs button.active .tab-count {
          background: rgba(33, 150, 243, 0.15);
          color: var(--accent);
        }

        /* Tab Content */
        .tab-content {
          min-height: 300px;
        }

        /* Spinner */
        .spinner-icon {
          animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        /* Responsive - tablet */
        @media (max-width: 900px) {
          .resource-cards {
            grid-template-columns: repeat(2, 1fr);
          }
        }

        @media (max-width: 768px) {
          .header-row {
            flex-direction: column;
            align-items: flex-start;
            gap: 16px;
          }

          .vm-name-section {
            flex-wrap: wrap;
          }

          .vm-name-section h1 {
            font-size: 24px;
          }

          .header-actions {
            width: 100%;
            flex-wrap: wrap;
          }

          .action-btn {
            flex: 1;
            min-width: 100px;
            justify-content: center;
          }

          .quick-access {
            flex-direction: column;
          }

          .quick-access .tooltip-wrapper {
            width: 100%;
          }

          .quick-btn {
            width: 100%;
          }

          .tabs {
            overflow-x: auto;
            -webkit-overflow-scrolling: touch;
          }

          .tabs button {
            padding: 12px 16px;
            white-space: nowrap;
          }
        }

        /* Responsive - mobile */
        @media (max-width: 480px) {
          .resource-cards {
            grid-template-columns: 1fr;
          }

          .vm-name-section h1 {
            font-size: 20px;
          }

          .action-btn {
            padding: 8px 10px;
            font-size: 12px;
          }

          .action-btn span:not(.spinner-icon) {
            display: none;
          }

          .action-btn svg {
            margin: 0;
          }
        }
      `}</style>
    </div>
  )
}
