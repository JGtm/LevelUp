/**
 * useZoneStates.test.ts — LA STABILITÉ DE RÉFÉRENCE, parce que c'est elle qui coûte des images.
 *
 * L'objet rendu par ce hook entre dans les dépendances de `draw` (`ReplayCanvas`). Un littéral
 * neuf à chaque rendu recuit donc le `useCallback` du tracé — TOUTE la scène — à chaque
 * mouvement de pointeur, puisque `usePlacementHover` porte un `useState` qui fait rendre le
 * canvas. Ce fichier verrouille l'invariant : à entrées inchangées, MÊME référence.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ObjectiveElementReady } from './objectivesLayer'
import { useZoneStates } from './useZoneStates'

/** Un élément servi, dans la forme normalisée que le canvas passe au hook. */
function element(kind: 'zone' | 'marker', x: number): ObjectiveElementReady {
  return {
    role: 'strongholds_zone', team: -1, x, y: 0, z: 0, kind,
    halfX: 4, halfY: 4, radius: 4, fwd: { x: 1, y: 0 },
  }
}

const OBJECTIFS: ObjectiveElementReady[] = [element('zone', -20), element('marker', 0), element('zone', 20)]

/** Le tableau de bord réduit à ce que le hook lit : la ligne « moi » et son `team_side`. */
const TABLEAU = [{ is_me: true, team_side: 't1' }] as unknown as MatchScoreboardRow[]

const ENCRE = (isAlly: boolean) => (isAlly ? '#allie' : '#adverse')

describe('useZoneStates', () => {
  it('rend la MÊME référence quand rien ne change — un survol ne doit pas recuire le tracé', () => {
    const { result, rerender } = renderHook(
      (p: { objectifs: ObjectiveElementReady[] }) =>
        useZoneStates(p.objectifs, TABLEAU, ENCRE, '#neutre'),
      { initialProps: { objectifs: OBJECTIFS } },
    )
    const premier = result.current
    // Un rendu déclenché par un état LOCAL du canvas (le survol) ne change aucune entrée.
    rerender({ objectifs: OBJECTIFS })
    rerender({ objectifs: OBJECTIFS })
    expect(result.current).toBe(premier)
    expect(result.current.zoneElements).toBe(premier.zoneElements)
    expect(result.current.style).toBe(premier.style)
    expect(result.current.colorOfTeam).toBe(premier.colorOfTeam)
  })

  it('rend une NOUVELLE référence quand les objectifs servis changent', () => {
    const { result, rerender } = renderHook(
      (p: { objectifs: ObjectiveElementReady[] }) =>
        useZoneStates(p.objectifs, TABLEAU, ENCRE, '#neutre'),
      { initialProps: { objectifs: OBJECTIFS } },
    )
    const premier = result.current
    rerender({ objectifs: [element('zone', -20)] })
    expect(result.current).not.toBe(premier)
    expect(result.current.zoneElements).toHaveLength(1)
  })

  it("ne garde que les zones, dans l'ordre servi — c'est ce que `zoneRef` indexe", () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre'))
    expect(result.current.zoneElements.map((e) => e.x)).toEqual([-20, 20])
  })

  it('sans ligne « moi », aucun camp n\'est allié : l\'encre du propriétaire reste inconnue', () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, null, ENCRE, '#neutre'))
    expect(result.current.style.colorOfOwner(1)).toBeNull()
  })

  it('avec la ligne « moi », le camp du tableau de bord est l\'allié', () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre'))
    expect(result.current.style.colorOfOwner(1)).toBe('#allie')
    expect(result.current.style.colorOfOwner(0)).toBe('#adverse')
  })
})
