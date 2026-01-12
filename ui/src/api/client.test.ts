import { describe, it, expect, beforeEach } from 'vitest'
import { api } from './client'
import { server } from '../__tests__/setup'
import { http, HttpResponse } from 'msw'

describe('APIClient', () => {
  beforeEach(() => {
    api.setToken('')
  })

  describe('token management', () => {
    it('should set and get token', () => {
      api.setToken('test-token')
      expect(api.getToken()).toBe('test-token')
    })

    it('should start with empty token', () => {
      expect(api.getToken()).toBe('')
    })
  })

  describe('login', () => {
    it('should set token on successful login', async () => {
      await api.login('valid-token')
      expect(api.getToken()).toBe('valid-token')
    })

    it('should throw on invalid token', async () => {
      await expect(api.login('invalid-token')).rejects.toThrow('Invalid token')
    })

    it('should throw on server error', async () => {
      server.use(
        http.post('/api/auth/login', () => {
          return HttpResponse.json({ error: 'Server error' }, { status: 500 })
        })
      )
      await expect(api.login('test')).rejects.toThrow('Server error')
    })
  })

  describe('logout', () => {
    it('should clear token on logout', async () => {
      api.setToken('test-token')
      await api.logout()
      expect(api.getToken()).toBe('')
    })
  })

  describe('listVMs', () => {
    it('should return list of VMs', async () => {
      const vms = await api.listVMs()
      expect(vms).toHaveLength(2)
      expect(vms[0].name).toBe('test-vm')
      expect(vms[0].state).toBe('Running')
    })

    it('should handle empty list', async () => {
      server.use(http.get('/api/vms', () => HttpResponse.json([])))
      const vms = await api.listVMs()
      expect(vms).toHaveLength(0)
    })

    it('should throw on error', async () => {
      server.use(
        http.get('/api/vms', () => {
          return HttpResponse.json({ error: 'Unauthorized' }, { status: 401 })
        })
      )
      await expect(api.listVMs()).rejects.toThrow('Unauthorized')
    })
  })

  describe('getVM', () => {
    it('should return VM info', async () => {
      const vm = await api.getVM('test-vm')
      expect(vm.state).toBe('Running')
      expect(vm.ipv4).toContain('192.168.1.100')
    })

    it('should throw for non-existent VM', async () => {
      server.use(
        http.get('/api/vms/:name', ({ params }) => {
          if (params.name === 'nonexistent') {
            return HttpResponse.json({ error: 'VM not found' }, { status: 404 })
          }
          return HttpResponse.json({})
        })
      )
      await expect(api.getVM('nonexistent')).rejects.toThrow('VM not found')
    })
  })

  describe('createVM', () => {
    it('should create VM with provided options', async () => {
      server.use(
        http.post('/api/vms', async ({ request }) => {
          const body = (await request.json()) as { name: string }
          return HttpResponse.json(
            { status: 'created', name: body.name },
            { status: 201 }
          )
        })
      )
      const result = await api.createVM({ name: 'new-vm' })
      expect(result.status).toBe('created')
      expect(result.name).toBe('new-vm')
    })
  })

  describe('deleteVM', () => {
    it('should delete VM', async () => {
      const result = await api.deleteVM('test-vm')
      expect(result.status).toBe('ok')
    })
  })

  describe('startVM', () => {
    it('should start VM', async () => {
      const result = await api.startVM('test-vm')
      expect(result.status).toBe('ok')
    })
  })

  describe('stopVM', () => {
    it('should stop VM', async () => {
      const result = await api.stopVM('test-vm')
      expect(result.status).toBe('ok')
    })
  })

  describe('restartVM', () => {
    it('should restart VM', async () => {
      const result = await api.restartVM('test-vm')
      expect(result.status).toBe('ok')
    })
  })

  describe('cloneVM', () => {
    it('should clone VM', async () => {
      const result = await api.cloneVM('test-vm', 'clone-vm')
      expect(result.status).toBe('ok')
      expect(result.name).toBe('clone-vm')
    })
  })

  describe('snapshots', () => {
    it('should list snapshots', async () => {
      const snapshots = await api.listSnapshots('test-vm')
      expect(snapshots).toHaveProperty('snap1')
      expect(snapshots.snap1.comment).toBe('Before update')
    })

    it('should create snapshot', async () => {
      const result = await api.createSnapshot('test-vm', 'new-snap')
      expect(result.status).toBe('ok')
    })

    it('should restore snapshot', async () => {
      const result = await api.restoreSnapshot('test-vm', 'snap1')
      expect(result.status).toBe('ok')
    })

    it('should delete snapshot', async () => {
      const result = await api.deleteSnapshot('test-vm', 'snap1')
      expect(result.status).toBe('ok')
    })
  })

  describe('files', () => {
    it('should list files', async () => {
      server.use(
        http.get('/api/vms/:name/files', () => {
          return HttpResponse.json({
            path: '/home/ubuntu',
            entries: [
              { name: 'Documents', is_dir: true, mode: 'drwxr-xr-x', size: 4096 },
              { name: 'file.txt', is_dir: false, mode: '-rw-r--r--', size: 1234 },
            ],
          })
        })
      )
      const files = await api.listFiles('test-vm', '/home/ubuntu')
      expect(files.entries).toHaveLength(2)
      expect(files.entries[0].name).toBe('Documents')
    })
  })

  describe('mounts', () => {
    it('should list mounts', async () => {
      server.use(
        http.get('/api/vms/:name/mounts', () => {
          return HttpResponse.json([
            { host_path: '/home/user/project', vm_path: '/project' },
          ])
        })
      )
      const mounts = await api.listMounts('test-vm')
      expect(mounts).toHaveLength(1)
    })

    it('should add mount', async () => {
      const result = await api.addMount('test-vm', '/home/user', '/mnt')
      expect(result.status).toBe('ok')
    })

    it('should remove mount', async () => {
      const result = await api.removeMount('test-vm', '/mnt')
      expect(result.status).toBe('ok')
    })
  })

  describe('tunnels', () => {
    it('should list tunnels', async () => {
      server.use(
        http.get('/api/tunnels', () => {
          return HttpResponse.json([
            { host_port: 12345, vm_name: 'test-vm', vm_port: 8080 },
          ])
        })
      )
      const tunnels = await api.listTunnels()
      expect(tunnels).toHaveLength(1)
    })

    it('should create tunnel', async () => {
      const result = await api.createTunnel('test-vm', 8080)
      expect(result.host_port).toBe(12345)
      expect(result.vm_name).toBe('test-vm')
    })

    it('should delete tunnel', async () => {
      const result = await api.deleteTunnel(12345)
      expect(result.status).toBe('ok')
    })
  })

  describe('network', () => {
    it('should get network config', async () => {
      const config = await api.getNetworkConfig('test-vm')
      expect(config.mode).toBe('none')
    })

    it('should update network config', async () => {
      const result = await api.updateNetworkConfig('test-vm', {
        mode: 'allowlist',
        rules: [{ type: 'ip', value: '8.8.8.8' }],
      })
      expect(result.status).toBe('ok')
    })

    it('should remove network config', async () => {
      const result = await api.removeNetworkConfig('test-vm')
      expect(result.status).toBe('ok')
    })
  })

  describe('agent', () => {
    it('should get agent URL', async () => {
      const result = await api.getAgentURL('test-vm')
      expect(result.url).toBe('http://localhost:11234')
    })
  })

  describe('error handling', () => {
    it('should parse JSON error response', async () => {
      server.use(
        http.get('/api/vms', () => {
          return HttpResponse.json(
            { error: 'Custom error message' },
            { status: 400 }
          )
        })
      )
      await expect(api.listVMs()).rejects.toThrow('Custom error message')
    })

    it('should handle plain text error response', async () => {
      server.use(
        http.get('/api/vms', () => {
          return new HttpResponse('Plain text error', { status: 500 })
        })
      )
      await expect(api.listVMs()).rejects.toThrow('Plain text error')
    })

    it('should handle empty response on success', async () => {
      server.use(
        http.post('/api/vms/:name/state', () => {
          return new HttpResponse(null, { status: 200 })
        })
      )
      const result = await api.startVM('test-vm')
      expect(result).toEqual({})
    })
  })
})
