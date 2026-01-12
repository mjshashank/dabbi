import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { api, FileEntry } from '../api/client'

// Icons
const ArrowLeftIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="19" y1="12" x2="5" y2="12"/>
    <polyline points="12,19 5,12 12,5"/>
  </svg>
)

const FolderIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M22,19a2,2,0,0,1-2,2H4a2,2,0,0,1-2-2V5A2,2,0,0,1,4,3H9l2,3h9a2,2,0,0,1,2,2Z"/>
  </svg>
)

const FileIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M13,2H6a2,2,0,0,0-2,2V20a2,2,0,0,0,2,2H18a2,2,0,0,0,2-2V9Z"/>
    <polyline points="13,2 13,9 20,9"/>
  </svg>
)

const RefreshIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="23,4 23,10 17,10"/>
    <path d="M20.49,15a9,9,0,1,1-2.12-9.36L23,10"/>
  </svg>
)

const ChevronUpIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="18,15 12,9 6,15"/>
  </svg>
)

const HomeIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M3,9l9-7,9,7v11a2,2,0,0,1-2,2H5a2,2,0,0,1-2-2Z"/>
    <polyline points="9,22 9,12 15,12 15,22"/>
  </svg>
)

const UploadIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
    <polyline points="17 8 12 3 7 8"/>
    <line x1="12" y1="3" x2="12" y2="15"/>
  </svg>
)

const DownloadIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
    <polyline points="7 10 12 15 17 10"/>
    <line x1="12" y1="15" x2="12" y2="3"/>
  </svg>
)

export default function FilesPage() {
  const { name } = useParams<{ name: string }>()
  const [currentPath, setCurrentPath] = useState('/home/ubuntu')
  const [files, setFiles] = useState<FileEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [downloading, setDownloading] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Get theme from localStorage or system preference
  const getTheme = () => {
    const saved = localStorage.getItem('dabbi_theme')
    if (saved) return saved === 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  }

  const isDark = getTheme()

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light')
    document.title = `${name} - Files`

    // Initialize API token
    const token = localStorage.getItem('dabbi_token')
    if (token) {
      api.setToken(token)
    }
  }, [name, isDark])

  const loadFiles = useCallback(async (path: string) => {
    if (!name) return

    setLoading(true)
    setError('')
    try {
      const response = await api.listFiles(name, path)
      setFiles(response.entries || [])
      setCurrentPath(response.path)
    } catch (err) {
      setError(`Failed to load files: ${err}`)
      setFiles([])
    } finally {
      setLoading(false)
    }
  }, [name])

  useEffect(() => {
    loadFiles('/home/ubuntu')
  }, [loadFiles])

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

  const handleGoHome = () => {
    loadFiles('/home/ubuntu')
  }

  const handlePathSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    loadFiles(currentPath)
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
    return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
  }

  const handleBack = () => {
    window.close()
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !name) return

    setUploading(true)
    setError('')
    try {
      const targetPath = currentPath.endsWith('/') ? currentPath : `${currentPath}/`
      await api.uploadFile(name, targetPath, file)
      loadFiles(currentPath)
    } catch (err) {
      setError(`Upload failed: ${err}`)
    } finally {
      setUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const handleDownload = async (file: FileEntry) => {
    if (!name || file.is_dir) return

    setDownloading(file.name)
    try {
      const filePath = currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`
      const blob = await api.downloadFile(name, filePath)

      // Create download link
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = file.name
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
    } catch (err) {
      setError(`Download failed: ${err}`)
    } finally {
      setDownloading(null)
    }
  }

  // Sort: directories first, then alphabetically
  const sortedFiles = [...files].sort((a, b) => {
    if (a.is_dir && !b.is_dir) return -1
    if (!a.is_dir && b.is_dir) return 1
    return a.name.localeCompare(b.name)
  })

  return (
    <div className="files-page">
      <header className="files-header">
        <div className="header-left">
          <button className="btn btn-ghost btn-sm" onClick={handleBack} title="Close">
            <ArrowLeftIcon />
          </button>
          <span className="vm-name">{name}</span>
        </div>
        <div className="header-right">
          <input
            ref={fileInputRef}
            type="file"
            onChange={handleUpload}
            style={{ display: 'none' }}
          />
          <button
            className="btn btn-action btn-sm"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            title="Upload file"
          >
            <UploadIcon />
            {uploading ? 'Uploading...' : 'Upload'}
          </button>
          <button className="btn btn-ghost btn-sm" onClick={() => loadFiles(currentPath)} title="Refresh">
            <RefreshIcon />
          </button>
        </div>
      </header>

      <div className="toolbar">
        <button
          className="btn btn-ghost btn-sm"
          onClick={handleGoUp}
          disabled={currentPath === '/'}
          title="Go up"
        >
          <ChevronUpIcon />
        </button>
        <button
          className="btn btn-ghost btn-sm"
          onClick={handleGoHome}
          title="Go to home"
        >
          <HomeIcon />
        </button>
        <form className="path-form" onSubmit={handlePathSubmit}>
          <input
            type="text"
            className="path-input"
            value={currentPath}
            onChange={(e) => setCurrentPath(e.target.value)}
          />
        </form>
      </div>

      {error && (
        <div className="error-message">{error}</div>
      )}

      <div className="files-content">
        {loading ? (
          <div className="empty-state">Loading files...</div>
        ) : sortedFiles.length === 0 ? (
          <div className="empty-state">Empty directory</div>
        ) : (
          <div className="files-list">
            <div className="files-list-header">
              <div className="col-name">Name</div>
              <div className="col-size">Size</div>
              <div className="col-mode">Mode</div>
              <div className="col-actions"></div>
            </div>
            {sortedFiles.map((file) => (
              <div
                key={file.name}
                className={`files-list-row${file.is_dir ? ' is-dir' : ''}`}
                onClick={() => handleNavigate(file)}
              >
                <div className="col-name">
                  <span className="file-icon">
                    {file.is_dir ? <FolderIcon /> : <FileIcon />}
                  </span>
                  <span className="file-name">{file.name}</span>
                </div>
                <div className="col-size">{file.is_dir ? '-' : formatSize(file.size)}</div>
                <div className="col-mode">{file.mode || '-'}</div>
                <div className="col-actions">
                  {!file.is_dir && (
                    <button
                      className="btn btn-ghost btn-xs"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleDownload(file)
                      }}
                      disabled={downloading === file.name}
                      title="Download"
                    >
                      {downloading === file.name ? '...' : <DownloadIcon />}
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <style>{`
        .files-page {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background: var(--bg-primary);
        }

        .files-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-sm) var(--space-lg);
          background: var(--bg-secondary);
          border-bottom: 1px solid var(--border);
          flex-shrink: 0;
        }

        .header-left {
          display: flex;
          align-items: center;
          gap: var(--space-md);
        }

        .header-right {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
        }

        .vm-name {
          font-family: var(--font-mono);
          font-weight: 500;
          font-size: var(--text-sm);
        }

        .toolbar {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
          padding: var(--space-sm) var(--space-lg);
          background: var(--bg-secondary);
          border-bottom: 1px solid var(--border);
          flex-shrink: 0;
        }

        .path-form {
          flex: 1;
        }

        .path-input {
          width: 100%;
          padding: var(--space-xs) var(--space-md);
          background: var(--bg-primary);
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
          font-family: var(--font-mono);
          font-size: var(--text-sm);
          color: var(--text-primary);
        }

        .path-input:focus {
          outline: none;
          border-color: var(--accent);
        }

        .error-message {
          padding: var(--space-md) var(--space-lg);
          background: var(--error-bg);
          color: var(--error);
          font-size: var(--text-sm);
        }

        .files-content {
          flex: 1;
          overflow: auto;
        }

        .empty-state {
          padding: var(--space-2xl);
          text-align: center;
          color: var(--text-secondary);
        }

        /* File list using CSS Grid */
        .files-list {
          font-size: var(--text-sm);
        }

        .files-list-header,
        .files-list-row {
          display: grid;
          grid-template-columns: 1fr 80px 100px 44px;
          align-items: center;
        }

        .files-list-header {
          position: sticky;
          top: 0;
          background: var(--bg-tertiary);
          font-size: var(--text-xs);
          font-weight: 500;
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.04em;
          border-bottom: 1px solid var(--border);
        }

        .files-list-header > div {
          padding: var(--space-sm) var(--space-md);
        }

        .files-list-row {
          border-bottom: 1px solid var(--border);
        }

        .files-list-row:last-child {
          border-bottom: none;
        }

        .files-list-row > div {
          padding: var(--space-sm) var(--space-md);
        }

        .files-list-row.is-dir {
          cursor: pointer;
        }

        .files-list-row.is-dir:hover {
          background: var(--accent-bg);
        }

        /* Column styles */
        .col-name {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
          min-width: 0;
        }

        .col-size {
          text-align: right;
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--text-secondary);
        }

        .col-mode {
          text-align: right;
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--text-secondary);
        }

        .col-actions {
          text-align: center;
        }

        .files-list-header .col-size,
        .files-list-header .col-mode {
          text-align: right;
        }

        .file-icon {
          flex-shrink: 0;
          display: flex;
          align-items: center;
          color: var(--text-secondary);
        }

        .is-dir .file-icon {
          color: var(--accent);
        }

        .file-name {
          font-family: var(--font-mono);
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .btn-xs {
          padding: var(--space-xs);
          font-size: var(--text-xs);
        }

        .btn-action {
          display: inline-flex;
          align-items: center;
          gap: var(--space-xs);
          padding: var(--space-xs) var(--space-md);
          background: var(--accent);
          color: var(--bg-primary);
          border: none;
          border-radius: var(--radius-sm);
          font-size: var(--text-sm);
          font-weight: 500;
          cursor: pointer;
          transition: background 0.15s;
        }

        .btn-action:hover:not(:disabled) {
          background: var(--accent-hover);
        }

        .btn-action:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }
      `}</style>
    </div>
  )
}
