import { useState, useEffect, useRef } from 'react'

interface TooltipProps {
  children: React.ReactNode
  text: string
  show?: boolean
  position?: 'top' | 'bottom'
}

export default function Tooltip({ children, text, show = true, position = 'top' }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const wrapperRef = useRef<HTMLDivElement>(null)

  // Close tooltip when clicking outside on mobile
  useEffect(() => {
    if (!visible) return

    const handleClickOutside = (e: MouseEvent | TouchEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setVisible(false)
      }
    }

    document.addEventListener('touchstart', handleClickOutside)
    return () => document.removeEventListener('touchstart', handleClickOutside)
  }, [visible])

  if (!show || !text) {
    return <>{children}</>
  }

  return (
    <div
      ref={wrapperRef}
      className="tooltip-wrapper"
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
      onTouchStart={(e) => {
        e.stopPropagation()
        setVisible(v => !v)
      }}
    >
      {children}
      {visible && (
        <div className={`tooltip-bubble tooltip-${position}`}>
          {text}
        </div>
      )}

      <style>{`
        .tooltip-wrapper {
          position: relative;
          display: inline-flex;
        }

        .tooltip-bubble {
          position: absolute;
          left: 50%;
          transform: translateX(-50%);
          padding: 8px 12px;
          background: var(--bg-primary);
          border: 1px solid var(--border);
          border-radius: 6px;
          font-size: 12px;
          font-weight: 500;
          color: var(--text-secondary);
          white-space: nowrap;
          z-index: 1000;
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
          pointer-events: none;
        }

        /* Top position (default) */
        .tooltip-bubble.tooltip-top {
          bottom: calc(100% + 8px);
        }

        .tooltip-bubble.tooltip-top::after {
          content: '';
          position: absolute;
          top: 100%;
          left: 50%;
          transform: translateX(-50%);
          border: 6px solid transparent;
          border-top-color: var(--border);
        }

        .tooltip-bubble.tooltip-top::before {
          content: '';
          position: absolute;
          top: 100%;
          left: 50%;
          transform: translateX(-50%);
          border: 5px solid transparent;
          border-top-color: var(--bg-primary);
          margin-top: -1px;
          z-index: 1;
        }

        /* Bottom position */
        .tooltip-bubble.tooltip-bottom {
          top: calc(100% + 8px);
        }

        .tooltip-bubble.tooltip-bottom::after {
          content: '';
          position: absolute;
          bottom: 100%;
          left: 50%;
          transform: translateX(-50%);
          border: 6px solid transparent;
          border-bottom-color: var(--border);
        }

        .tooltip-bubble.tooltip-bottom::before {
          content: '';
          position: absolute;
          bottom: 100%;
          left: 50%;
          transform: translateX(-50%);
          border: 5px solid transparent;
          border-bottom-color: var(--bg-primary);
          margin-bottom: -1px;
          z-index: 1;
        }

        /* Mobile: always show below */
        @media (max-width: 768px) {
          .tooltip-bubble {
            max-width: 200px;
            white-space: normal;
            text-align: center;
          }

          .tooltip-bubble.tooltip-top {
            bottom: auto;
            top: calc(100% + 8px);
          }

          .tooltip-bubble.tooltip-top::after {
            top: auto;
            bottom: 100%;
            border-top-color: transparent;
            border-bottom-color: var(--border);
          }

          .tooltip-bubble.tooltip-top::before {
            top: auto;
            bottom: 100%;
            margin-top: 0;
            margin-bottom: -1px;
            border-top-color: transparent;
            border-bottom-color: var(--bg-primary);
          }
        }
      `}</style>
    </div>
  )
}
