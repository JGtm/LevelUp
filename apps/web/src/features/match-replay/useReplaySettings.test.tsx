/**
 * useReplaySettings.test.tsx — les calques et la vitesse survivent à la page (même règle
 * de persistance que le son), et démarrent sur les valeurs par défaut d'aujourd'hui quand
 * rien n'a encore été choisi.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SPEED_MULTIPLIERS, useReplaySettings } from './useReplaySettings'

describe('useReplaySettings — valeurs par défaut', () => {
  it("visée, zones et noms allumés, vitesse à 1x — comportement inchangé sans préférence stockée", () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(true)
    expect(result.current.showZones).toBe(true)
    expect(result.current.showNames).toBe(true)
    expect(result.current.speed).toBe(1)
  })

  it('carte de chaleur ÉTEINTE, en lecture présence — le rejeu s ouvre sur ce qui bouge', () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showHeatmap).toBe(false)
    expect(result.current.heatmapMode).toBe('presence')
  })

  it('effets de tirs ALLUMÉS, effets de mort ÉTEINTS — les deux défauts du 16/08', () => {
    // Ce ne sont pas deux réglages symétriques : l'éclair de bouche dit où le match se joue
    // et il a été validé sans réserve ; le trait tueur -> victime, lui, est « optionnel,
    // désactivé par défaut » — c'est la décision produit, elle se teste.
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showShotFx).toBe(true)
    expect(result.current.showKillFx).toBe(false)
  })

  it('emplacements d arme ALLUMÉS — « les infos sont intéressantes à avoir » (18/08)', () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showWeaponPads).toBe(true)
  })
})

describe('useReplaySettings — bascules', () => {
  it('toggleAim et toggleZones inversent leur propre calque, jamais l autre', () => {
    const { result } = renderHook(() => useReplaySettings())
    act(() => result.current.toggleAim())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(true)
    act(() => result.current.toggleZones())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(false)
  })

  it('setSpeed accepte un multiplicateur de la liste proposée', () => {
    const { result } = renderHook(() => useReplaySettings())
    act(() => result.current.setSpeed(4))
    expect(result.current.speed).toBe(4)
    expect(SPEED_MULTIPLIERS).toContain(4)
  })
})

describe('useReplaySettings — préférences persistées (localStorage, comme le son)', () => {
  it('survivent au remontage', () => {
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleAim())
    act(() => first.result.current.toggleZones())
    act(() => first.result.current.toggleNames())
    act(() => first.result.current.setSpeed(2))
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(false)
    expect(result.current.showNames).toBe(false)
    expect(result.current.speed).toBe(2)
  })

  it('une vitesse hors liste stockée par un autre moyen retombe sur 1x, jamais une valeur inventée', () => {
    localStorage.setItem('replay-speed', '3')
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.speed).toBe(1)
  })

  it('la carte de chaleur et sa lecture survivent au remontage', () => {
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleHeatmap())
    act(() => first.result.current.setHeatmapMode('kills'))
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showHeatmap).toBe(true)
    expect(result.current.heatmapMode).toBe('kills')
  })

  it('une lecture inconnue stockée par un autre moyen retombe sur la présence', () => {
    localStorage.setItem('replay-heatmap-mode', 'temperature')
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.heatmapMode).toBe('presence')
  })

  it('les deux effets survivent au remontage, chacun sur SA clé', () => {
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleShotFx())
    act(() => first.result.current.toggleKillFx())
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showShotFx).toBe(false)
    expect(result.current.showKillFx).toBe(true)
  })

  it('les emplacements d arme survivent au remontage, sur LEUR clé', () => {
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleWeaponPads())
    expect(first.result.current.showWeaponPads).toBe(false)
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showWeaponPads).toBe(false)
    // Les poses, elles, n'ont pas bougé : deux clés distinctes, jamais une pour deux.
    expect(result.current.showPlacements).toBe(true)
  })
})
