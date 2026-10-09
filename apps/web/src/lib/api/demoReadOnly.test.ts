/**
 * Refus « démo en lecture seule » : message localisé posé sur l'erreur, toast unique quand
 * la mutation ne gère pas son erreur, rien pour une autre erreur.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

const toastInfo = vi.fn()
vi.mock('sonner', () => ({ toast: { info: (...args: unknown[]) => toastInfo(...args) } }))

import { DEMO_TOAST_ID, handleDemoRefusal } from './demoReadOnly'

function demoError() {
  return { code: 'demo_mode_forbidden', message: 'This action is disabled in demo mode.', retryable: false, status: 403 }
}

describe('handleDemoRefusal', () => {
  beforeEach(() => toastInfo.mockReset())

  it('localise le message et affiche un toast dédupliqué sans gestion propre', () => {
    const err = demoError()
    expect(handleDemoRefusal(err, false, 'fr')).toBe(true)
    expect(err.message).toBe('Action indisponible dans la démo : elle est en lecture seule.')
    expect(toastInfo).toHaveBeenCalledWith(err.message, { id: DEMO_TOAST_ID })
  })

  it("laisse le toast à la mutation qui gère son erreur, message localisé en anglais", () => {
    const err = demoError()
    expect(handleDemoRefusal(err, true, 'en')).toBe(true)
    expect(err.message).toBe('This action is not available in the demo: it is read-only.')
    expect(toastInfo).not.toHaveBeenCalled()
  })

  it("ignore toute autre erreur", () => {
    const err = { code: 'player_forbidden', message: 'x', retryable: false, status: 403 }
    expect(handleDemoRefusal(err, false, 'fr')).toBe(false)
    expect(err.message).toBe('x')
    expect(toastInfo).not.toHaveBeenCalled()
  })
})
