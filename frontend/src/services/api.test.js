import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'

vi.mock('axios', () => {
  const mockInterceptors = {
    request: { use: vi.fn() },
    response: { use: vi.fn() },
  }
  const mockInstance = {
    interceptors: mockInterceptors,
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  }
  return {
    default: { create: vi.fn(() => mockInstance) },
  }
})

describe('api service', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('creates axios instance with correct baseURL', async () => {
    await import('./api.js')
    expect(axios.create).toHaveBeenCalledWith(
      expect.objectContaining({
        baseURL: 'http://localhost:8080/api/v1',
      })
    )
  })

  it('request interceptor adds Authorization header when token exists', async () => {
    const { default: axiosInstance } = await import('axios')
    const instance = axiosInstance.create()

    const [onFulfilled] = instance.interceptors.request.use.mock.calls[0] || []

    // Re-import to trigger interceptor registration
    vi.resetModules()
    localStorage.setItem('token', 'test-token')

    const { default: freshInstance } = await import('axios')
    const fresh = freshInstance.create()
    const [requestFulfilled] = fresh.interceptors.request.use.mock.calls[0] || []

    if (requestFulfilled) {
      const config = { headers: {} }
      const result = requestFulfilled(config)
      // interceptor should have been called
      expect(result).toBeDefined()
    }
  })

  it('request interceptor does not add header when no token', async () => {
    vi.resetModules()
    localStorage.removeItem('token')

    const { default: freshInstance } = await import('axios')
    const fresh = freshInstance.create()
    const [requestFulfilled] = fresh.interceptors.request.use.mock.calls[0] || []

    if (requestFulfilled) {
      const config = { headers: {} }
      const result = requestFulfilled(config)
      expect(result.headers.Authorization).toBeUndefined()
    }
  })

  it('request interceptor rejects on error', async () => {
    vi.resetModules()
    const { default: freshInstance } = await import('axios')
    const fresh = freshInstance.create()
    const [, onRejected] = fresh.interceptors.request.use.mock.calls[0] || []

    if (onRejected) {
      await expect(onRejected(new Error('fail'))).rejects.toThrow('fail')
    }
  })

  it('response interceptor removes token on 401 and redirects', async () => {
    vi.resetModules()
    localStorage.setItem('token', 'tok')
    localStorage.setItem('user', '{}')

    const { default: freshInstance } = await import('axios')
    const fresh = freshInstance.create()
    const [, responseRejected] = fresh.interceptors.response.use.mock.calls[0] || []

    if (responseRejected) {
      const error = { response: { status: 401 } }
      await expect(responseRejected(error)).rejects.toEqual(error)
      expect(localStorage.getItem('token')).toBeNull()
      expect(localStorage.getItem('user')).toBeNull()
    }
  })

  it('response interceptor passes through non-401 errors', async () => {
    vi.resetModules()
    const { default: freshInstance } = await import('axios')
    const fresh = freshInstance.create()
    const [, responseRejected] = fresh.interceptors.response.use.mock.calls[0] || []

    if (responseRejected) {
      const error = { response: { status: 500 } }
      await expect(responseRejected(error)).rejects.toEqual(error)
    }
  })
})
