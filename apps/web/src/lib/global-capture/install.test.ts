import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest'
import {
  _resetInstallFlagForTests,
  _uninstallGlobalCaptureForTests,
  installGlobalCapture,
} from './install'
import {
  getRecentConsoleEntries,
  getRecentFailedRequests,
  resetCaptureBuffersForTests,
} from './buffers'

beforeEach(() => {
  _resetInstallFlagForTests()
  resetCaptureBuffersForTests()
})

afterEach(() => {
  // Critique : restaure console.error/warn, retire les listeners window
  // et le wrap fetch. Sans ce cleanup, le worker jsdom accumule les
  // listeners entre fichiers de tests, ralentissant les suites lourdes.
  _uninstallGlobalCaptureForTests()
})

describe('installGlobalCapture — idempotence', () => {
  let originalConsoleError: typeof console.error
  let originalFetch: typeof window.fetch

  beforeEach(() => {
    originalConsoleError = console.error
    originalFetch = window.fetch
  })

  afterEach(() => {
    console.error = originalConsoleError
    window.fetch = originalFetch
  })

  it('install une seule fois, ré-appel = no-op', () => {
    installGlobalCapture()
    const wrappedOnce = console.error
    installGlobalCapture()
    const wrappedTwice = console.error
    expect(wrappedTwice).toBe(wrappedOnce)
  })

  it('console.error capturé n\'apparait qu\'une fois dans le buffer', () => {
    installGlobalCapture()
    installGlobalCapture()
    console.error('test message')
    expect(getRecentConsoleEntries()).toHaveLength(1)
  })

  it('le flag globalThis survit (simule HMR)', () => {
    installGlobalCapture()
    expect(
      (globalThis as Record<string, unknown>)['__levelup_global_capture_installed__'],
    ).toBe(true)
  })
})

describe('installGlobalCapture — wrap console', () => {
  let originalError: typeof console.error
  let originalWarn: typeof console.warn

  beforeEach(() => {
    originalError = console.error
    originalWarn = console.warn
  })

  afterEach(() => {
    console.error = originalError
    console.warn = originalWarn
  })

  it("appelle toujours l'original console.error", () => {
    const spy = vi.fn()
    console.error = spy
    installGlobalCapture()
    console.error('msg', 42)
    expect(spy).toHaveBeenCalledWith('msg', 42)
  })

  it('capture le stack si une Error est passée en argument', () => {
    installGlobalCapture()
    const err = new Error('boom')
    console.error('context:', err)
    const [entry] = getRecentConsoleEntries()
    expect(entry?.stack).toBe(err.stack)
    expect(entry?.message).toBe('context: boom')
  })

  it("capture warn comme un niveau distinct", () => {
    installGlobalCapture()
    console.warn('caution')
    const [entry] = getRecentConsoleEntries()
    expect(entry?.level).toBe('warn')
  })
})

describe('_uninstallGlobalCaptureForTests', () => {
  it("restaure console.error original", () => {
    const beforeInstall = console.error
    installGlobalCapture()
    expect(console.error).not.toBe(beforeInstall)
    _uninstallGlobalCaptureForTests()
    expect(console.error).toBe(beforeInstall)
  })

  it("restaure window.fetch original", () => {
    const beforeInstall = window.fetch
    installGlobalCapture()
    expect(window.fetch).not.toBe(beforeInstall)
    _uninstallGlobalCaptureForTests()
    expect(window.fetch).toBe(beforeInstall)
  })

  it("retire les listeners window 'error' (events suivants ne capturent plus)", () => {
    installGlobalCapture()
    _uninstallGlobalCaptureForTests()
    resetCaptureBuffersForTests()
    window.dispatchEvent(new ErrorEvent('error', { message: 'apres-uninstall' }))
    expect(getRecentConsoleEntries().some((e) => e.message.includes('apres-uninstall'))).toBe(false)
  })

  it("permet une réinstallation propre après uninstall", () => {
    installGlobalCapture()
    _uninstallGlobalCaptureForTests()
    installGlobalCapture()
    expect(
      (globalThis as Record<string, unknown>)['__levelup_global_capture_installed__'],
    ).toBe(true)
  })
})

describe('installGlobalCapture — safeRun resilience', () => {
  let originalAddEventListener: typeof window.addEventListener
  let originalConsoleError: typeof console.error

  beforeEach(() => {
    originalAddEventListener = window.addEventListener
    originalConsoleError = console.error
  })

  afterEach(() => {
    window.addEventListener = originalAddEventListener
    console.error = originalConsoleError
  })

  it("si wrapWindowErrors throw, les autres wraps s'installent quand même + log capture:install_failed", () => {
    const errorSpy = vi.fn()
    console.error = errorSpy
    window.addEventListener = vi.fn(() => {
      throw new Error('addEventListener KO')
    }) as typeof window.addEventListener
    installGlobalCapture()
    // log d'erreur capté avec la clé stable
    const errCalls = errorSpy.mock.calls.map((c) => String(c[0]))
    expect(errCalls.some((c) => c.includes('capture:install_failed'))).toBe(true)
    // Mais console.error ET fetch sont quand même wrappés
    expect(console.error).not.toBe(originalConsoleError)
  })
})

describe('installGlobalCapture — wrap window errors', () => {
  it("capture les ErrorEvent dans le buffer", () => {
    installGlobalCapture()
    window.dispatchEvent(
      new ErrorEvent('error', { message: 'window-level boom', error: new Error('e') }),
    )
    const entries = getRecentConsoleEntries()
    expect(entries.some((e) => e.message.includes('window-level boom'))).toBe(true)
  })

  it("capture les unhandledrejection (reason: Error)", () => {
    installGlobalCapture()
    const event = new Event('unhandledrejection') as PromiseRejectionEvent
    Object.defineProperty(event, 'reason', { value: new Error('promise rejected') })
    window.dispatchEvent(event)
    const entries = getRecentConsoleEntries()
    expect(entries.some((e) => e.message.includes('promise rejected'))).toBe(true)
  })

  it("capture les unhandledrejection (reason: string)", () => {
    installGlobalCapture()
    const event = new Event('unhandledrejection') as PromiseRejectionEvent
    Object.defineProperty(event, 'reason', { value: 'string reason' })
    window.dispatchEvent(event)
    const entries = getRecentConsoleEntries()
    expect(entries.some((e) => e.message === 'string reason')).toBe(true)
  })
})

describe('installGlobalCapture — wrap fetch', () => {
  let originalFetch: typeof window.fetch

  beforeEach(() => {
    originalFetch = window.fetch
  })

  afterEach(() => {
    window.fetch = originalFetch
  })

  it("ne capture pas les réponses 2xx", async () => {
    window.fetch = vi.fn().mockResolvedValue(
      new Response('{"ok": true}', { status: 200 }),
    )
    installGlobalCapture()
    await window.fetch('/api/v1/foo')
    expect(getRecentFailedRequests()).toHaveLength(0)
  })

  it('capture les 4xx et 5xx', async () => {
    window.fetch = vi.fn().mockResolvedValue(new Response('', { status: 500 }))
    installGlobalCapture()
    await window.fetch('/api/v1/foo', { method: 'POST' })
    const [req] = getRecentFailedRequests()
    expect(req?.url).toBe('/api/v1/foo')
    expect(req?.method).toBe('POST')
    expect(req?.status).toBe(500)
  })

  it('capture les erreurs réseau (status=0) et propage le throw', async () => {
    window.fetch = vi.fn().mockRejectedValue(new Error('network'))
    installGlobalCapture()
    await expect(window.fetch('/api/v1/foo')).rejects.toThrow('network')
    const [req] = getRecentFailedRequests()
    expect(req?.status).toBe(0)
  })

  it('strippe la query string sensible avant stockage', async () => {
    window.fetch = vi.fn().mockResolvedValue(new Response('', { status: 500 }))
    installGlobalCapture()
    await window.fetch('/api/v1/players/Foo?token=secret')
    expect(getRecentFailedRequests()[0]?.url).toBe('/api/v1/players/Foo')
  })
})
