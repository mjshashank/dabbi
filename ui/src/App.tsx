import { useState, useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import VMDetail from './pages/VMDetail'
import TerminalPage from './pages/TerminalPage'
import FilesPage from './pages/FilesPage'
import Login from './pages/Login'
import Logo from './components/Logo'
import { api } from './api/client'
import { ThemeContext, Theme } from './context/ThemeContext'

// Icons
const SunIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="5"/>
    <line x1="12" y1="1" x2="12" y2="3"/>
    <line x1="12" y1="21" x2="12" y2="23"/>
    <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
    <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
    <line x1="1" y1="12" x2="3" y2="12"/>
    <line x1="21" y1="12" x2="23" y2="12"/>
    <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
    <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
  </svg>
)

const MoonIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
  </svg>
)

const LogoutIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
    <polyline points="16 17 21 12 16 7"/>
    <line x1="21" y1="12" x2="9" y2="12"/>
  </svg>
)

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [loading, setLoading] = useState(true)
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('dabbi_theme') as Theme
    if (saved) return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('dabbi_theme', theme)
  }, [theme])

  const toggleTheme = () => {
    setTheme(prev => prev === 'light' ? 'dark' : 'light')
  }

  useEffect(() => {
    const token = localStorage.getItem('dabbi_token')
    if (token) {
      api.setToken(token)
      api.listVMs()
        .then(() => setIsAuthenticated(true))
        .catch(() => {
          localStorage.removeItem('dabbi_token')
          setIsAuthenticated(false)
        })
        .finally(() => setLoading(false))
    } else {
      setLoading(false)
    }
  }, [])

  const handleLogin = (token: string) => {
    api.setToken(token)
    localStorage.setItem('dabbi_token', token)
    setIsAuthenticated(true)
  }

  const handleLogout = () => {
    localStorage.removeItem('dabbi_token')
    setIsAuthenticated(false)
  }

  if (loading) {
    return (
      <ThemeContext.Provider value={{ theme, toggleTheme }}>
        <div className="loading-screen">
          <div className="loading-spinner" />
        </div>
        <style>{`
          .loading-screen {
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
          }
          .loading-spinner {
            width: 24px;
            height: 24px;
            border: 2px solid var(--border);
            border-top-color: var(--accent);
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
          }
          @keyframes spin {
            to { transform: rotate(360deg); }
          }
        `}</style>
      </ThemeContext.Provider>
    )
  }

  if (!isAuthenticated) {
    return (
      <ThemeContext.Provider value={{ theme, toggleTheme }}>
        <Login onLogin={handleLogin} />
      </ThemeContext.Provider>
    )
  }

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      <Routes>
        {/* Full-screen pages (no header) */}
        <Route path="/vm/:name/terminal" element={<TerminalPage />} />
        <Route path="/vm/:name/files" element={<FilesPage />} />

        {/* Main layout with header */}
        <Route path="/*" element={
          <div className="app">
            <header className="header">
              <div className="header-content container">
                <div className="header-left">
                  <a href="/" className="logo">
                    <Logo size={24} />
                    <span>dabbi</span>
                  </a>
                </div>
                <div className="header-right">
                  <button
                    className="btn btn-icon btn-ghost theme-toggle"
                    onClick={toggleTheme}
                    title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
                  >
                    {theme === 'light' ? <MoonIcon /> : <SunIcon />}
                  </button>
                  <button
                    className="btn btn-icon btn-ghost"
                    onClick={handleLogout}
                    title="Sign out"
                  >
                    <LogoutIcon />
                  </button>
                </div>
              </div>
            </header>
            <main className="main">
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/vm/:name" element={<VMDetail />} />
                <Route path="*" element={<Navigate to="/" />} />
              </Routes>
            </main>
          </div>
        } />
      </Routes>

      <style>{`
        .app {
          min-height: 100vh;
          display: flex;
          flex-direction: column;
        }

        .header {
          background: var(--bg-secondary);
          border-bottom: 1px solid var(--border);
          position: sticky;
          top: 0;
          z-index: 100;
        }

        .header-content {
          display: flex;
          align-items: center;
          justify-content: space-between;
          height: 48px;
        }

        .header-left {
          display: flex;
          align-items: center;
          gap: var(--space-lg);
        }

        .logo {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
          color: var(--text-primary);
          font-weight: 600;
          font-size: var(--text-lg);
          text-decoration: none;
        }

        .logo:hover {
          color: var(--text-primary);
          text-decoration: none;
        }

        .logo svg {
          color: var(--accent);
        }

        .header-right {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
        }

        .theme-toggle {
          color: var(--text-secondary);
        }

        .main {
          flex: 1;
          padding: var(--space-xl) 0;
        }
      `}</style>
    </ThemeContext.Provider>
  )
}

export default App
