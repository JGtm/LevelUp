/**
 * useZoneStates.test.ts — LA STABILITÉ DE RÉFÉRENCE, LA GARDE DE JOINTURE et LES ENCRES.
 *
 * L'objet rendu par ce hook entre dans les dépendances de `draw` (`ReplayCanvas`). Un littéral
 * neuf à chaque rendu recuit donc le `useCallback` du tracé — TOUTE la scène — à chaque
 * mouvement de pointeur, puisque `usePlacementHover` porte un `useState` qui fait rendre le
 * canvas. Ce fichier verrouille l'invariant : à entrées inchangées, MÊME référence.
 *
 * Il verrouille aussi ce que le hook REFUSE : `zoneRef` est un index figé à la cuisson de
 * l'artefact, la liste servie est reconstruite à la requête. Quand les deux ne s'accordent pas,
 * le calque vivant se tait plutôt que de teinter la mauvaise zone.
 *
 * Et depuis le schéma 18 : la TENUE de la jauge en direct est convertie ici, en frames de CE
 * document, et l'encre du camp QUI CAPTURE est celle du camp d'en face du propriétaire.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ObjectiveElementReady } from './objectivesLayer'
import { testReplayDoc } from './test/testDoc'
import { useZoneStates } from './useZoneStates'
import { zoneCatalogMatches } from './zoneStatesLayer'

/** Un élément servi, dans la forme normalisée que le canvas passe au hook. */
function element(kind: 'zone' | 'marker', x: number): ObjectiveElementReady {
  return {
    role: 'strongholds_zone', team: -1, x, y: 0, z: 0, kind,
    halfX: 4, halfY: 4, radius: 4, fwd: { x: 1, y: 0 },
  }
}

const OBJECTIFS: ObjectiveElementReady[] = [element('zone', -20), element('marker', 0), element('zone', 20)]

/**
 * Le document tel que le hook le lit : le catalogue que l'artefact avait sous les yeux (deux
 * zones, comme la liste servie) et sa cadence (100 ms par frame).
 */
const docWith = (catalog: number | undefined) =>
  testReplayDoc({
    frameIntervalMs: 100,
    coverage: catalog === undefined ? undefined : { zones: { catalog } },
  } as never)
const DOC = docWith(2)

/** Le tableau de bord réduit à ce que le hook lit : la ligne « moi » et son `team_side`. */
const TABLEAU = [{ is_me: true, team_side: 't1' }] as unknown as MatchScoreboardRow[]

const ENCRE = (isAlly: boolean) => (isAlly ? '#allie' : '#adverse')

describe('useZoneStates', () => {
  it('rend la MÊME référence quand rien ne change — un survol ne doit pas recuire le tracé', () => {
    const { result, rerender } = renderHook(
      (p: { objectifs: ObjectiveElementReady[] }) =>
        useZoneStates(p.objectifs, TABLEAU, ENCRE, '#neutre', DOC),
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
        useZoneStates(p.objectifs, TABLEAU, ENCRE, '#neutre', DOC),
      { initialProps: { objectifs: OBJECTIFS } },
    )
    const premier = result.current
    rerender({ objectifs: [element('zone', -20)] })
    expect(result.current).not.toBe(premier)
    expect(result.current.zoneElements).toHaveLength(1)
  })

  it("ne garde que les zones, dans l'ordre servi — c'est ce que `zoneRef` indexe", () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', DOC))
    expect(result.current.zoneElements.map((e) => e.x)).toEqual([-20, 20])
    expect(result.current.joinable).toBe(true)
  })

  it("sans ligne « moi », aucun camp n'est allié : les encres du propriétaire et du capteur restent inconnues", () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, null, ENCRE, '#neutre', DOC))
    expect(result.current.style.colorOfOwner(1)).toBeNull()
    expect(result.current.style.colorOfCapturer(1)).toBeNull()
  })

  it("avec la ligne « moi », le camp du tableau de bord est l'allié", () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', DOC))
    expect(result.current.style.colorOfOwner(1)).toBe('#allie')
    expect(result.current.style.colorOfOwner(0)).toBe('#adverse')
  })

  it("le camp QUI CAPTURE une zone tenue est le camp d'en face du propriétaire", () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', DOC))
    // Ma zone (camp 1) se fait capturer : l'arc est ADVERSE ; leur zone (camp 0) : l'arc est ALLIÉ.
    expect(result.current.style.colorOfCapturer(1)).toBe('#adverse')
    expect(result.current.style.colorOfCapturer(0)).toBe('#allie')
  })

  it('la tenue de la jauge en direct est UNE seconde, en frames de ce document', () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', DOC))
    expect(result.current.gaugeHoldFrames).toBe(10)
    const lent = renderHook(() =>
      useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', testReplayDoc({ frameIntervalMs: 250 })),
    )
    expect(lent.result.current.gaugeHoldFrames).toBe(4)
  })

  // VERROU DE LA REVUE R1 : le catalogue a bougé depuis la cuisson (une zone ajoutée au
  // catalogue de formes, ou un rôle de plus dans la table du titre). `zoneRef` ne désigne plus
  // la même zone : le calque vivant ne peint RIEN.
  it('catalogue de l\'artefact différent de la liste servie : la jointure est refusée', () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', docWith(3)))
    expect(result.current.joinable).toBe(false)
  })

  it('couverture absente : « pas vérifiable » se traite comme « pas joignable »', () => {
    const { result } = renderHook(() => useZoneStates(OBJECTIFS, TABLEAU, ENCRE, '#neutre', docWith(undefined)))
    expect(result.current.joinable).toBe(false)
  })
})

describe('zoneCatalogMatches', () => {
  it('exige un catalogue publié ET de même taille que la liste servie', () => {
    expect(zoneCatalogMatches(3, 3)).toBe(true)
    expect(zoneCatalogMatches(0, 0)).toBe(true)
    expect(zoneCatalogMatches(2, 3)).toBe(false)
    expect(zoneCatalogMatches(4, 3)).toBe(false)
    expect(zoneCatalogMatches(undefined, 3)).toBe(false)
    expect(zoneCatalogMatches(null, 3)).toBe(false)
  })
})
