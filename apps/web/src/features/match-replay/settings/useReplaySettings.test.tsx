/**
 * useReplaySettings.test.tsx — les calques et la vitesse survivent à la page (même règle
 * de persistance que le son), et démarrent sur les valeurs par défaut d'aujourd'hui quand
 * rien n'a encore été choisi.
 */
import { StrictMode } from 'react'
import { act, fireEvent, render, renderHook, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import {
  AUTOPLAY_KEY,
  SPEED_MULTIPLIERS,
  TIMELINE_EXPANDED_KEY,
  usePersistedFlag,
  useReplaySettings,
} from './useReplaySettings'

describe('useReplaySettings — valeurs par défaut', () => {
  it("visée et zones allumées, vitesse à 1x — comportement inchangé sans préférence stockée", () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(true)
    expect(result.current.showZones).toBe(true)
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

  // POINT 22 DU 2026-08-29 : la lecture automatique devient un réglage, ÉTEINT par défaut à la
  // demande de l'utilisateur. C'est un CHANGEMENT de comportement (le rejeu partait tout seul
  // depuis l'origine) : ce test est là pour qu'il ne se reperde pas silencieusement.
  it('lecture automatique ÉTEINTE par défaut — le rejeu s ouvre en pause', () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.autoPlay).toBe(false)
  })
})

describe('useReplaySettings — lecture automatique (point 22 du 2026-08-29)', () => {
  it('la bascule persiste le choix sous sa clé', () => {
    const { result } = renderHook(() => useReplaySettings())
    act(() => result.current.toggleAutoPlay())
    expect(result.current.autoPlay).toBe(true)
    expect(localStorage.getItem(AUTOPLAY_KEY)).toBe('true')
  })

  it('un choix déjà stocké est relu au montage', () => {
    localStorage.setItem(AUTOPLAY_KEY, 'true')
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.autoPlay).toBe(true)
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
    act(() => first.result.current.toggleTrail())
    act(() => first.result.current.setSpeed(2))
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(false)
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

describe('useReplaySettings — couleur des points', () => {
  it("'team' par défaut (la couleur dit le camp), et le choix se persiste", () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.markerColors).toBe('team')
    act(() => result.current.setMarkerColors('player'))
    expect(result.current.markerColors).toBe('player')
    expect(localStorage.getItem('replay-marker-colors')).toBe('player')
  })
})

/**
 * LE BUG DU « RETOUR ARRIÈRE », DANS SA VRAIE CONFIGURATION.
 *
 * Deux `renderHook` côte à côte ne le reproduisent PAS : ce sont deux racines React
 * indépendantes, et la notification ré-entrante n'y traverse jamais une frontière
 * ancêtre/descendant. Le bug a été vécu sur la clé des fiches compactes (supprimée le
 * 2026-08-24 avec son réglage) : une préférence lue par DEUX instances de hook, l'une chez
 * un ANCÊTRE, l'autre chez un DESCENDANT. Persister depuis l'updater de `setValue` faisait
 * notifier les abonnés en pleine phase de rendu du descendant — donc appeler `setValue` de
 * l'ancêtre pendant que le descendant rend, ce que React refuse d'appliquer tel quel.
 * L'INVARIANT PROTÉGÉ SURVIT À LA CLÉ : toute clé partagée entre deux instances y est
 * exposée — le test le rejoue sur `showAim`.
 *
 * Ce test monte donc l'arbre RÉEL, et sous `StrictMode` — qui invoque les updaters deux fois
 * pour débusquer précisément les updaters impurs. Il ne teste rien de plus que ce que
 * l'utilisateur fait : cliquer deux fois et retrouver son état de départ.
 */
describe('useReplaySettings — la bascule partagée revient bien en arrière (arbre réel)', () => {
  function Tiroir() {
    const { showAim, toggleAim } = useReplaySettings()
    return (
      <button type="button" data-testid="bascule" onClick={toggleAim}>
        {String(showAim)}
      </button>
    )
  }

  /** L'ancêtre : il LIT la même préférence, et rend le tiroir plus bas dans l'arbre. */
  function Page() {
    const { showAim } = useReplaySettings()
    return (
      <div>
        <span data-testid="lecteur">{String(showAim)}</span>
        <Tiroir />
      </div>
    )
  }

  it('deux clics ramènent le tiroir ET le lecteur à leur état de départ, sous StrictMode', () => {
    render(
      <StrictMode>
        <Page />
      </StrictMode>,
    )
    expect(screen.getByTestId('lecteur')).toHaveTextContent('true')
    expect(screen.getByTestId('bascule')).toHaveTextContent('true')

    fireEvent.click(screen.getByTestId('bascule'))
    expect(screen.getByTestId('lecteur')).toHaveTextContent('false')
    expect(screen.getByTestId('bascule')).toHaveTextContent('false')

    fireEvent.click(screen.getByTestId('bascule'))
    expect(screen.getByTestId('lecteur')).toHaveTextContent('true')
    expect(screen.getByTestId('bascule')).toHaveTextContent('true')
    expect(localStorage.getItem('replay-show-aim')).toBe('true')
  })
})

/**
 * LE REPLI DE LA FRISE (retour utilisateur du 2026-08-28) est une préférence persistée comme
 * les autres, mais elle ne passe PAS par le tiroir : `useReplayTimeline` lit directement le
 * helper. C'est la raison pour laquelle celui-ci est exporté — une seconde copie de son corps
 * aurait rouvert la divergence que sa centralisation avait fermée.
 */
describe('repli de la frise — la préférence survit à la page', () => {
  beforeEach(() => {
    localStorage.removeItem(TIMELINE_EXPANDED_KEY)
  })

  it('DÉPLIÉE par défaut : la frise à pistes est ce qu on vient de livrer, on ne la cache pas', () => {
    const { result } = renderHook(() => usePersistedFlag(TIMELINE_EXPANDED_KEY, true))
    expect(result.current[0]).toBe(true)
  })

  it('replier ÉCRIT la préférence, et un remontage la relit', () => {
    const { result, unmount } = renderHook(() => usePersistedFlag(TIMELINE_EXPANDED_KEY, true))
    act(() => result.current[1]())
    expect(result.current[0]).toBe(false)
    expect(localStorage.getItem(TIMELINE_EXPANDED_KEY)).toBe('false')

    unmount()
    const remonte = renderHook(() => usePersistedFlag(TIMELINE_EXPANDED_KEY, true))
    expect(remonte.result.current[0]).toBe(false)
  })

  it('la clé du repli est la SIENNE : replier ne touche à aucun calque', () => {
    const settings = renderHook(() => useReplaySettings())
    const { result } = renderHook(() => usePersistedFlag(TIMELINE_EXPANDED_KEY, true))
    act(() => result.current[1]())
    expect(settings.result.current.showAim).toBe(true)
    expect(settings.result.current.showZones).toBe(true)
  })
})
