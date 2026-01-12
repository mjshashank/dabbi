import { useEffect, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

const getTheme = () => {
  const saved = localStorage.getItem('dabbi_theme')
  if (saved) return saved === 'dark'
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

const darkTheme = {
  background: '#0d0d0d',
  foreground: '#e5e5e5',
  cursor: '#3b9eff',
  cursorAccent: '#0d0d0d',
  selectionBackground: 'rgba(59, 158, 255, 0.3)',
  black: '#0d0d0d',
  red: '#ef4444',
  green: '#22c55e',
  yellow: '#f59e0b',
  blue: '#3b9eff',
  magenta: '#a855f7',
  cyan: '#06b6d4',
  white: '#e5e5e5',
  brightBlack: '#737373',
  brightRed: '#f87171',
  brightGreen: '#4ade80',
  brightYellow: '#fbbf24',
  brightBlue: '#60a5fa',
  brightMagenta: '#c084fc',
  brightCyan: '#22d3ee',
  brightWhite: '#ffffff',
}

const lightTheme = {
  background: '#ffffff',
  foreground: '#1a1a1a',
  cursor: '#0066cc',
  cursorAccent: '#ffffff',
  selectionBackground: 'rgba(0, 102, 204, 0.2)',
  black: '#1a1a1a',
  red: '#cc0000',
  green: '#1a8754',
  yellow: '#cc7000',
  blue: '#0066cc',
  magenta: '#7c3aed',
  cyan: '#0891b2',
  white: '#ffffff',
  brightBlack: '#666666',
  brightRed: '#ef4444',
  brightGreen: '#22c55e',
  brightYellow: '#f59e0b',
  brightBlue: '#3b82f6',
  brightMagenta: '#a855f7',
  brightCyan: '#06b6d4',
  brightWhite: '#ffffff',
}

export default function TerminalPage() {
  const { name } = useParams<{ name: string }>()
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)

  useEffect(() => {
    if (!containerRef.current || !name) return

    const isDark = getTheme()

    // Lock body scroll
    document.body.style.overflow = 'hidden'
    document.title = `${name} - Terminal`

    // Create terminal
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: '"IBM Plex Mono", monospace',
      scrollback: 10000,
      scrollSensitivity: 3,
      theme: isDark ? darkTheme : lightTheme,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.loadAddon(new WebLinksAddon())
    term.open(containerRef.current)
    termRef.current = term

    // Initial fit
    requestAnimationFrame(() => {
      fitAddon.fit()
      term.focus()
    })

    // Connect WebSocket
    const token = localStorage.getItem('dabbi_token')
    if (!token) {
      term.writeln('\x1b[31mNo authentication token\x1b[0m')
      return
    }

    const { cols, rows } = term
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(
      `${protocol}//${location.host}/api/vms/${name}/shell?token=${token}&cols=${cols}&rows=${rows}`
    )
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
    }

    ws.onmessage = (e) => {
      term.write(typeof e.data === 'string' ? e.data : new Uint8Array(e.data))
    }

    ws.onclose = () => {
      term.writeln('\r\n\x1b[33mConnection closed\x1b[0m')
    }

    ws.onerror = () => {
      term.writeln('\r\n\x1b[31mConnection error\x1b[0m')
    }

    // User input → WebSocket
    const inputDisposable = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(data)
    })

    // Resize handling
    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    })
    resizeObserver.observe(containerRef.current)

    // Touch scroll handling - xterm.js doesn't have native momentum scrolling
    // Best practice: target .xterm-screen element specifically (per xterm.js GitHub issues #594, #5377)
    const screen = containerRef.current.querySelector('.xterm-screen') as HTMLElement | null

    let lastTouchY = 0
    let touchVelocity = 0
    let lastTouchTime = 0
    let momentumFrame: number | null = null

    const handleTouchStart = (e: TouchEvent) => {
      // Stop any ongoing momentum
      if (momentumFrame) {
        cancelAnimationFrame(momentumFrame)
        momentumFrame = null
      }
      lastTouchY = e.touches[0].clientY
      lastTouchTime = performance.now()
      touchVelocity = 0
    }

    const handleTouchMove = (e: TouchEvent) => {
      const touchY = e.touches[0].clientY
      const now = performance.now()
      const deltaY = lastTouchY - touchY
      const deltaTime = now - lastTouchTime

      // Calculate velocity for momentum (pixels per ms)
      if (deltaTime > 0) {
        // Smooth velocity with previous value
        const newVelocity = deltaY / deltaTime
        touchVelocity = touchVelocity * 0.4 + newVelocity * 0.6
      }

      // Scroll immediately - 1 line per 10 pixels (as recommended in xterm.js issues)
      const lines = Math.round(deltaY / 10)
      if (lines !== 0) {
        term.scrollLines(lines)
        lastTouchY = touchY
        lastTouchTime = now
      }
    }

    const handleTouchEnd = () => {
      // Apply momentum/ballistic scrolling
      const applyMomentum = () => {
        if (Math.abs(touchVelocity) < 0.005) {
          momentumFrame = null
          return
        }

        // Convert velocity to lines (16ms frame time, 10px per line)
        const lines = Math.round(touchVelocity * 16 / 10)
        if (lines !== 0) {
          term.scrollLines(lines)
        }

        // Decay velocity (friction)
        touchVelocity *= 0.95

        momentumFrame = requestAnimationFrame(applyMomentum)
      }

      if (Math.abs(touchVelocity) > 0.1) {
        momentumFrame = requestAnimationFrame(applyMomentum)
      }
    }

    // Apply touch-action: none to .xterm-screen to prevent browser interference
    if (screen) {
      screen.style.touchAction = 'none'
      screen.addEventListener('touchstart', handleTouchStart, { passive: true })
      screen.addEventListener('touchmove', handleTouchMove, { passive: true })
      screen.addEventListener('touchend', handleTouchEnd, { passive: true })
    }

    // Cleanup
    return () => {
      document.body.style.overflow = ''
      inputDisposable.dispose()
      resizeObserver.disconnect()
      if (momentumFrame) cancelAnimationFrame(momentumFrame)
      if (screen) {
        screen.removeEventListener('touchstart', handleTouchStart)
        screen.removeEventListener('touchmove', handleTouchMove)
        screen.removeEventListener('touchend', handleTouchEnd)
      }
      ws.close()
      term.dispose()
    }
  }, [name])

  const isDark = getTheme()

  return (
    <>
      <div
        ref={containerRef}
        onClick={() => termRef.current?.focus()}
        style={{
          position: 'fixed',
          inset: 0,
          height: '100dvh',
          background: isDark ? '#0d0d0d' : '#ffffff',
        }}
      />
      <style>{`
        body { margin: 0; }
        .xterm { height: 100%; }
      `}</style>
    </>
  )
}
