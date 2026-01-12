import { useEffect, useRef, useState, useCallback } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { WebglAddon } from '@xterm/addon-webgl'
import '@fontsource/ibm-plex-mono/400.css'
import '@fontsource/ibm-plex-mono/700.css'
import '@xterm/xterm/css/xterm.css'

const TERMINAL_FONT = '"IBM Plex Mono", monospace'

interface TerminalProps {
  vmName: string
  isRunning: boolean
}

export default function Terminal({ vmName, isRunning }: TerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')

  const connectWebSocket = useCallback((term: XTerm, fitAddon: FitAddon) => {
    setStatus('connecting')

    fitAddon.fit()
    const { cols, rows } = term

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/vms/${vmName}/shell?cols=${cols}&rows=${rows}`

    const ws = new WebSocket(wsUrl)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    // CRITICAL: Use a single TextDecoder with streaming mode to handle
    // UTF-8 sequences that span WebSocket message boundaries.
    const decoder = new TextDecoder('utf-8', { fatal: false })

    ws.onopen = () => {
      setStatus('connected')
      term.writeln('\x1b[32mConnected to VM shell\x1b[0m\r\n')
      const { cols, rows } = term
      ws.send(JSON.stringify({ type: 'resize', cols, rows }))
    }

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        term.write(event.data)
      } else if (event.data instanceof ArrayBuffer) {
        // Use stream: true to handle partial UTF-8 sequences at message boundaries
        term.write(decoder.decode(event.data, { stream: true }))
      }
    }

    ws.onclose = () => {
      setStatus('disconnected')
      term.writeln('\r\n\x1b[33mConnection closed\x1b[0m')
    }

    ws.onerror = () => {
      setStatus('error')
      term.writeln('\r\n\x1b[31mConnection error\x1b[0m')
    }

    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })
  }, [vmName])

  useEffect(() => {
    if (!terminalRef.current || !isRunning) return

    let term: XTerm | null = null
    let cleanup = () => {}

    // Wait for IBM Plex Mono font to load before initializing terminal
    document.fonts.load('14px "IBM Plex Mono"').then(() => {
      if (!terminalRef.current) return

      term = new XTerm({
        cursorBlink: true,
        cursorStyle: 'block',
        fontSize: 14,
        fontFamily: TERMINAL_FONT,
        fontWeight: '400',
        fontWeightBold: '700',
        scrollback: 10000,
        customGlyphs: true,
        rescaleOverlappingGlyphs: true,
        theme: {
          background: '#1a1a2e',
          foreground: '#e0e0e0',
          cursor: '#00d4ff',
          cursorAccent: '#1a1a2e',
          selectionBackground: 'rgba(0, 212, 255, 0.3)',
          black: '#1a1a2e',
          red: '#f44336',
          green: '#4caf50',
          yellow: '#ff9800',
          blue: '#2196f3',
          magenta: '#9c27b0',
          cyan: '#00bcd4',
          white: '#e0e0e0',
          brightBlack: '#6c7086',
          brightRed: '#ef5350',
          brightGreen: '#66bb6a',
          brightYellow: '#ffb74d',
          brightBlue: '#42a5f5',
          brightMagenta: '#ab47bc',
          brightCyan: '#26c6da',
          brightWhite: '#ffffff',
        },
      })

      const fitAddon = new FitAddon()
      const webLinksAddon = new WebLinksAddon()

      term.loadAddon(fitAddon)
      term.loadAddon(webLinksAddon)

      term.open(terminalRef.current)

      // Try to use WebGL renderer
      try {
        const webglAddon = new WebglAddon()
        webglAddon.onContextLoss(() => webglAddon.dispose())
        term.loadAddon(webglAddon)
      } catch {
        console.log('WebGL not supported, using canvas renderer')
      }

      xtermRef.current = term
      fitAddonRef.current = fitAddon

      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          fitAddon.fit()
          connectWebSocket(term!, fitAddon)
        })
      })

      const handleResize = () => {
        fitAddon.fit()
        if (wsRef.current?.readyState === WebSocket.OPEN && term) {
          const { cols, rows } = term
          wsRef.current.send(JSON.stringify({ type: 'resize', cols, rows }))
        }
      }

      window.addEventListener('resize', handleResize)

      cleanup = () => {
        window.removeEventListener('resize', handleResize)
        wsRef.current?.close()
        term?.dispose()
      }
    })

    return () => cleanup()
  }, [vmName, isRunning, connectWebSocket])

  const handleReconnect = () => {
    if (xtermRef.current && fitAddonRef.current) {
      xtermRef.current.clear()
      connectWebSocket(xtermRef.current, fitAddonRef.current)
    }
  }

  if (!isRunning) {
    return (
      <div className="terminal-disabled">
        <p>VM must be running to access terminal</p>
        <style>{`
          .terminal-disabled {
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
    <div className="terminal-container">
      <div className="terminal-header">
        <div className="terminal-status">
          <span className={`status-dot ${status}`}></span>
          <span className="status-text">
            {status === 'connecting' && 'Connecting...'}
            {status === 'connected' && 'Connected'}
            {status === 'disconnected' && 'Disconnected'}
            {status === 'error' && 'Error'}
          </span>
        </div>
        {(status === 'disconnected' || status === 'error') && (
          <button className="btn-reconnect" onClick={handleReconnect}>
            Reconnect
          </button>
        )}
      </div>
      <div ref={terminalRef} className="terminal-content" />

      <style>{`
        .terminal-container {
          background: #1a1a2e;
          border: 1px solid var(--border);
          border-radius: 8px;
          overflow: hidden;
        }
        .terminal-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 10px 15px;
          background: rgba(0, 0, 0, 0.3);
          border-bottom: 1px solid var(--border);
        }
        .terminal-status {
          display: flex;
          align-items: center;
          gap: 8px;
        }
        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }
        .status-dot.connecting {
          background: var(--warning);
          animation: pulse 1s infinite;
        }
        .status-dot.connected {
          background: var(--success);
        }
        .status-dot.disconnected {
          background: var(--text-secondary);
        }
        .status-dot.error {
          background: var(--error);
        }
        .status-text {
          font-size: 12px;
          color: var(--text-secondary);
        }
        .btn-reconnect {
          padding: 4px 12px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: 4px;
          color: var(--text-secondary);
          font-size: 12px;
          cursor: pointer;
        }
        .btn-reconnect:hover {
          border-color: var(--accent);
          color: var(--accent);
        }
        .terminal-content {
          height: 400px;
        }
        .terminal-content .xterm {
          height: 100%;
        }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }
      `}</style>
    </div>
  )
}
