import { useState, useEffect, useCallback } from 'react'
import { api, NetworkConfig, NetworkRule, NetworkMode } from '../api/client'

interface NetworkPanelProps {
  vmName: string
  isRunning: boolean
}

const MODES: { value: NetworkMode; label: string; description: string; color: string }[] = [
  { value: 'none', label: 'None', description: 'No network restrictions', color: 'var(--success)' },
  { value: 'allowlist', label: 'Allowlist', description: 'Only allow specific hosts', color: '#2196f3' },
  { value: 'blocklist', label: 'Blocklist', description: 'Block specific hosts', color: '#ff9800' },
  { value: 'isolated', label: 'Isolated', description: 'No network access at all', color: 'var(--error)' },
]

export default function NetworkPanel({ vmName, isRunning }: NetworkPanelProps) {
  const [config, setConfig] = useState<NetworkConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  // Edit state
  const [editMode, setEditMode] = useState<NetworkMode>('none')
  const [editRules, setEditRules] = useState<NetworkRule[]>([])
  const [newRule, setNewRule] = useState('')
  const [hasChanges, setHasChanges] = useState(false)

  const loadConfig = useCallback(async () => {
    if (!isRunning) {
      setLoading(false)
      return
    }

    try {
      const data = await api.getNetworkConfig(vmName)
      setConfig(data)
      setEditMode(data.mode || 'none')
      setEditRules(data.rules || [])
      setError('')
      setHasChanges(false)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      if (!msg.includes('must be running')) {
        setError(`Failed to load network config: ${msg}`)
      }
    } finally {
      setLoading(false)
    }
  }, [vmName, isRunning])

  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  // Track changes
  useEffect(() => {
    if (!config) {
      setHasChanges(editMode !== 'none' || editRules.length > 0)
      return
    }

    const modeChanged = editMode !== (config.mode || 'none')
    const rulesChanged = JSON.stringify(editRules) !== JSON.stringify(config.rules || [])
    setHasChanges(modeChanged || rulesChanged)
  }, [editMode, editRules, config])

  const handleModeChange = (mode: NetworkMode) => {
    setEditMode(mode)
    // Clear rules when switching to isolated or none
    if (mode === 'isolated' || mode === 'none') {
      setEditRules([])
    }
  }

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
    if (editRules.some(r => r.type === rule.type && r.value === rule.value)) {
      setError('This rule already exists')
      return
    }

    setEditRules([...editRules, rule])
    setNewRule('')
    setError('')
  }

  const handleRemoveRule = (index: number) => {
    setEditRules(editRules.filter((_, i) => i !== index))
  }

  const handleSave = async () => {
    // Validate
    if ((editMode === 'allowlist' || editMode === 'blocklist') && editRules.length === 0) {
      setError(`${editMode === 'allowlist' ? 'Allowlist' : 'Blocklist'} mode requires at least one rule`)
      return
    }

    setSaving(true)
    setError('')

    try {
      if (editMode === 'none') {
        await api.removeNetworkConfig(vmName)
      } else {
        await api.updateNetworkConfig(vmName, {
          mode: editMode,
          rules: editRules,
        })
      }
      await loadConfig()
    } catch (err) {
      setError(`Failed to save: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    setEditMode(config?.mode || 'none')
    setEditRules(config?.rules || [])
    setError('')
  }

  const getRuleIcon = (type: string) => {
    switch (type) {
      case 'ip':
        return <IpIcon />
      case 'cidr':
        return <CidrIcon />
      case 'domain':
        return <DomainIcon />
      default:
        return null
    }
  }

  const getModeInfo = (mode: NetworkMode) => {
    return MODES.find(m => m.value === mode) || MODES[0]
  }

  if (!isRunning) {
    return (
      <div className="network-panel">
        <div className="empty-state">
          <NetworkIcon />
          <p>Start the VM to configure network restrictions</p>
          <p className="hint">Network settings can only be viewed and modified when the VM is running</p>
        </div>
        <style>{styles}</style>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="network-panel">
        <div className="loading-state">
          <div className="loading-spinner" />
          <span>Loading network configuration...</span>
        </div>
        <style>{styles}</style>
      </div>
    )
  }

  const currentModeInfo = getModeInfo(editMode)

  return (
    <div className="network-panel">
      {/* Current Status Banner */}
      <div className="status-banner" style={{ borderColor: currentModeInfo.color }}>
        <div className="status-icon" style={{ background: currentModeInfo.color }}>
          {editMode === 'none' ? <UnlockedIcon /> :
           editMode === 'isolated' ? <ShieldIcon /> :
           editMode === 'allowlist' ? <ChecklistIcon /> :
           <BlockIcon />}
        </div>
        <div className="status-text">
          <span className="status-mode" style={{ color: currentModeInfo.color }}>
            {currentModeInfo.label}
          </span>
          <span className="status-desc">{currentModeInfo.description}</span>
        </div>
        {hasChanges && (
          <span className="unsaved-badge">Unsaved changes</span>
        )}
      </div>

      {/* Mode Selector */}
      <div className="mode-section">
        <label className="section-label">Network Mode</label>
        <div className="mode-grid">
          {MODES.map((mode) => (
            <button
              key={mode.value}
              className={`mode-option ${editMode === mode.value ? 'active' : ''}`}
              onClick={() => handleModeChange(mode.value)}
              style={{
                '--mode-color': mode.color,
                borderColor: editMode === mode.value ? mode.color : undefined,
              } as React.CSSProperties}
            >
              <span className="mode-label">{mode.label}</span>
              <span className="mode-desc">{mode.description}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Rules Section - only for allowlist/blocklist */}
      {(editMode === 'allowlist' || editMode === 'blocklist') && (
        <div className="rules-section">
          <label className="section-label">
            {editMode === 'allowlist' ? 'Allowed Hosts' : 'Blocked Hosts'}
          </label>

          {/* Add Rule Form */}
          <div className="add-rule-form">
            <input
              type="text"
              value={newRule}
              onChange={(e) => setNewRule(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAddRule()}
              placeholder="github.com, 10.0.0.0/8, or 192.168.1.1"
            />
            <button
              onClick={handleAddRule}
              disabled={!newRule.trim()}
              className="add-rule-btn"
            >
              Add
            </button>
          </div>
          <p className="rule-hint">
            Enter a domain name, IP address, or CIDR range
          </p>

          {/* Rules List */}
          {editRules.length === 0 ? (
            <div className="rules-empty">
              <p>No rules configured</p>
              <p className="hint">
                {editMode === 'allowlist'
                  ? 'Add hosts that this VM should be able to reach'
                  : 'Add hosts that this VM should NOT be able to reach'}
              </p>
            </div>
          ) : (
            <div className="rules-list">
              {editRules.map((rule, index) => (
                <div key={index} className="rule-item">
                  <span className="rule-icon">{getRuleIcon(rule.type)}</span>
                  <span className="rule-type">{rule.type}</span>
                  <span className="rule-value">{rule.value}</span>
                  <button
                    className="rule-remove"
                    onClick={() => handleRemoveRule(index)}
                    aria-label="Remove rule"
                  >
                    <CloseIcon />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Isolated Mode Info */}
      {editMode === 'isolated' && (
        <div className="isolated-info">
          <ShieldIcon />
          <p>
            <strong>Complete Network Isolation</strong>
          </p>
          <p>
            The VM will have no network access except for the connection to the host.
            This is useful for running untrusted code or ensuring complete isolation.
          </p>
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="error-message">
          <AlertIcon />
          <span>{error}</span>
          <button onClick={() => setError('')} className="dismiss-btn">
            <CloseIcon />
          </button>
        </div>
      )}

      {/* Action Buttons */}
      {hasChanges && (
        <div className="actions">
          <button className="btn-secondary" onClick={handleCancel} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Applying...' : 'Apply Changes'}
          </button>
        </div>
      )}

      <style>{styles}</style>
    </div>
  )
}

// Icons
const NetworkIcon = () => (
  <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <circle cx="12" cy="12" r="10" />
    <line x1="2" y1="12" x2="22" y2="12" />
    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
  </svg>
)

const UnlockedIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
    <path d="M7 11V7a5 5 0 0 1 9.9-1" />
  </svg>
)

const ShieldIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
  </svg>
)

const ChecklistIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M9 11l3 3L22 4" />
    <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
  </svg>
)

const BlockIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
  </svg>
)

const IpIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
    <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
    <line x1="6" y1="6" x2="6.01" y2="6" />
    <line x1="6" y1="18" x2="6.01" y2="18" />
  </svg>
)

const CidrIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="3" />
    <circle cx="12" cy="12" r="9" />
  </svg>
)

const DomainIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <line x1="2" y1="12" x2="22" y2="12" />
    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
  </svg>
)

const CloseIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="18" y1="6" x2="6" y2="18" />
    <line x1="6" y1="6" x2="18" y2="18" />
  </svg>
)

const AlertIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <line x1="12" y1="8" x2="12" y2="12" />
    <line x1="12" y1="16" x2="12.01" y2="16" />
  </svg>
)

const styles = `
  .network-panel {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }

  .loading-state {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 60px 20px;
    color: var(--text-secondary);
  }

  .loading-spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 60px 20px;
    text-align: center;
    color: var(--text-secondary);
  }

  .empty-state svg {
    opacity: 0.4;
    margin-bottom: 8px;
  }

  .empty-state .hint {
    font-size: 13px;
    color: var(--text-tertiary);
    max-width: 300px;
  }

  /* Status Banner */
  .status-banner {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 16px 20px;
    background: var(--bg-primary);
    border-bottom: 2px solid;
  }

  .status-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 10px;
    color: white;
  }

  .status-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .status-mode {
    font-size: 15px;
    font-weight: 600;
  }

  .status-desc {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .unsaved-badge {
    padding: 4px 10px;
    background: rgba(255, 152, 0, 0.15);
    color: var(--warning);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.3px;
    border-radius: 12px;
  }

  /* Section Labels */
  .section-label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-tertiary);
    margin-bottom: 12px;
  }

  /* Mode Section */
  .mode-section {
    padding: 20px;
    border-bottom: 1px solid var(--border);
  }

  .mode-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
  }

  .mode-option {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    padding: 14px;
    background: var(--bg-primary);
    border: 2px solid var(--border);
    border-radius: 8px;
    text-align: left;
    transition: all 0.15s;
  }

  .mode-option:hover {
    border-color: var(--mode-color);
    background: var(--bg-tertiary);
  }

  .mode-option.active {
    background: var(--bg-tertiary);
  }

  .mode-label {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .mode-option.active .mode-label {
    color: var(--mode-color);
  }

  .mode-desc {
    font-size: 11px;
    color: var(--text-tertiary);
    line-height: 1.4;
  }

  /* Rules Section */
  .rules-section {
    padding: 20px;
    border-bottom: 1px solid var(--border);
  }

  .add-rule-form {
    display: flex;
    gap: 10px;
    margin-bottom: 8px;
  }

  .add-rule-form input {
    flex: 1;
    padding: 10px 14px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 14px;
    font-family: var(--font-mono);
  }

  .add-rule-form input:focus {
    outline: none;
    border-color: var(--accent);
  }

  .add-rule-form input::placeholder {
    color: var(--text-tertiary);
    font-family: var(--font-sans);
  }

  .add-rule-btn {
    padding: 10px 20px;
    background: var(--accent);
    border: none;
    border-radius: 6px;
    color: white;
    font-size: 14px;
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

  .rule-hint {
    font-size: 12px;
    color: var(--text-tertiary);
    margin-bottom: 16px;
  }

  .rules-empty {
    padding: 30px;
    text-align: center;
    color: var(--text-secondary);
    background: var(--bg-primary);
    border-radius: 8px;
  }

  .rules-empty .hint {
    font-size: 13px;
    color: var(--text-tertiary);
    margin-top: 4px;
  }

  .rules-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 300px;
    overflow-y: auto;
  }

  .rule-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
  }

  .rule-icon {
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
  }

  .rule-type {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-tertiary);
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 4px;
    min-width: 48px;
    text-align: center;
  }

  .rule-value {
    flex: 1;
    font-family: var(--font-mono);
    font-size: 14px;
    color: var(--text-primary);
  }

  .rule-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 4px;
    color: var(--text-tertiary);
    transition: all 0.15s;
  }

  .rule-remove:hover {
    background: rgba(244, 67, 54, 0.1);
    color: var(--error);
  }

  /* Isolated Info */
  .isolated-info {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 30px 20px;
    text-align: center;
    color: var(--text-secondary);
  }

  .isolated-info svg {
    color: var(--error);
    opacity: 0.7;
  }

  .isolated-info p {
    margin: 0;
    max-width: 400px;
  }

  .isolated-info strong {
    color: var(--text-primary);
  }

  /* Error Message */
  .error-message {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    margin: 20px;
    margin-top: 0;
    background: rgba(244, 67, 54, 0.1);
    border: 1px solid rgba(244, 67, 54, 0.3);
    border-radius: 8px;
    color: var(--error);
    font-size: 14px;
  }

  .error-message span {
    flex: 1;
  }

  .dismiss-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 4px;
    color: var(--error);
    opacity: 0.7;
    transition: all 0.15s;
  }

  .dismiss-btn:hover {
    opacity: 1;
    background: rgba(244, 67, 54, 0.15);
  }

  /* Actions */
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 16px 20px;
    background: var(--bg-primary);
    border-top: 1px solid var(--border);
  }

  .btn-secondary {
    padding: 10px 20px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 14px;
    font-weight: 500;
    transition: all 0.15s;
  }

  .btn-secondary:hover:not(:disabled) {
    border-color: var(--text-secondary);
    background: var(--bg-hover);
  }

  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    padding: 10px 20px;
    background: var(--accent);
    border: none;
    border-radius: 6px;
    color: white;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.15s;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Responsive */
  @media (max-width: 768px) {
    .mode-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .add-rule-form {
      flex-direction: column;
    }

    .add-rule-btn {
      width: 100%;
    }

    .rule-item {
      flex-wrap: wrap;
    }

    .rule-value {
      order: 3;
      width: 100%;
      margin-top: 4px;
      font-size: 13px;
    }

    .actions {
      flex-direction: column;
    }

    .actions button {
      width: 100%;
    }
  }

  @media (max-width: 480px) {
    .mode-grid {
      grid-template-columns: 1fr;
    }

    .status-banner {
      flex-direction: column;
      text-align: center;
    }

    .status-text {
      align-items: center;
    }
  }
`
