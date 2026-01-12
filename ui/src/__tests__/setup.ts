import { afterAll, afterEach, beforeAll } from 'vitest'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import '@testing-library/jest-dom/vitest'

// Mock VM data
export const mockVMs = [
  {
    name: 'test-vm',
    state: 'Running',
    ipv4: ['192.168.1.100'],
    release: 'Ubuntu 24.04 LTS',
  },
  {
    name: 'dev-vm',
    state: 'Stopped',
    ipv4: [],
    release: 'Ubuntu 22.04 LTS',
  },
]

export const mockVMInfo = {
  name: 'test-vm',
  state: 'Running',
  ipv4: ['192.168.1.100'],
  release: 'Ubuntu 24.04 LTS',
  image_hash: 'abc123',
  cpu_count: '2',
  disk_total: '20G',
  disk_used: '5G',
  memory_total: '4G',
  memory_used: '1G',
  mounts: {},
  load: [0.1, 0.2, 0.15],
}

export const mockSnapshots = {
  'snap1': { comment: 'Before update', parent: '' },
  'snap2': { comment: 'After update', parent: 'snap1' },
}

// Default MSW handlers
export const handlers = [
  // Auth endpoints
  http.post('/api/auth/login', async ({ request }) => {
    const body = await request.json() as { token: string }
    if (body.token === 'valid-token') {
      return HttpResponse.json({ status: 'ok' })
    }
    return HttpResponse.json({ error: 'Invalid token' }, { status: 401 })
  }),

  http.post('/api/auth/logout', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  // VM endpoints
  http.get('/api/vms', () => {
    return HttpResponse.json(mockVMs)
  }),

  http.get('/api/vms/:name', ({ params }) => {
    const vm = mockVMs.find((v) => v.name === params.name)
    if (vm) {
      return HttpResponse.json({ ...mockVMInfo, ...vm })
    }
    return HttpResponse.json({ error: 'VM not found' }, { status: 404 })
  }),

  http.post('/api/vms', async ({ request }) => {
    const body = await request.json() as { name: string }
    return HttpResponse.json({ status: 'created', name: body.name }, { status: 201 })
  }),

  http.delete('/api/vms/:name', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  http.post('/api/vms/:name/state', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  http.post('/api/vms/:name/clone', async ({ request }) => {
    const body = await request.json() as { new_name: string }
    return HttpResponse.json({ status: 'ok', name: body.new_name }, { status: 201 })
  }),

  // Snapshot endpoints
  http.get('/api/vms/:name/snapshots', () => {
    return HttpResponse.json(mockSnapshots)
  }),

  http.post('/api/vms/:name/snapshots', () => {
    return HttpResponse.json({ status: 'ok', name: 'new-snap' }, { status: 201 })
  }),

  http.post('/api/vms/:name/snapshots/restore', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  http.delete('/api/vms/:name/snapshots/:snap', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  // Files endpoints
  http.get('/api/vms/:name/files', () => {
    return HttpResponse.json({
      path: '/home/ubuntu',
      entries: [
        { name: 'Documents', is_dir: true, mode: 'drwxr-xr-x', size: 4096 },
        { name: 'file.txt', is_dir: false, mode: '-rw-r--r--', size: 1234 },
      ],
    })
  }),

  // Mounts endpoints
  http.get('/api/vms/:name/mounts', () => {
    return HttpResponse.json([])
  }),

  http.post('/api/vms/:name/mounts', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  http.delete('/api/vms/:name/mounts', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  // Tunnels endpoints
  http.get('/api/tunnels', () => {
    return HttpResponse.json([])
  }),

  http.post('/api/tunnels', () => {
    return HttpResponse.json({ host_port: 12345, vm_name: 'test-vm', vm_port: 8080 })
  }),

  http.delete('/api/tunnels/:port', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  // Network endpoints
  http.get('/api/vms/:name/network', () => {
    return HttpResponse.json({ mode: 'none', rules: [] })
  }),

  http.put('/api/vms/:name/network', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  http.delete('/api/vms/:name/network', () => {
    return HttpResponse.json({ status: 'ok' })
  }),

  // Agent endpoint
  http.get('/api/vms/:name/agent-url', () => {
    return HttpResponse.json({ url: 'http://localhost:11234' })
  }),
]

export const server = setupServer(...handlers)

// Start server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))

// Reset handlers after each test
afterEach(() => server.resetHandlers())

// Clean up after all tests
afterAll(() => server.close())
