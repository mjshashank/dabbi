const API_BASE = '/api'

class APIClient {
  private token: string = ''

  setToken(token: string) {
    this.token = token
  }

  getToken(): string {
    return this.token
  }

  // Login and set HttpOnly cookie
  async login(token: string): Promise<void> {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ token }),
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || 'Login failed')
    }
    this.token = token
  }

  // Logout and clear cookie
  async logout(): Promise<void> {
    await fetch(`${API_BASE}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    })
    this.token = ''
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include', // Send cookies
      body: body ? JSON.stringify(body) : undefined,
    })

    if (!res.ok) {
      const text = await res.text()
      let message = text
      try {
        const json = JSON.parse(text)
        message = json.error || text
      } catch {
        // Use raw text
      }
      throw new Error(message || res.statusText)
    }

    const text = await res.text()
    if (!text) return {} as T
    return JSON.parse(text)
  }

  // Defaults
  getDefaults() {
    return this.request<VMDefaults>('GET', '/defaults')
  }

  // VMs
  listVMs() {
    return this.request<VM[]>('GET', '/vms')
  }

  getVM(name: string) {
    return this.request<VMInfo>('GET', `/vms/${name}`)
  }

  createVM(data: CreateVMRequest) {
    return this.request<{ status: string; name: string }>('POST', '/vms', data)
  }

  deleteVM(name: string) {
    return this.request<{ status: string }>('DELETE', `/vms/${name}`)
  }

  startVM(name: string) {
    return this.request<{ status: string }>('POST', `/vms/${name}/state`, {
      action: 'start',
    })
  }

  stopVM(name: string) {
    return this.request<{ status: string }>('POST', `/vms/${name}/state`, {
      action: 'stop',
    })
  }

  restartVM(name: string) {
    return this.request<{ status: string }>('POST', `/vms/${name}/state`, {
      action: 'restart',
    })
  }

  cloneVM(name: string, newName: string) {
    return this.request<{ status: string; name: string }>(
      'POST',
      `/vms/${name}/clone`,
      { new_name: newName }
    )
  }

  // Snapshots
  listSnapshots(vmName: string) {
    return this.request<Record<string, Snapshot>>('GET', `/vms/${vmName}/snapshots`)
  }

  createSnapshot(vmName: string, snapshotName?: string) {
    return this.request<{ status: string }>(
      'POST',
      `/vms/${vmName}/snapshots`,
      snapshotName ? { name: snapshotName } : {}
    )
  }

  restoreSnapshot(vmName: string, snapshotName: string) {
    return this.request<{ status: string }>(
      'POST',
      `/vms/${vmName}/snapshots/restore`,
      { snapshot_name: snapshotName }
    )
  }

  deleteSnapshot(vmName: string, snapshotName: string) {
    return this.request<{ status: string }>(
      'DELETE',
      `/vms/${vmName}/snapshots/${snapshotName}`
    )
  }

  // Files
  listFiles(vmName: string, path: string) {
    return this.request<FilesResponse>(
      'GET',
      `/vms/${vmName}/files?path=${encodeURIComponent(path)}`
    )
  }

  async uploadFile(vmName: string, path: string, file: File): Promise<void> {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch(
      `${API_BASE}/vms/${vmName}/files?path=${encodeURIComponent(path)}`,
      {
        method: 'POST',
        credentials: 'include',
        body: formData,
      }
    )
    if (!res.ok) {
      const text = await res.text()
      let message = text
      try {
        const json = JSON.parse(text)
        message = json.error || text
      } catch {
        // Use raw text
      }
      throw new Error(message || res.statusText)
    }
  }

  async downloadFile(vmName: string, path: string): Promise<Blob> {
    const res = await fetch(
      `${API_BASE}/vms/${vmName}/files/download?path=${encodeURIComponent(path)}`,
      {
        credentials: 'include',
      }
    )
    if (!res.ok) {
      const text = await res.text()
      let message = text
      try {
        const json = JSON.parse(text)
        message = json.error || text
      } catch {
        // Use raw text
      }
      throw new Error(message || res.statusText)
    }
    return res.blob()
  }

  // Mounts
  listMounts(vmName: string) {
    return this.request<MountEntry[]>('GET', `/vms/${vmName}/mounts`)
  }

  addMount(vmName: string, hostPath: string, vmPath: string) {
    return this.request<{ status: string }>('POST', `/vms/${vmName}/mounts`, {
      host_path: hostPath,
      vm_path: vmPath,
    })
  }

  removeMount(vmName: string, vmPath: string) {
    return this.request<{ status: string }>(
      'DELETE',
      `/vms/${vmName}/mounts?path=${encodeURIComponent(vmPath)}`
    )
  }

  // Tunnels
  listTunnels() {
    return this.request<TunnelInfo[]>('GET', '/tunnels')
  }

  createTunnel(vmName: string, vmPort: number) {
    return this.request<TunnelInfo>('POST', '/tunnels', {
      vm_name: vmName,
      vm_port: vmPort,
    })
  }

  deleteTunnel(hostPort: number) {
    return this.request<{ status: string }>('DELETE', `/tunnels/${hostPort}`)
  }

  // Network
  getNetworkConfig(vmName: string) {
    return this.request<NetworkConfig>('GET', `/vms/${vmName}/network`)
  }

  updateNetworkConfig(vmName: string, config: NetworkConfig) {
    return this.request<{ status: string; mode: string }>(
      'PUT',
      `/vms/${vmName}/network`,
      config
    )
  }

  removeNetworkConfig(vmName: string) {
    return this.request<{ status: string }>('DELETE', `/vms/${vmName}/network`)
  }

  applyNetworkConfig(vmName: string) {
    return this.request<{ status: string; mode: string }>(
      'POST',
      `/vms/${vmName}/network/apply`
    )
  }

  getNetworkDefaults() {
    return this.request<NetworkConfig>('GET', '/network/defaults')
  }

  setNetworkDefaults(config: NetworkConfig) {
    return this.request<{ status: string; mode: string }>(
      'PUT',
      '/network/defaults',
      config
    )
  }

  // Agent
  getAgentURL(vmName: string) {
    return this.request<{ url: string }>('GET', `/vms/${vmName}/agent-url`)
  }
}

export const api = new APIClient()

// Types
export interface VM {
  name: string
  state: string
  ipv4: string[]
  release: string
}

export interface VMInfo {
  cpu_count: string
  disks: Record<string, { total: string; used: string }>
  image_hash: string
  image_release: string
  ipv4: string[]
  load: number[]
  memory: { total: number; used: number }
  mounts: Record<string, { source_path: string }>
  release: string
  snapshot_count: string
  state: string
}

export interface CreateVMRequest {
  name: string
  cpu?: number
  mem?: string
  disk?: string
  image?: string
  network?: NetworkConfig
}

// Network types
export type NetworkMode = 'none' | 'allowlist' | 'blocklist' | 'isolated'

export interface NetworkRule {
  type: 'ip' | 'cidr' | 'domain'
  value: string
  comment?: string
}

export interface NetworkConfig {
  mode: NetworkMode
  rules?: NetworkRule[]
}

export interface Snapshot {
  comment: string
  parent: string
}

export interface FileEntry {
  name: string
  is_dir: boolean
  size: number
  mode: string
}

export interface FilesResponse {
  path: string
  entries: FileEntry[]
}

export interface MountEntry {
  host_path: string
  vm_path: string
}

export interface TunnelInfo {
  host_port: number
  vm_name: string
  vm_port: number
}

export interface VMDefaults {
  cpu: number
  mem: string
  disk: string
}
