import { useState, useEffect, useCallback } from 'react'
import { api, Snapshot } from '../api/client'
import ConfirmModal from './ConfirmModal'
import Tooltip from './Tooltip'

interface SnapshotPanelProps {
  vmName: string
  isRunning: boolean
  onRefresh: () => void
}

interface ConfirmState {
  type: 'restore' | 'delete' | null
  snapshotName: string
}

export default function SnapshotPanel({ vmName, isRunning, onRefresh }: SnapshotPanelProps) {
  const [snapshots, setSnapshots] = useState<Record<string, Snapshot>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [newSnapshotName, setNewSnapshotName] = useState('')
  const [creating, setCreating] = useState(false)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [confirmState, setConfirmState] = useState<ConfirmState>({ type: null, snapshotName: '' })

  const loadSnapshots = useCallback(async () => {
    try {
      const data = await api.listSnapshots(vmName)
      setSnapshots(data || {})
      setError('')
    } catch (err) {
      setError(`Failed to load snapshots: ${err}`)
    } finally {
      setLoading(false)
    }
  }, [vmName])

  useEffect(() => {
    loadSnapshots()
  }, [loadSnapshots])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newSnapshotName.trim()) return

    setCreating(true)
    try {
      await api.createSnapshot(vmName, newSnapshotName.trim())
      setNewSnapshotName('')
      loadSnapshots()
      onRefresh()
    } catch (err) {
      alert(`Failed to create snapshot: ${err}`)
    } finally {
      setCreating(false)
    }
  }

  const handleConfirm = async () => {
    const { type, snapshotName } = confirmState
    if (!type || !snapshotName) return

    setActionLoading(snapshotName)
    setConfirmState({ type: null, snapshotName: '' })

    try {
      if (type === 'restore') {
        await api.restoreSnapshot(vmName, snapshotName)
        onRefresh()
      } else if (type === 'delete') {
        await api.deleteSnapshot(vmName, snapshotName)
        loadSnapshots()
        onRefresh()
      }
    } catch (err) {
      alert(`Failed to ${type} snapshot: ${err}`)
    } finally {
      setActionLoading(null)
    }
  }

  const snapshotList = Object.entries(snapshots)

  return (
    <div className="snapshot-panel">
      <div className="snapshot-create">
        <form onSubmit={handleCreate}>
          <input
            type="text"
            value={newSnapshotName}
            onChange={(e) => setNewSnapshotName(e.target.value)}
            placeholder="Snapshot name"
            pattern="[a-zA-Z][a-zA-Z0-9-]*"
            disabled={isRunning}
          />
          <Tooltip text="Stop the VM to create snapshots" show={isRunning} position="bottom">
            <button
              type="submit"
              disabled={creating || !newSnapshotName.trim() || isRunning}
            >
              {creating ? 'Creating...' : 'Create Snapshot'}
            </button>
          </Tooltip>
        </form>
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading-state">Loading snapshots...</div>
      ) : snapshotList.length === 0 ? (
        <div className="empty-state">
          <p>No snapshots yet</p>
          <p className="hint">Create a snapshot to save the current VM state</p>
        </div>
      ) : (
        <div className="snapshot-list">
          {snapshotList.map(([name, snapshot]) => (
            <div key={name} className="snapshot-item">
              <div className="snapshot-info">
                <h4>{name}</h4>
                {snapshot.comment && (
                  <p className="snapshot-comment">{snapshot.comment}</p>
                )}
                {snapshot.parent && (
                  <p className="snapshot-parent">Parent: {snapshot.parent}</p>
                )}
              </div>
              <div className="snapshot-actions">
                <button
                  className="btn-restore"
                  onClick={() => setConfirmState({ type: 'restore', snapshotName: name })}
                  disabled={actionLoading === name || isRunning}
                  title={isRunning ? 'Stop VM to restore snapshot' : ''}
                >
                  {actionLoading === name ? '...' : 'Restore'}
                </button>
                <button
                  className="btn-danger"
                  onClick={() => setConfirmState({ type: 'delete', snapshotName: name })}
                  disabled={actionLoading === name}
                >
                  {actionLoading === name ? '...' : 'Delete'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {confirmState.type === 'restore' && (
        <ConfirmModal
          title="Restore Snapshot"
          message={`Restore "${confirmState.snapshotName}"? This will REVERT all changes since this snapshot was taken. Any unsaved work will be permanently LOST.`}
          confirmText="Restore"
          variant="warning"
          onConfirm={handleConfirm}
          onCancel={() => setConfirmState({ type: null, snapshotName: '' })}
        />
      )}

      {confirmState.type === 'delete' && (
        <ConfirmModal
          title="Delete Snapshot"
          message={`Delete "${confirmState.snapshotName}"? This cannot be undone.`}
          confirmText="Delete"
          variant="danger"
          onConfirm={handleConfirm}
          onCancel={() => setConfirmState({ type: null, snapshotName: '' })}
        />
      )}

      <style>{`
        .snapshot-panel {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 8px;
          overflow: hidden;
        }
        .snapshot-create {
          padding: 16px;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border);
        }
        .snapshot-create form {
          display: flex;
          gap: 12px;
        }
        .snapshot-create input {
          flex: 1;
          padding: 10px 14px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 6px;
          color: var(--text-primary);
          font-size: 14px;
        }
        .snapshot-create input:focus {
          outline: none;
          border-color: var(--accent);
        }
        .snapshot-create button {
          padding: 10px 20px;
          background: var(--accent);
          border: none;
          border-radius: 6px;
          color: var(--bg-primary);
          font-size: 14px;
          font-weight: 500;
          white-space: nowrap;
        }
        .snapshot-create button:hover:not(:disabled) {
          background: var(--accent-hover);
        }
        .snapshot-create button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }
        .snapshot-create input:disabled {
          opacity: 0.6;
        }
        .error-message {
          padding: 12px 16px;
          background: rgba(244, 67, 54, 0.1);
          color: var(--error);
          font-size: 14px;
        }
        .loading-state, .empty-state {
          padding: 40px;
          text-align: center;
          color: var(--text-secondary);
        }
        .empty-state .hint {
          margin-top: 8px;
          font-size: 14px;
        }
        .snapshot-list {
          max-height: 400px;
          overflow-y: auto;
        }
        .snapshot-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 16px;
          border-bottom: 1px solid var(--border);
        }
        .snapshot-item:last-child {
          border-bottom: none;
        }
        .snapshot-info h4 {
          font-size: 15px;
          font-weight: 600;
          margin-bottom: 4px;
        }
        .snapshot-comment, .snapshot-parent {
          font-size: 13px;
          color: var(--text-secondary);
          margin: 0;
        }
        .snapshot-actions {
          display: flex;
          gap: 8px;
        }
        .snapshot-actions button {
          padding: 6px 14px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: 4px;
          color: var(--text-primary);
          font-size: 13px;
          transition: all 0.2s;
        }
        .snapshot-actions button:hover:not(:disabled) {
          border-color: var(--accent);
          color: var(--accent);
        }
        .snapshot-actions .btn-restore {
          border-color: var(--warning);
          color: var(--warning);
        }
        .snapshot-actions .btn-restore:hover:not(:disabled) {
          background: var(--warning);
          color: white;
        }
        .snapshot-actions .btn-danger {
          border-color: var(--error);
          color: var(--error);
        }
        .snapshot-actions .btn-danger:hover:not(:disabled) {
          background: var(--error);
          color: white;
        }
        .snapshot-actions button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .snapshot-create form {
            flex-direction: column;
          }
          .snapshot-create input {
            width: 100%;
          }
          .snapshot-create .tooltip-wrapper {
            width: 100%;
          }
          .snapshot-create button {
            width: 100%;
          }
          .snapshot-item {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
          }
          .snapshot-actions {
            width: 100%;
          }
          .snapshot-actions button {
            flex: 1;
          }
        }
      `}</style>
    </div>
  )
}
