/**
 * useReplaySettings.test.tsx — les calques et la vitesse survivent à la page (même règle
 * de persistance que le son), et démarrent sur les valeurs par défaut d'aujourd'hui quand
 * rien n'a encore été choisi.
 */
import { StrictMode } from 'react'
import { act, fireEvent, render, renderHook, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SPEED_MULTIPLIERS, useReplayCompactCards, useReplaySettings } from './useReplaySettings'

describe('useReplaySettings — valeurs par défaut', () => {
  it("visée, zones et noms allumés, vitesse à 1x — comportement inchangé sans préférence stockée", () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(true)
    expect(result.current.showZones).toBe(true)
    expect(result.current.showNames).toBe(true)
    // V1 : la traînée est une OPTION depuis le 18/08, mais elle reste ALLUMÉE par défaut —
    // le marqueur validé le 16/08 la comptait dans son « Parfait ».
    expect(result.current.showTrail).toBe(true)
    expect(result.current.speed).toBe(1)
  })

  it('carte de chaleur ÉTEINTE, en lecture présence — le rejeu s ouvre sur ce qui bouge', () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showHeatmap).toBe(false)
    expect(result.current.heatmapMode).toBe('presence')
  })

  it('les DEUX effets d événement sont ALLUMÉS par défaut', () => {
    // Les effets de mort ont rejoint les éclairs de bouche le 2026-08-20. Éteints, ils
    // demandaient de savoir qu'ils existaient pour aller les allumer : l'utilisateur les
    // cherchait sans les trouver. C'est la décision produit, elle se teste.
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showShotFx).toBe(true)
    expect(result.current.showKillFx).toBe(true)
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

  /**
   * L'ALLER-RETOUR, et pas seulement l'aller. Les tests n'épinglaient que ÉTEINT -> ALLUMÉ ;
   * le bug vécu était au RETOUR — la bascule « revenait en arrière » parce que la
   * persistance, appelée depuis l'updater de `setValue`, notifiait les abonnés en pleine
   * phase de rendu. Un aller simple ne l'aurait jamais vu.
   */
  it('toggleHeatmap ramène le calque à son état de départ au second appel', () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showHeatmap).toBe(false)
    act(() => result.current.toggleHeatmap())
    expect(result.current.showHeatmap).toBe(true)
    act(() => result.current.toggleHeatmap())
    expect(result.current.showHeatmap).toBe(false)
    // Et le stockage suit la valeur affichée : sinon le retour se perdrait au rechargement.
    expect(localStorage.getItem('replay-show-heatmap')).toBe('false')
  })
})

describe('useReplaySettings — préférences persistées (localStorage, comme le son)', () => {
  it('survivent au remontage', () => {
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleAim())
    act(() => first.result.current.toggleZones())
    act(() => first.result.current.toggleNames())
    act(() => first.result.current.toggleTrail())
    act(() => first.result.current.setSpeed(2))
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(false)
    expect(result.current.showNames).toBe(false)
    expect(result.current.showTrail).toBe(false)
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
    // Les deux partent ALLUMÉS : une bascule chacun les éteint, et c'est cet état-là qui doit
    // survivre. Deux clés distinctes, jamais une pour deux.
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleShotFx())
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

/**
 * B2/R2-7 — LES FICHES COMPACTES SONT UNE OPTION, ET ELLE EST PARTAGÉE.
 *
 * La bascule vit dans le tiroir (sous le canvas), les fiches vivent dans une autre colonne de
 * la page : deux `useState` initialisés du même stockage ne se parleraient pas. C'est
 * l'abonnement de `usePersistedFlag` qui les tient ensemble, et c'est ce que ce test épingle
 * — sans lui, la bascule bougerait sans que les fiches changent.
 */
describe('useReplaySettings — fiches compactes', () => {
  it('ÉTEINTES par défaut : la fiche validée reste le défaut', () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.compactCards).toBe(false)
  })

  it('la bascule du tiroir change AUSSI ce que lit la colonne des fiches', () => {
    const tiroir = renderHook(() => useReplaySettings())
    const fiches = renderHook(() => useReplayCompactCards())
    expect(fiches.result.current).toBe(false)
    act(() => tiroir.result.current.toggleCompactCards())
    expect(tiroir.result.current.compactCards).toBe(true)
    expect(fiches.result.current).toBe(true)
  })

  /**
   * LE RETOUR, sur les DEUX instances à la fois — c'est précisément là que le bug vécu se
   * manifestait. `compactCards` est la seule clé lue par deux hooks distincts (le tiroir dans
   * ReplayCanvas, les fiches dans la page, qui en est l'ANCÊTRE) : persister depuis l'updater
   * de `setValue` déclenchait la notification des abonnés en pleine phase de rendu, et une
   * des deux instances gardait l'ancienne valeur. Le second appel doit ramener les DEUX à
   * l'état de départ, et le stockage avec.
   */
  it('un aller-retour ramène le tiroir ET les fiches à leur état de départ', () => {
    const tiroir = renderHook(() => useReplaySettings())
    const fiches = renderHook(() => useReplayCompactCards())
    act(() => tiroir.result.current.toggleCompactCards())
    expect(tiroir.result.current.compactCards).toBe(true)
    expect(fiches.result.current).toBe(true)
    act(() => tiroir.result.current.toggleCompactCards())
    expect(tiroir.result.current.compactCards).toBe(false)
    expect(fiches.result.current).toBe(false)
    expect(localStorage.getItem('replay-compact-cards')).toBe('false')
  })

  it('elle survit au rechargement de la page', () => {
    const premier = renderHook(() => useReplaySettings())
    act(() => premier.result.current.toggleCompactCards())
    premier.unmount()
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.compactCards).toBe(true)
  })
})

/**
 * LE BUG DU « RETOUR ARRIÈRE », DANS SA VRAIE CONFIGURATION.
 *
 * Deux `renderHook` côte à côte ne le reproduisent PAS : ce sont deux racines React
 * indépendantes, et la notification ré-entrante n'y traverse jamais une frontière
 * ancêtre/descendant. Or c'est exactement ce que fait la page : `useReplayCompactCards` vit
 * dans la route (ANCÊTRE), la bascule vit dans le tiroir de `ReplayCanvas` (DESCENDANT), sur
 * la MÊME clé. Persister depuis l'updater de `setValue` faisait alors notifier les abonnés en
 * pleine phase de rendu du descendant — donc appeler `setValue` de l'ancêtre pendant que le
 * descendant rend, ce que React refuse d'appliquer tel quel.
 *
 * Ce test monte donc l'arbre RÉEL, et sous `StrictMode` — qui invoque les updaters deux fois
 * pour débusquer précisément les updaters impurs. Il ne teste rien de plus que ce que
 * l'utilisateur fait : cliquer deux fois et retrouver son état de départ.
 */
describe('useReplaySettings — la bascule partagée revient bien en arrière (arbre réel)', () => {
  function Tiroir() {
    const { compactCards, toggleCompactCards } = useReplaySettings()
    return (
      <button type="button" data-testid="bascule" onClick={toggleCompactCards}>
        {String(compactCards)}
      </button>
    )
  }

  /** La route : elle LIT la préférence, et rend le tiroir plus bas dans l'arbre. */
  function Page() {
    const compact = useReplayCompactCards()
    return (
      <div>
        <span data-testid="fiches">{String(compact)}</span>
        <Tiroir />
      </div>
    )
  }

  it('deux clics ramènent le tiroir ET les fiches à ÉTEINT, sous StrictMode', () => {
    render(
      <StrictMode>
        <Page />
      </StrictMode>,
    )
    expect(screen.getByTestId('fiches')).toHaveTextContent('false')
    expect(screen.getByTestId('bascule')).toHaveTextContent('false')

    fireEvent.click(screen.getByTestId('bascule'))
    expect(screen.getByTestId('fiches')).toHaveTextContent('true')
    expect(screen.getByTestId('bascule')).toHaveTextContent('true')

    fireEvent.click(screen.getByTestId('bascule'))
    expect(screen.getByTestId('fiches')).toHaveTextContent('false')
    expect(screen.getByTestId('bascule')).toHaveTextContent('false')
    expect(localStorage.getItem('replay-compact-cards')).toBe('false')
  })
})
