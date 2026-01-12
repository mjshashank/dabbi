import { useState, useEffect, useCallback } from 'react'
import { api, FileEntry } from '../api/client'

interface FileBrowserProps {
  vmName: string
  isRunning: boolean
}

export default function FileBrowser({ vmName, isRunning }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState('/home/ubuntu')
  const [files, setFiles] = useState<FileEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const loadFiles = useCallback(async (path: string) => {
    setLoading(true)
    setError('')
    try {
      const response = await api.listFiles(vmName, path)
      setFiles(response.entries || [])
      setCurrentPath(response.path)
    } catch (err) {
      setError(`Failed to load files: ${err}`)
      setFiles([])
    } finally {
      setLoading(false)
    }
  }, [vmName])

  useEffect(() => {
    if (isRunning) {
      loadFiles('/home/ubuntu')
    }
  }, [vmName, isRunning, loadFiles])

  const handleNavigate = (entry: FileEntry) => {
    if (entry.is_dir) {
      const newPath = currentPath === '/'
        ? `/${entry.name}`
        : `${currentPath}/${entry.name}`
      loadFiles(newPath)
    }
  }

  const handleGoUp = () => {
    if (currentPath === '/') return
    const parentPath = currentPath.split('/').slice(0, -1).join('/') || '/'
    loadFiles(parentPath)
  }

  const handlePathChange = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const target = e.target as HTMLInputElement
      loadFiles(target.value)
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '-'
    const units = ['B', 'KB', 'MB', 'GB']
    let i = 0
    let value = bytes
    while (value >= 1024 && i < units.length - 1) {
      value /= 1024
      i++
    }
    return `${value.toFixed(1)} ${units[i]}`
  }

  const formatMode = (mode: string) => {
    return mode || '-'
  }

  if (!isRunning) {
    return (
      <div className="files-disabled">
        <p>VM must be running to browse files</p>
        <style>{`
          .files-disabled {
            background: var(--bg-primary);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 40px;
            text-align: center;
            color: var(--text-secondary);
          }
        `}</style>
      </div>
    )
  }

  return (
    <div className="file-browser">
      <div className="file-browser-header">
        <button
          className="btn-up"
          onClick={handleGoUp}
          disabled={currentPath === '/'}
        >
          ..
        </button>
        <input
          type="text"
          className="path-input"
          value={currentPath}
          onChange={(e) => setCurrentPath(e.target.value)}
          onKeyDown={handlePathChange}
        />
        <button className="btn-refresh" onClick={() => loadFiles(currentPath)}>
          Refresh
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading-state">Loading files...</div>
      ) : (
        <div className="file-list">
          <div className="file-list-header">
            <span className="col-name">Name</span>
            <span className="col-size">Size</span>
            <span className="col-mode">Mode</span>
          </div>
          {files.length === 0 ? (
            <div className="empty-state">No files found</div>
          ) : (
            files.map((file) => (
              <div
                key={file.name}
                className={`file-item ${file.is_dir ? 'is-dir' : ''}`}
                onClick={() => handleNavigate(file)}
              >
                <span className="col-name">
                  <span className="file-icon">
                    {file.is_dir ? '📁' : '📄'}
                  </span>
                  {file.name}
                </span>
                <span className="col-size">{formatSize(file.size)}</span>
                <span className="col-mode">{formatMode(file.mode)}</span>
              </div>
            ))
          )}
        </div>
      )}

      <style>{`
        .file-browser {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 8px;
          overflow: hidden;
        }
        .file-browser-header {
          display: flex;
          gap: 10px;
          padding: 12px;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border);
        }
        .btn-up {
          padding: 8px 16px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: 6px;
          color: var(--text-primary);
          font-weight: bold;
        }
        .btn-up:hover:not(:disabled) {
          border-color: var(--accent);
          color: var(--accent);
        }
        .btn-up:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
        .path-input {
          flex: 1;
          padding: 8px 12px;
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: 6px;
          color: var(--text-primary);
          font-family: monospace;
          font-size: 14px;
        }
        .path-input:focus {
          outline: none;
          border-color: var(--accent);
        }
        .btn-refresh {
          padding: 8px 16px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: 6px;
          color: var(--text-primary);
        }
        .btn-refresh:hover {
          border-color: var(--accent);
          color: var(--accent);
        }
        .error-message {
          padding: 12px;
          background: rgba(244, 67, 54, 0.1);
          color: var(--error);
          font-size: 14px;
        }
        .loading-state, .empty-state {
          padding: 40px;
          text-align: center;
          color: var(--text-secondary);
        }
        .file-list {
          max-height: 400px;
          overflow-y: auto;
        }
        .file-list-header {
          display: grid;
          grid-template-columns: 1fr 100px 120px;
          gap: 12px;
          padding: 12px 16px;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border);
          font-size: 12px;
          color: var(--text-secondary);
          text-transform: uppercase;
          position: sticky;
          top: 0;
        }
        .file-item {
          display: grid;
          grid-template-columns: 1fr 100px 120px;
          gap: 12px;
          padding: 12px 16px;
          border-bottom: 1px solid var(--border);
          font-size: 14px;
          transition: background 0.2s;
        }
        .file-item:hover {
          background: var(--bg-primary);
        }
        .file-item.is-dir {
          cursor: pointer;
        }
        .file-item.is-dir:hover {
          background: rgba(0, 212, 255, 0.1);
        }
        .file-item:last-child {
          border-bottom: none;
        }
        .col-name {
          display: flex;
          align-items: center;
          gap: 8px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        .file-icon {
          flex-shrink: 0;
        }
        .col-size, .col-mode {
          font-family: monospace;
          color: var(--text-secondary);
        }
      `}</style>
    </div>
  )
}
