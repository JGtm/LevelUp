/**
 * Tests — useReplayViewpoint (l'ÉTAT « par les yeux de qui regarde-t-on ? »).
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-07, revue F5). La RÈGLE de résolution est pure et déjà
 * couverte (`model/replayViewpoint.test.ts`) ; la COUTURE React, elle, ne l'était par rien —
 * aucun test ne montait ce hook. Or c'est elle qui porte les trois affirmations du produit :
 * la sélection se prend, elle se rend, et elle se DÉFAIT (retour au joueur de la page). Une
 * inversion ici — un `select` qui n'écrit pas, un `null` qui ne remet pas le défaut — laisserait
 * le menu de la frise sans effet, ou bloqué sur un joueur, sans qu'aucune suite ne rougisse.
 *
 * CE QU'ILS NE TESTENT PAS : le temps. Ce hook ne connaît ni la lecture, ni le curseur
 * (décision 1 du plan « frise, point de vue ») — il n'y a rien à vérifier de ce côté, il n'en
 * reçoit rien.
 */
import { describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'

import { useReplayViewpoint } from './useReplayViewpoint'
import type { ViewpointRow } from '../model/replayViewpoint'

/** Le lobby témoin : le joueur de la page, un coéquipier, un adversaire. */
const SB: ViewpointRow[] = [
  { xuid: 'moi', is_me: true },
  { xuid: 'pote', is_me: false },
  { xuid: 'eux', is_me: false },
]

/**
 * Le tableau de score est EXPLICITE À CHAQUE APPEL, sans valeur par défaut : `monter(undefined)`
 * doit vraiment monter le hook sans tableau — un défaut de paramètre le remplacerait par le
 * lobby et le cas « la vue match n'est pas encore là » ne serait jamais éprouvé.
 */
function monter(scoreboard: readonly ViewpointRow[] | null | undefined) {
  return renderHook(({ sb }: { sb: typeof scoreboard }) => useReplayViewpoint(sb), {
    initialProps: { sb: scoreboard },
  })
}

describe('useReplayViewpoint — le défaut, la sélection, le retour', () => {
  it('au montage : le joueur de la page, jamais une sélection retenue d’ailleurs', () => {
    expect(monter(SB).result.current.xuid).toBe('moi')
  })

  it('`select(x)` avec `x` au tableau de score : c’est lui qu’on regarde', () => {
    const { result } = monter(SB)
    act(() => result.current.select('eux'))
    expect(result.current.xuid).toBe('eux')
  })

  it('`select(null)` : retour au joueur de la page', () => {
    const { result } = monter(SB)
    act(() => result.current.select('eux'))
    act(() => result.current.select(null))
    expect(result.current.xuid).toBe('moi')
  })

  it('`select` d’un xuid ABSENT du tableau : on retombe sur le joueur de la page', () => {
    // Le cas d'un tableau rechargé sous une sélection survivante. Sans ce repli, le point de
    // vue vaudrait un joueur inexistant : aucun camp allié, aucune marque, aucun écran de fin.
    const { result } = monter(SB)
    act(() => result.current.select('xuid-jamais-vu'))
    expect(result.current.xuid).toBe('moi')
  })

  it('tableau de score pas encore là : `null`, et rien ne casse', () => {
    expect(monter(undefined).result.current.xuid).toBeNull()
    expect(monter(null).result.current.xuid).toBeNull()
  })

  it('la sélection SURVIT à l’arrivée du tableau de score', () => {
    // La vue match arrive après le document du rejeu : une sélection prise trop tôt (menu
    // ouvert dès le premier rendu) ne doit pas se perdre au re-rendu suivant.
    const view = monter(undefined)
    act(() => view.result.current.select('eux'))
    expect(view.result.current.xuid).toBeNull()
    view.rerender({ sb: SB })
    expect(view.result.current.xuid).toBe('eux')
  })

  it('AUCUNE PERSISTANCE (décision 8) : un remontage repart du joueur de la page', () => {
    const premier = monter(SB)
    act(() => premier.result.current.select('eux'))
    premier.unmount()
    expect(monter(SB).result.current.xuid).toBe('moi')
    expect(localStorage.length).toBe(0)
  })

  it('`select` est stable d’un rendu à l’autre : le menu ne se re-rend pas pour rien', () => {
    const view = monter(SB)
    const avant = view.result.current.select
    view.rerender({ sb: [...SB] })
    expect(view.result.current.select).toBe(avant)
  })
})
