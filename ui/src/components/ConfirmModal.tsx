import { useEffect, useRef } from 'react'

interface ConfirmModalProps {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'warning' | 'default'
  onConfirm: () => void
  onCancel: () => void
}

export default function ConfirmModal({
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'default',
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  const confirmRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    // Focus the cancel button by default for safety
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCancel()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onCancel()
    }
  }

  return (
    <div className="modal-overlay" onClick={handleOverlayClick}>
      <div className="modal confirm-modal" role="alertdialog" aria-modal="true">
        <div className="modal-header">
          <h3 className="modal-title">{title}</h3>
        </div>
        <div className="modal-body">
          <p className="confirm-message">{message}</p>
        </div>
        <div className="modal-footer">
          <button className="btn btn-secondary" onClick={onCancel}>
            {cancelText}
          </button>
          <button
            ref={confirmRef}
            className={`btn ${variant === 'danger' ? 'btn-danger-solid' : variant === 'warning' ? 'btn-warning-solid' : 'btn-primary'}`}
            onClick={onConfirm}
          >
            {confirmText}
          </button>
        </div>
      </div>

      <style>{`
        .confirm-modal {
          max-width: 400px;
          animation: modalSlideIn 0.15s ease-out;
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

        .confirm-message {
          color: var(--text-secondary);
          line-height: 1.6;
        }

        .btn-danger-solid {
          background: var(--error);
          color: white;
        }

        .btn-danger-solid:hover:not(:disabled) {
          filter: brightness(1.1);
        }

        .btn-warning-solid {
          background: var(--warning);
          color: white;
        }

        .btn-warning-solid:hover:not(:disabled) {
          filter: brightness(1.1);
        }
      `}</style>
    </div>
  )
}
