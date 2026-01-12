import { useState, useEffect, useCallback } from 'react'
import { api, MountEntry } from '../api/client'

interface MountsPanelProps {
  vmName: string
  onRefresh: () => void
}

export default function MountsPanel({ vmName, onRefresh }: MountsPanelProps) {
  const [mounts, setMounts] = useState<MountEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [hostPath, setHostPath] = useState('')
  const [vmPath, setVmPath] = useState('')
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<string | null>(null)

  const loadMounts = useCallback(async () => {
    try {
      const data = await api.listMounts(vmName)
      setMounts(data || [])
      setError('')
    } catch (err) {
      setError(`Failed to load mounts: ${err}`)
    } finally {
      setLoading(false)
    }
  }, [vmName])

  useEffect(() => {
    loadMounts()
  }, [loadMounts])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!hostPath.trim() || !vmPath.trim()) return

    setCreating(true)
    setError('')
    try {
      await api.addMount(vmName, hostPath.trim(), vmPath.trim())
      setHostPath('')
      setVmPath('')
      loadMounts()
      onRefresh()
    } catch (err) {
      setError(`Failed to create mount: ${err}`)
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (mountVmPath: string) => {
    setDeleting(mountVmPath)
    try {
      await api.removeMount(vmName, mountVmPath)
      loadMounts()
      onRefresh()
    } catch (err) {
      setError(`Failed to remove mount: ${err}`)
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="mounts-panel">
      <div className="mount-create">
        <form onSubmit={handleCreate}>
            <input
              type="text"
              value={hostPath}
              onChange={(e) => setHostPath(e.target.value)}
              placeholder="Host path (e.g. /Users/me/code)"
            />
            <span className="mount-arrow">→</span>
            <input
              type="text"
              value={vmPath}
              onChange={(e) => setVmPath(e.target.value)}
              placeholder="VM path (e.g. /home/ubuntu/code)"
            />
          <button type="submit" disabled={creating || !hostPath.trim() || !vmPath.trim()}>
            {creating ? 'Mounting...' : 'Mount'}
          </button>
        </form>
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading-state">Loading mounts...</div>
      ) : mounts.length === 0 ? (
        <div className="empty-state">
          <p>No mounts configured</p>
          <p className="hint">Mount host directories to share files with the VM</p>
        </div>
      ) : (
        <div className="mount-list">
          {mounts.map((mount) => (
            <div key={mount.vm_path} className="mount-item">
              <div className="mount-info">
                <div className="mount-mapping">
                  <span className="mount-host">{mount.host_path}</span>
                  <span className="mount-arrow">→</span>
                  <span className="mount-vm">{mount.vm_path}</span>
                </div>
              </div>
              <div className="mount-actions">
                <button
                  className="btn-danger"
                  onClick={() => handleDelete(mount.vm_path)}
                  disabled={deleting === mount.vm_path}
                >
                  {deleting === mount.vm_path ? '...' : 'Unmount'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <style>{`
        .mounts-panel {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 8px;
          overflow: hidden;
        }
        .mount-create {
          padding: 16px;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border);
        }
        .mount-create form {
          display: flex;
          align-items: center;
          gap: 12px;
        }
        .mount-create input {
          flex: 1;
          padding: 10px 14px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 6px;
          color: var(--text-primary);
          font-size: 14px;
          font-family: var(--font-mono);
        }
        .mount-create input:focus {
          outline: none;
          border-color: var(--accent);
        }
        .mount-create .mount-arrow {
          color: var(--text-tertiary);
          font-size: 16px;
          flex-shrink: 0;
        }
        .mount-create button {
          padding: 10px 20px;
          background: var(--accent);
          border: none;
          border-radius: 6px;
          color: var(--bg-primary);
          font-size: 14px;
          font-weight: 500;
          white-space: nowrap;
        }
        .mount-create button:hover:not(:disabled) {
          background: var(--accent-hover);
        }
        .mount-create button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
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
        .mount-list {
          max-height: 400px;
          overflow-y: auto;
        }
        .mount-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 16px;
          border-bottom: 1px solid var(--border);
        }
        .mount-item:last-child {
          border-bottom: none;
        }
        .mount-mapping {
          display: flex;
          align-items: center;
          gap: 12px;
          font-family: var(--font-mono);
          font-size: 14px;
        }
        .mount-host {
          color: var(--text-secondary);
          max-width: 300px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        .mount-arrow {
          color: var(--text-tertiary);
        }
        .mount-vm {
          color: var(--accent);
          font-weight: 500;
        }
        .mount-actions {
          display: flex;
          gap: 8px;
        }
        .mount-actions button {
          padding: 6px 14px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: 4px;
          color: var(--text-primary);
          font-size: 13px;
          transition: all 0.2s;
        }
        .mount-actions .btn-danger {
          border-color: var(--error);
          color: var(--error);
        }
        .mount-actions .btn-danger:hover:not(:disabled) {
          background: var(--error);
          color: white;
        }
        .mount-actions button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .mount-create form {
            flex-direction: column;
            align-items: stretch;
          }
          .mount-create input {
            width: 100%;
          }
          .mount-create .mount-arrow {
            align-self: center;
            transform: rotate(90deg);
          }
          .mount-create button {
            width: 100%;
          }
          .mount-item {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
          }
          .mount-mapping {
            flex-direction: column;
            align-items: flex-start;
            gap: 4px;
          }
          .mount-mapping .mount-arrow {
            transform: rotate(90deg);
            align-self: flex-start;
            margin-left: 20px;
          }
          .mount-actions {
            width: 100%;
          }
          .mount-actions button {
            width: 100%;
          }
        }
      `}</style>
    </div>
  )
}
