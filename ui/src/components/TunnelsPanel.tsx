import { useState, useEffect, useCallback } from 'react'
import { api, TunnelInfo } from '../api/client'

interface TunnelsPanelProps {
  vmName: string
  isRunning?: boolean
}

export default function TunnelsPanel({ vmName, isRunning = false }: TunnelsPanelProps) {
  const [tunnels, setTunnels] = useState<TunnelInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [vmPort, setVmPort] = useState('')
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<number | null>(null)

  const loadTunnels = useCallback(async () => {
    try {
      const data = await api.listTunnels()
      // Filter tunnels for this VM
      setTunnels((data || []).filter((t) => t.vm_name === vmName))
      setError('')
    } catch (err) {
      setError(`Failed to load tunnels: ${err}`)
    } finally {
      setLoading(false)
    }
  }, [vmName])

  useEffect(() => {
    loadTunnels()
  }, [loadTunnels])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    const port = parseInt(vmPort, 10)
    if (isNaN(port) || port < 1 || port > 65535) {
      setError('Please enter a valid port number (1-65535)')
      return
    }

    setCreating(true)
    setError('')
    try {
      await api.createTunnel(vmName, port)
      setVmPort('')
      loadTunnels()
    } catch (err) {
      setError(`Failed to create tunnel: ${err}`)
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (hostPort: number) => {
    setDeleting(hostPort)
    try {
      await api.deleteTunnel(hostPort)
      loadTunnels()
    } catch (err) {
      setError(`Failed to delete tunnel: ${err}`)
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="tunnels-panel">
      <div className="tunnel-create">
        <form onSubmit={handleCreate}>
          <input
            type="number"
            value={vmPort}
            onChange={(e) => setVmPort(e.target.value)}
            placeholder={isRunning ? "VM port (e.g. 5432)" : "Start VM to create tunnels"}
            min="1"
            max="65535"
            disabled={!isRunning}
          />
          <button type="submit" disabled={!isRunning || creating || !vmPort.trim()}>
            {creating ? 'Creating...' : 'Create Tunnel'}
          </button>
        </form>
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading-state">Loading tunnels...</div>
      ) : tunnels.length === 0 ? (
        <div className="empty-state">
          <p>No active tunnels</p>
          <p className="hint">
            {isRunning
              ? 'Create a tunnel to access VM ports from localhost'
              : 'Start the VM to create tunnels'}
          </p>
        </div>
      ) : (
        <div className="tunnel-list">
          {tunnels.map((tunnel) => (
            <div key={tunnel.host_port} className="tunnel-item">
              <div className="tunnel-info">
                <div className="tunnel-mapping">
                  <span className="tunnel-host">localhost:{tunnel.host_port}</span>
                  <span className="tunnel-arrow">→</span>
                  <span className="tunnel-vm">{tunnel.vm_port}</span>
                </div>
                <p className="tunnel-hint">
                  Connect to <code>localhost:{tunnel.host_port}</code>
                </p>
              </div>
              <div className="tunnel-actions">
                <button
                  className="btn-danger"
                  onClick={() => handleDelete(tunnel.host_port)}
                  disabled={deleting === tunnel.host_port}
                >
                  {deleting === tunnel.host_port ? '...' : 'Close'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <style>{`
        .tunnels-panel {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 8px;
          overflow: hidden;
        }
        .tunnel-create {
          padding: 16px;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border);
        }
        .tunnel-create form {
          display: flex;
          gap: 12px;
        }
        .tunnel-create input {
          flex: 1;
          padding: 10px 14px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 6px;
          color: var(--text-primary);
          font-size: 14px;
          font-family: var(--font-mono);
        }
        .tunnel-create input:focus {
          outline: none;
          border-color: var(--accent);
        }
        .tunnel-create input::-webkit-inner-spin-button,
        .tunnel-create input::-webkit-outer-spin-button {
          -webkit-appearance: none;
          margin: 0;
        }
        .tunnel-create input[type=number] {
          -moz-appearance: textfield;
        }
        .tunnel-create button {
          padding: 10px 20px;
          background: var(--accent);
          border: none;
          border-radius: 6px;
          color: var(--bg-primary);
          font-size: 14px;
          font-weight: 500;
          white-space: nowrap;
        }
        .tunnel-create button:hover:not(:disabled) {
          background: var(--accent-hover);
        }
        .tunnel-create button:disabled {
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
        .tunnel-list {
          max-height: 400px;
          overflow-y: auto;
        }
        .tunnel-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 16px;
          border-bottom: 1px solid var(--border);
        }
        .tunnel-item:last-child {
          border-bottom: none;
        }
        .tunnel-mapping {
          display: flex;
          align-items: center;
          gap: 8px;
          font-family: var(--font-mono);
          font-size: 15px;
          font-weight: 500;
        }
        .tunnel-host {
          color: var(--success);
        }
        .tunnel-arrow {
          color: var(--text-tertiary);
        }
        .tunnel-vm {
          color: var(--text-primary);
        }
        .tunnel-hint {
          font-size: 13px;
          color: var(--text-secondary);
          margin: 4px 0 0;
        }
        .tunnel-hint code {
          font-family: var(--font-mono);
          background: var(--bg-primary);
          padding: 2px 6px;
          border-radius: 4px;
          font-size: 12px;
        }
        .tunnel-actions {
          display: flex;
          gap: 8px;
        }
        .tunnel-actions button {
          padding: 6px 14px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: 4px;
          color: var(--text-primary);
          font-size: 13px;
          transition: all 0.2s;
        }
        .tunnel-actions .btn-danger {
          border-color: var(--error);
          color: var(--error);
        }
        .tunnel-actions .btn-danger:hover:not(:disabled) {
          background: var(--error);
          color: white;
        }
        .tunnel-actions button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .tunnel-create form {
            flex-direction: column;
          }
          .tunnel-create input {
            width: 100%;
          }
          .tunnel-create button {
            width: 100%;
          }
          .tunnel-item {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
          }
          .tunnel-actions {
            width: 100%;
          }
          .tunnel-actions button {
            flex: 1;
          }
        }
      `}</style>
    </div>
  )
}
