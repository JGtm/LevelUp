/**
 * useServerDraft — la copie éditable d'un objet servi.
 *
 * Le cas qui compte : l'objet est DÉJÀ en cache au montage (la coquille lit les réglages avant
 * la page). La copie doit alors porter la valeur servie dès le premier rendu, pas un objet vide
 * qui laisse chaque contrôle sur son défaut.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { useServerDraft } from './useServerDraft'

interface Reglages {
  delai: number
  coach: boolean
}

describe('useServerDraft', () => {
  it('objet déjà servi au montage : la copie porte la valeur servie', () => {
    const servi: Reglages = { delai: 45, coach: false }
    const { result } = renderHook(() => useServerDraft(servi))
    expect(result.current[0]).toEqual({ delai: 45, coach: false })
  })

  it('rien de servi : copie vide, puis réalignée à l’arrivée de l’objet', () => {
    const { result, rerender } = renderHook(({ servi }) => useServerDraft<Reglages>(servi), {
      initialProps: { servi: undefined as Reglages | undefined },
    })
    expect(result.current[0]).toEqual({})
    rerender({ servi: { delai: 30, coach: true } })
    expect(result.current[0]).toEqual({ delai: 30, coach: true })
  })

  it('une édition locale reste en place jusqu’au prochain objet servi, qui la remplace', () => {
    const premier: Reglages = { delai: 45, coach: false }
    const { result, rerender } = renderHook(({ servi }) => useServerDraft(servi), {
      initialProps: { servi: premier },
    })
    act(() => result.current[1]((d) => ({ ...d, delai: 60 })))
    rerender({ servi: premier })
    expect(result.current[0].delai).toBe(60)
    rerender({ servi: { delai: 90, coach: false } })
    expect(result.current[0].delai).toBe(90)
  })
})
