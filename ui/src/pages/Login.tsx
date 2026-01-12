import { useState, useRef, useEffect } from 'react'
import { api } from '../api/client'
import Logo from '../components/Logo'

interface LoginProps {
  onLogin: (token: string) => void
}

export default function Login({ onLogin }: LoginProps) {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      await api.login(token)
      onLogin(token)
    } catch {
      setError('Invalid token. Please check and try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-container">
        <div className="login-header">
          <Logo size={56} className="login-logo" />
          <h1 className="login-title">dabbi</h1>
        </div>

        <form onSubmit={handleSubmit} className="login-form">
          <div className="form-group">
            <label htmlFor="token">Auth Token</label>
            <input
              ref={inputRef}
              id="token"
              type="password"
              className="input"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="Enter your auth token"
              required
              autoComplete="off"
            />
            <span className="form-hint">
              Find your token in <code>~/.dabbi/config.json</code>
            </span>
          </div>

          {error && (
            <div className="login-error" role="alert">
              <AlertIcon />
              <span>{error}</span>
            </div>
          )}

          <button
            type="submit"
            className="btn btn-primary login-btn"
            disabled={loading || !token.trim()}
          >
            {loading ? (
              <>
                <span className="spinner" />
                Verifying...
              </>
            ) : (
              'Sign in'
            )}
          </button>
        </form>
      </div>

      <style>{`
        .login-page {
          min-height: 100vh;
          display: flex;
          justify-content: center;
          align-items: center;
          padding: var(--space-lg);
          background: var(--bg-primary);
        }

        .login-container {
          width: 100%;
          max-width: 360px;
        }

        .login-header {
          text-align: center;
          margin-bottom: var(--space-2xl);
        }

        .login-logo {
          color: var(--accent);
          margin-bottom: var(--space-md);
        }

        .login-title {
          font-size: 28px;
          font-weight: 600;
          font-family: var(--font-mono);
          color: var(--text-primary);
          letter-spacing: -0.02em;
          margin-bottom: var(--space-xs);
        }

        .login-subtitle {
          font-size: var(--text-sm);
          color: var(--text-tertiary);
        }

        .login-form {
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-lg);
          padding: var(--space-xl);
        }

        .login-form .form-group {
          margin-bottom: var(--space-lg);
        }

        .login-form label {
          display: block;
          margin-bottom: var(--space-xs);
          font-size: var(--text-sm);
          font-weight: 500;
          color: var(--text-secondary);
        }

        .login-form .input {
          width: 100%;
        }

        .login-form .form-hint {
          display: block;
          margin-top: var(--space-xs);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .login-form .form-hint code {
          background: var(--bg-primary);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
        }

        .login-error {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
          padding: var(--space-sm) var(--space-md);
          background: var(--error-bg);
          border: 1px solid var(--error);
          border-radius: var(--radius-sm);
          color: var(--error);
          font-size: var(--text-sm);
          margin-bottom: var(--space-lg);
        }

        .login-btn {
          width: 100%;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: var(--space-sm);
        }

        .login-btn .spinner {
          width: 14px;
          height: 14px;
          border: 2px solid rgba(255, 255, 255, 0.3);
          border-top-color: white;
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        @media (max-width: 480px) {
          .login-page {
            padding: var(--space-md);
            align-items: flex-start;
            padding-top: 15vh;
          }

          .login-form {
            padding: var(--space-lg);
          }
        }
      `}</style>
    </div>
  )
}

const AlertIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10"/>
    <line x1="12" y1="8" x2="12" y2="12"/>
    <line x1="12" y1="16" x2="12.01" y2="16"/>
  </svg>
)
