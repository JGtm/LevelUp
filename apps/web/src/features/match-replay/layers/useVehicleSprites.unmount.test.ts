import { renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useVehicleSprites } from './useVehicleSprites'

// Le manifeste des véhicules se lit par une requête qui peut échouer APRÈS le démontage du rejeu
// (page quittée, test terminé). Un échec tardif ne doit ni journaliser ni redessiner.
describe('useVehicleSprites : manifeste en échec après le démontage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  function echecDiffere() {
    let rejeter: (raison: unknown) => void = () => {}
    const attente = new Promise<Response>((_, reject) => {
      rejeter = reject
    })
    vi.stubGlobal('fetch', () => attente)
    return { rejeter: () => rejeter(new TypeError('réseau coupé')), attente }
  }

  it('démonté avant l’échec : aucun avertissement, aucun redessin', async () => {
    const { rejeter, attente } = echecDiffere()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const redraw = vi.fn()

    const { unmount } = renderHook(() =>
      useVehicleSprites({ enabled: true, labels: {}, titleSlug: 'halo_infinite', redraw }),
    )
    unmount()
    rejeter()
    await attente.catch(() => {})
    await Promise.resolve()

    expect(warn).not.toHaveBeenCalled()
    expect(redraw).not.toHaveBeenCalled()
  })

  it('encore monté à l’échec : l’avertissement part et la scène se redessine', async () => {
    const { rejeter, attente } = echecDiffere()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const redraw = vi.fn()

    renderHook(() => useVehicleSprites({ enabled: true, labels: {}, titleSlug: 'halo_infinite', redraw }))
    rejeter()
    await attente.catch(() => {})
    await vi.waitFor(() => expect(warn).toHaveBeenCalledTimes(1))

    expect(redraw).toHaveBeenCalled()
  })
})
