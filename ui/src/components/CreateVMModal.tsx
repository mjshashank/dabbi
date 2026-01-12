import { useState, useEffect, useRef } from 'react'
import { api, NetworkMode, NetworkRule } from '../api/client'

interface CreateVMModalProps {
  onClose: () => void
  onCreated: (vmName: string) => void
}

interface FormError {
  message: string
  type: 'network' | 'validation' | 'server'
}

function parseError(err: unknown): FormError {
  // Network errors
  if (err instanceof TypeError && err.message.includes('fetch')) {
    return { message: 'Unable to connect to server. Check your connection.', type: 'network' }
  }

  // Extract message from Error objects
  const message = err instanceof Error ? err.message : String(err)

  // Categorize common errors
  if (message.includes('already exists') || message.includes('duplicate')) {
    return { message: `A VM with this name already exists.`, type: 'validation' }
  }
  if (message.includes('invalid') || message.includes('Invalid')) {
    return { message, type: 'validation' }
  }
  if (message.includes('not found') || message.includes('Not found')) {
    return { message, type: 'server' }
  }
  if (message.includes('unauthorized') || message.includes('Unauthorized')) {
    return { message: 'Session expired. Please sign in again.', type: 'server' }
  }

  return { message: message || 'An unexpected error occurred.', type: 'server' }
}

export default function CreateVMModal({ onClose, onCreated }: CreateVMModalProps) {
  const [name, setName] = useState('')
  const [cpu, setCpu] = useState('')
  const [mem, setMem] = useState('')
  const [disk, setDisk] = useState('')
  const [image, setImage] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingDefaults, setLoadingDefaults] = useState(true)
  const [createdName, setCreatedName] = useState<string | null>(null)
  const [error, setError] = useState<FormError | null>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)

  // Network configuration
  const [showNetwork, setShowNetwork] = useState(false)
  const [networkMode, setNetworkMode] = useState<NetworkMode>('none')
  const [networkRules, setNetworkRules] = useState<NetworkRule[]>([])
  const [newRule, setNewRule] = useState('')
  const hasFocusedRef = useRef(false)

  useEffect(() => {
    api.getDefaults()
      .then((defaults) => {
        setCpu(String(defaults.cpu))
        setMem(defaults.mem)
        setDisk(defaults.disk)
      })
      .catch(() => {
        // Fallback to hardcoded defaults if API fails
        setCpu('2')
        setMem('4G')
        setDisk('20G')
      })
      .finally(() => setLoadingDefaults(false))
  }, [])

  // Focus name input once after defaults load
  useEffect(() => {
    if (!loadingDefaults && !hasFocusedRef.current) {
      hasFocusedRef.current = true
      nameInputRef.current?.focus()
    }
  }, [loadingDefaults])

  // Handle escape key to close modal
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    // Validate network config
    if ((networkMode === 'allowlist' || networkMode === 'blocklist') && networkRules.length === 0) {
      setError({
        message: `${networkMode === 'allowlist' ? 'Allowlist' : 'Blocklist'} mode requires at least one rule`,
        type: 'validation'
      })
      return
    }

    setLoading(true)

    // Notify parent to start polling IMMEDIATELY - don't wait for API
    // The API blocks until VM is fully ready, but it appears in list earlier as "Starting"
    onCreated(name)

    try {
      // Build network config if not "none"
      const networkConfig = networkMode !== 'none' ? {
        mode: networkMode,
        rules: networkRules.length > 0 ? networkRules : undefined,
      } : undefined

      await api.createVM({
        name,
        cpu: parseInt(cpu, 10),
        mem,
        disk,
        image: image || undefined,
        network: networkConfig,
      })
      // API returned successfully - VM is fully ready
      // Parent's polling will close the modal when VM is found
      setCreatedName(name)
    } catch (err) {
      setError(parseError(err))
      setLoading(false)
    }
  }

  // Network helper functions
  const parseRule = (input: string): NetworkRule | null => {
    const value = input.trim()
    if (!value) return null

    // Check if it's a CIDR
    if (value.includes('/')) {
      return { type: 'cidr', value }
    }

    // Check if it's an IP address (simple check)
    const parts = value.split('.')
    if (parts.length === 4 && parts.every(p => /^\d+$/.test(p))) {
      return { type: 'ip', value }
    }

    // Otherwise treat as domain
    return { type: 'domain', value }
  }

  const handleAddRule = () => {
    const rule = parseRule(newRule)
    if (!rule) return

    // Check for duplicates
    if (networkRules.some(r => r.type === rule.type && r.value === rule.value)) {
      return
    }

    setNetworkRules([...networkRules, rule])
    setNewRule('')
  }

  const handleRemoveRule = (index: number) => {
    setNetworkRules(networkRules.filter((_, i) => i !== index))
  }

  const handleNetworkModeChange = (mode: NetworkMode) => {
    setNetworkMode(mode)
    if (mode === 'none' || mode === 'isolated') {
      setNetworkRules([])
    }
  }

  const dismissError = () => setError(null)

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  // Determine loading state
  const isWaiting = loading || createdName !== null

  return (
    <div className="modal-overlay" onClick={handleOverlayClick}>
      <div className="modal create-modal" role="dialog" aria-modal="true">
        {isWaiting && (
          <div className="loading-overlay">
            <div className="loading-spinner" />
            <span className="loading-text">
              {createdName ? `Waiting for ${createdName}...` : `Creating ${name}...`}
            </span>
          </div>
        )}

        <div className="modal-header">
          <h3 className="modal-title">Create VM</h3>
          <button className="btn-close" onClick={onClose} aria-label="Close" disabled={isWaiting}>
            <CloseIcon />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div className="form-group">
              <label htmlFor="vm-name">Name</label>
              <input
                ref={nameInputRef}
                id="vm-name"
                type="text"
                className="input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="my-vm"
                pattern="[a-zA-Z][a-zA-Z0-9-]*"
                required
                autoComplete="off"
              />
              <span className="form-hint">Letters, numbers, hyphens. Must start with a letter.</span>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="vm-cpu">CPUs</label>
                <input
                  id="vm-cpu"
                  type="number"
                  className="input"
                  min="1"
                  max="16"
                  value={cpu}
                  onChange={(e) => setCpu(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label htmlFor="vm-mem">Memory</label>
                <input
                  id="vm-mem"
                  type="text"
                  className="input"
                  value={mem}
                  onChange={(e) => setMem(e.target.value)}
                  placeholder="2G"
                />
              </div>

              <div className="form-group">
                <label htmlFor="vm-disk">Disk</label>
                <input
                  id="vm-disk"
                  type="text"
                  className="input"
                  value={disk}
                  onChange={(e) => setDisk(e.target.value)}
                  placeholder="10G"
                />
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="vm-image">Image</label>
              <input
                id="vm-image"
                type="text"
                className="input"
                value={image}
                onChange={(e) => setImage(e.target.value)}
                placeholder="22.04, jammy, noble (default: latest LTS)"
              />
            </div>

            {/* Network Configuration (collapsible) */}
            <div className="network-section">
              <button
                type="button"
                className="network-toggle"
                onClick={() => setShowNetwork(!showNetwork)}
              >
                <ChevronIcon expanded={showNetwork} />
                <span>Network Restrictions</span>
                {networkMode !== 'none' && (
                  <span className="network-badge">{networkMode}</span>
                )}
              </button>

              {showNetwork && (
                <div className="network-config">
                  <div className="network-modes">
                    {(['none', 'allowlist', 'blocklist', 'isolated'] as NetworkMode[]).map((mode) => (
                      <button
                        key={mode}
                        type="button"
                        className={`network-mode-btn ${networkMode === mode ? 'active' : ''}`}
                        onClick={() => handleNetworkModeChange(mode)}
                      >
                        {mode === 'none' && 'None'}
                        {mode === 'allowlist' && 'Allowlist'}
                        {mode === 'blocklist' && 'Blocklist'}
                        {mode === 'isolated' && 'Isolated'}
                      </button>
                    ))}
                  </div>

                  <p className="network-mode-desc">
                    {networkMode === 'none' && 'No network restrictions'}
                    {networkMode === 'allowlist' && 'Only allow access to specified hosts'}
                    {networkMode === 'blocklist' && 'Block access to specified hosts'}
                    {networkMode === 'isolated' && 'Complete network isolation (no internet access)'}
                  </p>

                  {(networkMode === 'allowlist' || networkMode === 'blocklist') && (
                    <div className="network-rules">
                      <div className="add-rule-row">
                        <input
                          type="text"
                          value={newRule}
                          onChange={(e) => setNewRule(e.target.value)}
                          onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddRule())}
                          placeholder="github.com, 10.0.0.0/8, or 192.168.1.1"
                          className="input"
                        />
                        <button
                          type="button"
                          onClick={handleAddRule}
                          disabled={!newRule.trim()}
                          className="add-rule-btn"
                        >
                          Add
                        </button>
                      </div>

                      {networkRules.length > 0 && (
                        <div className="rules-list">
                          {networkRules.map((rule, index) => (
                            <div key={index} className="rule-tag">
                              <span className="rule-type">{rule.type}</span>
                              <span className="rule-value">{rule.value}</span>
                              <button
                                type="button"
                                onClick={() => handleRemoveRule(index)}
                                className="rule-remove"
                              >
                                <CloseIcon />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>

            {error && (
              <div className={`form-error form-error--${error.type}`} role="alert">
                <div className="form-error__icon">
                  {error.type === 'network' ? <WifiOffIcon /> : <AlertIcon />}
                </div>
                <div className="form-error__content">
                  <span className="form-error__message">{error.message}</span>
                </div>
                <button
                  type="button"
                  className="form-error__dismiss"
                  onClick={dismissError}
                  aria-label="Dismiss error"
                >
                  <CloseIcon />
                </button>
              </div>
            )}
          </div>

          <div className="modal-footer">
            <button type="button" className="btn btn-secondary" onClick={onClose} disabled={isWaiting}>
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={isWaiting || loadingDefaults || !name}>
              Create
            </button>
          </div>
        </form>
      </div>

      <style>{`
        .create-modal {
          max-width: 440px;
          animation: modalSlideIn 0.15s ease-out;
          position: relative;
          overflow: hidden;
        }

        @keyframes modalSlideIn {
          from {
            opacity: 0;
            transform: translateY(-8px) scale(0.98);
          }
          to {
            opacity: 1;
            transform: translateY(0) scale(1);
          }
        }

        .loading-overlay {
          position: absolute;
          inset: 0;
          background: var(--bg-primary);
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: var(--space-md);
          z-index: 10;
          animation: fadeIn 0.15s ease-out;
        }

        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }

        .loading-spinner {
          width: 32px;
          height: 32px;
          border: 2px solid var(--border);
          border-top-color: var(--accent);
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        .loading-text {
          font-size: var(--text-sm);
          color: var(--text-secondary);
          font-family: var(--font-mono);
        }

        .btn-close {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 28px;
          border-radius: var(--radius-sm);
          color: var(--text-tertiary);
          transition: all 0.15s;
        }

        .btn-close:hover {
          background: var(--bg-hover);
          color: var(--text-primary);
        }

        .form-group {
          margin-bottom: var(--space-lg);
        }

        .form-group:last-child {
          margin-bottom: 0;
        }

        .form-group label {
          display: block;
          margin-bottom: var(--space-xs);
          font-size: var(--text-sm);
          font-weight: 500;
          color: var(--text-secondary);
        }

        .form-group .input {
          width: 100%;
        }

        .form-hint {
          display: block;
          margin-top: var(--space-xs);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .form-row {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: var(--space-md);
        }

        .form-row .form-group {
          margin-bottom: var(--space-lg);
        }

        .form-error {
          display: flex;
          align-items: flex-start;
          gap: var(--space-sm);
          padding: var(--space-sm) var(--space-md);
          background: var(--error-bg);
          border: 1px solid var(--error);
          border-radius: var(--radius-sm);
          color: var(--error);
          font-size: var(--text-sm);
          margin-top: var(--space-md);
          animation: errorShake 0.4s ease-out;
        }

        @keyframes errorShake {
          0%, 100% { transform: translateX(0); }
          20% { transform: translateX(-4px); }
          40% { transform: translateX(4px); }
          60% { transform: translateX(-2px); }
          80% { transform: translateX(2px); }
        }

        .form-error__icon {
          flex-shrink: 0;
          margin-top: 1px;
        }

        .form-error__content {
          flex: 1;
          min-width: 0;
        }

        .form-error__message {
          display: block;
          line-height: 1.4;
        }

        .form-error__dismiss {
          flex-shrink: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          width: 20px;
          height: 20px;
          border-radius: var(--radius-xs);
          color: var(--error);
          opacity: 0.7;
          transition: opacity 0.15s, background 0.15s;
        }

        .form-error__dismiss:hover {
          opacity: 1;
          background: rgba(239, 68, 68, 0.15);
        }

        .form-error--network {
          border-color: var(--warning);
          background: var(--warning-bg);
          color: var(--warning);
        }

        .form-error--network .form-error__dismiss {
          color: var(--warning);
        }

        .form-error--network .form-error__dismiss:hover {
          background: rgba(245, 158, 11, 0.15);
        }

        /* Network Configuration Section */
        .network-section {
          margin-top: var(--space-lg);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          overflow: hidden;
        }

        .network-toggle {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
          width: 100%;
          padding: var(--space-md);
          background: var(--bg-tertiary);
          border: none;
          color: var(--text-secondary);
          font-size: var(--text-sm);
          font-weight: 500;
          text-align: left;
          transition: all 0.15s;
        }

        .network-toggle:hover {
          background: var(--bg-hover);
          color: var(--text-primary);
        }

        .network-badge {
          margin-left: auto;
          padding: 2px 8px;
          background: var(--accent-bg);
          color: var(--accent);
          font-size: 10px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.3px;
          border-radius: 10px;
        }

        .network-config {
          padding: var(--space-md);
          background: var(--bg-primary);
          border-top: 1px solid var(--border);
        }

        .network-modes {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: var(--space-xs);
          margin-bottom: var(--space-sm);
        }

        .network-mode-btn {
          padding: var(--space-sm) var(--space-xs);
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
          color: var(--text-secondary);
          font-size: 12px;
          font-weight: 500;
          transition: all 0.15s;
        }

        .network-mode-btn:hover {
          border-color: var(--accent);
          color: var(--text-primary);
        }

        .network-mode-btn.active {
          background: var(--accent-bg);
          border-color: var(--accent);
          color: var(--accent);
        }

        .network-mode-desc {
          font-size: 12px;
          color: var(--text-tertiary);
          margin-bottom: var(--space-md);
          text-align: center;
        }

        .network-rules {
          border-top: 1px solid var(--border);
          padding-top: var(--space-md);
        }

        .add-rule-row {
          display: flex;
          gap: var(--space-sm);
          margin-bottom: var(--space-sm);
        }

        .add-rule-row input {
          flex: 1;
          font-family: var(--font-mono);
          font-size: 13px;
        }

        .add-rule-btn {
          padding: var(--space-sm) var(--space-md);
          background: var(--accent);
          border: none;
          border-radius: var(--radius-sm);
          color: white;
          font-size: 13px;
          font-weight: 500;
          white-space: nowrap;
        }

        .add-rule-btn:hover:not(:disabled) {
          background: var(--accent-hover);
        }

        .add-rule-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .rules-list {
          display: flex;
          flex-wrap: wrap;
          gap: var(--space-xs);
        }

        .rule-tag {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 4px 4px 4px 8px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
        }

        .rule-type {
          font-size: 9px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.3px;
          color: var(--text-tertiary);
          background: var(--bg-tertiary);
          padding: 2px 4px;
          border-radius: 3px;
        }

        .rule-value {
          font-family: var(--font-mono);
          font-size: 12px;
          color: var(--text-primary);
        }

        .rule-remove {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 18px;
          height: 18px;
          border-radius: 3px;
          color: var(--text-tertiary);
          transition: all 0.15s;
        }

        .rule-remove:hover {
          background: rgba(244, 67, 54, 0.1);
          color: var(--error);
        }

        @media (max-width: 480px) {
          .network-modes {
            grid-template-columns: repeat(2, 1fr);
          }

          .add-rule-row {
            flex-direction: column;
          }

          .add-rule-btn {
            width: 100%;
          }
        }
      `}</style>
    </div>
  )
}

const CloseIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="18" y1="6" x2="6" y2="18"/>
    <line x1="6" y1="6" x2="18" y2="18"/>
  </svg>
)

const ChevronIcon = ({ expanded }: { expanded: boolean }) => (
  <svg
    width="14"
    height="14"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    style={{ transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform 0.15s' }}
  >
    <polyline points="9 18 15 12 9 6" />
  </svg>
)

const AlertIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10"/>
    <line x1="12" y1="8" x2="12" y2="12"/>
    <line x1="12" y1="16" x2="12.01" y2="16"/>
  </svg>
)

const WifiOffIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="1" y1="1" x2="23" y2="23"/>
    <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55"/>
    <path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39"/>
    <path d="M10.71 5.05A16 16 0 0 1 22.58 9"/>
    <path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88"/>
    <path d="M8.53 16.11a6 6 0 0 1 6.95 0"/>
    <line x1="12" y1="20" x2="12.01" y2="20"/>
  </svg>
)
