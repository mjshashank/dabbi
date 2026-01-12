import { useNavigate } from 'react-router-dom'
import { VM } from '../api/client'

interface VMCardProps {
  vm: VM
  onAction: (name: string, action: string) => void
}

export default function VMCard({ vm, onAction }: VMCardProps) {
  const navigate = useNavigate()

  const handleClick = () => {
    navigate(`/vm/${vm.name}`)
  }

  const handleAction = (e: React.MouseEvent, action: string) => {
    e.stopPropagation()
    onAction(vm.name, action)
  }

  return (
    <div className="vm-card" onClick={handleClick}>
      <div className="vm-card-header">
        <h3>{vm.name}</h3>
        <span className={`state state-${vm.state.toLowerCase()}`}>
          {vm.state}
        </span>
      </div>

      <div className="vm-card-info">
        <div className="info-row">
          <span className="label">IP</span>
          <span className="value">{vm.ipv4?.[0] || '-'}</span>
        </div>
        <div className="info-row">
          <span className="label">Release</span>
          <span className="value">{vm.release || '-'}</span>
        </div>
      </div>

      <div className="vm-card-actions">
        {vm.state === 'Running' ? (
          <>
            <button onClick={(e) => handleAction(e, 'stop')}>Stop</button>
            <button onClick={(e) => handleAction(e, 'restart')}>Restart</button>
          </>
        ) : (
          <button className="btn-start" onClick={(e) => handleAction(e, 'start')}>
            Start
          </button>
        )}
        <button className="btn-danger" onClick={(e) => handleAction(e, 'delete')}>
          Delete
        </button>
      </div>

      <style>{`
        .vm-card {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 12px;
          padding: 20px;
          cursor: pointer;
          transition: all 0.2s;
        }
        .vm-card:hover {
          border-color: var(--accent);
          transform: translateY(-2px);
        }
        .vm-card-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 16px;
        }
        .vm-card-header h3 {
          font-size: 18px;
          font-weight: 600;
        }
        .state {
          padding: 4px 12px;
          border-radius: 20px;
          font-size: 12px;
          font-weight: 600;
          text-transform: uppercase;
        }
        .state-running {
          background: rgba(76, 175, 80, 0.2);
          color: var(--success);
        }
        .state-stopped {
          background: rgba(255, 152, 0, 0.2);
          color: var(--warning);
        }
        .state-suspended {
          background: rgba(33, 150, 243, 0.2);
          color: #2196f3;
        }
        .state-starting, .state-restarting {
          background: rgba(156, 39, 176, 0.2);
          color: #9c27b0;
        }
        .vm-card-info {
          margin-bottom: 16px;
        }
        .info-row {
          display: flex;
          justify-content: space-between;
          padding: 8px 0;
          border-bottom: 1px solid var(--border);
        }
        .info-row:last-child {
          border-bottom: none;
        }
        .info-row .label {
          color: var(--text-secondary);
          font-size: 13px;
        }
        .info-row .value {
          font-family: monospace;
          font-size: 13px;
        }
        .vm-card-actions {
          display: flex;
          gap: 8px;
          flex-wrap: wrap;
        }
        .vm-card-actions button {
          flex: 1;
          min-width: 70px;
          padding: 8px 12px;
          border: 1px solid var(--border);
          border-radius: 6px;
          background: transparent;
          color: var(--text-primary);
          font-size: 13px;
          transition: all 0.2s;
        }
        .vm-card-actions button:hover {
          border-color: var(--accent);
          color: var(--accent);
        }
        .vm-card-actions .btn-start {
          background: var(--success);
          border-color: var(--success);
          color: white;
        }
        .vm-card-actions .btn-start:hover {
          background: #45a049;
        }
        .vm-card-actions .btn-danger:hover {
          border-color: var(--error);
          color: var(--error);
        }
      `}</style>
    </div>
  )
}
