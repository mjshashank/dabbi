import { useState, useEffect, useRef } from "react";
import { api } from "../api/client";

interface CloneVMModalProps {
  sourceName: string;
  onClose: () => void;
  onCloned: () => void;
}

export default function CloneVMModal({
  sourceName,
  onClose,
  onCloned,
}: CloneVMModalProps) {
  const [newName, setNewName] = useState("");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await api.cloneVM(sourceName, newName);
      setLoading(false);
      setSuccess(true);
      setTimeout(() => {
        onCloned();
      }, 1200);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Clone failed");
      setLoading(false);
    }
  };

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (success) {
    return (
      <div className="modal-overlay">
        <div
          className="modal clone-modal clone-modal--success"
          role="dialog"
          aria-modal="true"
        >
          <div className="success-content">
            <div className="success-icon">
              <CheckIcon />
            </div>
            <h3 className="success-title">VM Cloned</h3>
            <p className="success-message">
              <span className="success-vm-name">{newName}</span> created from{" "}
              {sourceName}
            </p>
          </div>

          <style>{`
            .clone-modal--success {
              max-width: 320px;
              animation: modalSlideIn 0.15s ease-out;
            }

            .success-content {
              padding: var(--space-2xl) var(--space-xl);
              text-align: center;
            }

            .success-icon {
              display: inline-flex;
              align-items: center;
              justify-content: center;
              width: 48px;
              height: 48px;
              border-radius: 50%;
              background: var(--success-bg);
              color: var(--success);
              margin-bottom: var(--space-lg);
              animation: successPop 0.4s ease-out;
            }

            @keyframes successPop {
              0% { transform: scale(0); opacity: 0; }
              50% { transform: scale(1.2); }
              100% { transform: scale(1); opacity: 1; }
            }

            .success-title {
              font-size: var(--text-lg);
              font-weight: 600;
              margin-bottom: var(--space-xs);
            }

            .success-message {
              font-size: var(--text-sm);
              color: var(--text-secondary);
            }

            .success-vm-name {
              font-family: var(--font-mono);
              font-weight: 500;
              color: var(--text-primary);
            }
          `}</style>
        </div>
      </div>
    );
  }

  return (
    <div className="modal-overlay" onClick={handleOverlayClick}>
      <div className="modal clone-modal" role="dialog" aria-modal="true">
        {loading && (
          <div className="loading-overlay">
            <div className="loading-spinner" />
            <span className="loading-text">Cloning {sourceName}...</span>
          </div>
        )}

        <div className="modal-header">
          <h3 className="modal-title">Clone VM</h3>
          <button
            className="btn-close"
            onClick={onClose}
            aria-label="Close"
            disabled={loading}
          >
            <CloseIcon />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div className="source-info">
              <span className="source-label">Source</span>
              <span className="source-name">{sourceName}</span>
            </div>

            <div className="form-group">
              <label htmlFor="clone-name">New VM Name</label>
              <input
                ref={inputRef}
                id="clone-name"
                type="text"
                className="input"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder={`${sourceName}-clone`}
                pattern="[a-zA-Z][a-zA-Z0-9-]*"
                required
                autoComplete="off"
              />
              <span className="form-hint">
                Letters, numbers, hyphens. Must start with a letter.
              </span>
            </div>

            {error && (
              <div className="form-error" role="alert">
                <AlertIcon />
                <span>{error}</span>
              </div>
            )}
          </div>

          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
              disabled={loading}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading || !newName}
            >
              Clone
            </button>
          </div>
        </form>
      </div>

      <style>{`
        .clone-modal {
          max-width: 400px;
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

        .source-info {
          display: flex;
          align-items: center;
          gap: var(--space-md);
          padding: var(--space-md);
          background: var(--bg-secondary);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          margin-bottom: var(--space-lg);
        }

        .source-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.03em;
        }

        .source-name {
          font-family: var(--font-mono);
          font-weight: 500;
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

        .form-error {
          display: flex;
          align-items: center;
          gap: var(--space-sm);
          padding: var(--space-sm) var(--space-md);
          background: var(--error-bg);
          border: 1px solid var(--error);
          border-radius: var(--radius-sm);
          color: var(--error);
          font-size: var(--text-sm);
          margin-top: var(--space-md);
        }
      `}</style>
    </div>
  );
}

const CloseIcon = () => (
  <svg
    width="14"
    height="14"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
  >
    <line x1="18" y1="6" x2="6" y2="18" />
    <line x1="6" y1="6" x2="18" y2="18" />
  </svg>
);

const CheckIcon = () => (
  <svg
    width="24"
    height="24"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2.5"
  >
    <polyline points="20 6 9 17 4 12" />
  </svg>
);

const AlertIcon = () => (
  <svg
    width="16"
    height="16"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
  >
    <circle cx="12" cy="12" r="10" />
    <line x1="12" y1="8" x2="12" y2="12" />
    <line x1="12" y1="16" x2="12.01" y2="16" />
  </svg>
);
