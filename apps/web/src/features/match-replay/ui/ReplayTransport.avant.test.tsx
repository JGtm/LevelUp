/**
 * ReplayTransport.avant.test.tsx — fixations de la zone de lecture (barre et frise à pistes).
 *
 * Ce que les fixations couvrent : la barre (lecture automatique, son, vitesse, recommencer, sauts ±10 s,
 * lecture/pause, image, enregistrement) et la frise dépliée, avec quatre pistes (point de vue, coéquipiers,
 * dominance, médias), des marques, un segment de score et un média. Deux fixations s'y ajoutent pour
 * l'export vidéo : au repos, puis pendant l'encodage.
 *
 * Le transport ne lit pas le document de rejeu : il reçoit la frise déjà calculée. Le document de test
 * (`test/testDoc`) fournit les bornes (nombre d'images) ; les marques sont posées avec `test/timelineTracksHarness`.
 *
 * Aucune valeur ne varie : l'horloge est écrite par la boucle de dessin (`clockRef` reste vide) et aucun
 * identifiant généré n'entre dans le rendu. Si un lancement en faisait apparaître un, le neutraliser dans le test.
 *
 * LES FIXTURES RESTENT CELLES D'AVANT LES ICÔNES DE CADRE, jamais régénérées. Le rendu actuel ne diffère
 * d'elles que par UN groupe : l'ancien bouton de plein écran (`button[aria-label="Plein écran"]`, seul de
 * son espèce) est devenu le groupe des icônes de cadre (`[data-replay-frame-buttons]`), posé au même endroit.
 * La comparaison retire ce seul élément de chaque côté, puis exige le même HTML : tout le reste de la zone de
 * lecture est identique, octet pour octet. Les icônes de cadre ont leurs propres tests (ReplayFrameButtons.test.tsx).
 *
 * SEUL AUTRE ÉCART ASSUMÉ (2026-11-22, C.1 du plan des antagonistes, D-5) : sur la rangée du point de vue, une
 * élimination se pose dans la moitié haute (`top-0.5`) et une mort dans la moitié basse (`top-[14px]`), au lieu de
 * `top-2` pour les deux. Les trois fixtures portent ce seul changement de classe, rien d autre.
 */
import { createRef } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { render, within } from '@testing-library/react'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { ReplayTransport } from './ReplayTransport'
import avantNormal from './__fixtures__/replayTransport.avant.html?raw'
import avantExport from './__fixtures__/replayTransport.export.avant.html?raw'
import avantExportEnCours from './__fixtures__/replayTransport.exportEnCours.avant.html?raw'
import { makeFrameControl } from '../test/frameControlStub'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplayExport, ReplayExportState } from '../export/useReplayExport'
import type { ReplaySound } from '../sound/useReplaySound'
import type { PlacedMedia, ReplayScoreTrack } from '../model/replayTimelineTracksLogic'
import { mark } from '../test/timelineTracksHarness'
import { testReplayDoc } from '../test/testDoc'

/** Son désactivé mais présent : les commandes de son s'affichent telles qu'un match les montre. */
function makeSound(): ReplaySound {
  return {
    available: true,
    on: false,
    toggle: vi.fn(),
    wake: vi.fn(),
    volume: 0.7,
    setVolume: vi.fn(),
    mutedBySpeed: false,
    categories: { weapon: true, grenade: true, melee: true, equipment: true, objective: true },
    toggleCategory: vi.fn(),
    tick: vi.fn(),
    seek: vi.fn(),
    setTransportPlaying: vi.fn(),
    endMatch: vi.fn(),
    intro: vi.fn(),
    recordingTrack: () => null,
    exportTrack: () => ({ timeline: [], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] }, engines: [] }),
  } as unknown as ReplaySound
}

/** Un exporteur vidéo pris dans l'état donné : supporté, sans effet de bord (les actions sont des espions). */
function exporteur(state: ReplayExportState): ReplayExport {
  return {
    supported: true,
    state,
    defaultBounds: vi.fn(),
    run: vi.fn(),
    zoomLevel: 1,
    cancel: vi.fn(),
    clockOf: () => '0:00',
    lengthClock: () => '0:00',
  } as unknown as ReplayExport
}

/**
 * La frise du transport, riche : deux marques du joueur regardé (un kill, une mort), une marque de
 * coéquipier (ami, losange), deux segments de dominance, un segment de score avec une manche terminée,
 * un média posé. Chaque piste a au moins une marque, pour qu'une piste vide ne passe pas pour absente.
 */
function frise(): ReplayTimeline {
  const doc = testReplayDoc({ frameCount: 200 })
  const score: ReplayScoreTrack = {
    segments: [
      { key: 'score-1', from: 0, to: 0.5, teamId: 0 },
      { key: 'score-2', from: 0.5, to: 1, teamId: 1 },
    ],
    rounds: [{ key: 'round-1', endedIndex: 0, ratio: 0.5 }],
  }
  const media: PlacedMedia[] = [
    { item: { id: 'media-1', kind: 'image', replayMs: 30_000, thumbUrl: '/thumb.png', url: '/full.png', label: 'Capture' }, from: 0.7, to: 0.7 },
  ]
  return {
    sliderRef: createRef<HTMLInputElement>(),
    minFrame: 0,
    maxFrame: doc.frameCount - 1,
    onScrub: vi.fn(),
    own: [mark({ key: 'k1', ratio: 0.2, kind: 'kill', clock: '0:40' }), mark({ key: 'd1', ratio: 0.6, kind: 'death', clock: '1:30' })],
    teammates: [mark({ key: 't1', ratio: 0.35, kind: 'kill', clock: '1:00', friend: true })],
    shades: [],
    absence: [],
    allegiance: NO_ALLEGIANCE,
    onSeekFrame: vi.fn(),
    viewpoint: 'me-1',
    viewpointGroups: [
      {
        key: 'camp:0',
        label: 'Cobalt',
        options: [
          { value: 'me-1', rosterKey: 'me-1', label: 'JGtm', disabled: false, title: 'JGtm' },
          { value: 'bot-base', rosterKey: 'bot:Cortana', label: 'Cortana', disabled: false, title: 'Cortana' },
        ],
      },
    ],
    onSelectViewpoint: vi.fn(),
    dominance: score.segments,
    score,
    allyOf: (teamId: number) => teamId === 0,
    labelOf: (teamId: number) => `Équipe ${teamId}`,
    media,
    showMediaTrack: true,
    tracksExpanded: true,
    onToggleTracks: vi.fn(),
    playing: true,
    onRequestPause: vi.fn(),
    locale: 'fr',
  }
}

/** Le HTML, ré-analysé puis privé de l'élément que le sélecteur désigne : exactement un, ou le test échoue. */
function sansUnique(html: string, selecteur: string): string {
  const gabarit = document.createElement('div')
  gabarit.innerHTML = html
  const trouves = gabarit.querySelectorAll(selecteur)
  expect(trouves, `un seul '${selecteur}' attendu`).toHaveLength(1)
  trouves[0].remove()
  return gabarit.innerHTML
}

/** Le rendu actuel et la fixture d'avant, chacun privé de ce qui les distingue : le groupe d'icônes de cadre, l'ancien bouton de plein écran. */
function comparables(rendu: HTMLElement, avant: string) {
  return {
    actuel: sansUnique(rendu.innerHTML, '[data-replay-frame-buttons]'),
    ancien: sansUnique(avant, 'button[aria-label="Plein écran"]'),
  }
}

/** Rend la zone de lecture, lecture en cours, avec l'export vidéo donné (ou son absence). */
function rendreTransport(videoExport: ReplayExport | null) {
  return render(
    <ReplayTransport
      playing
      autoPlay={false}
      onToggleAutoPlay={vi.fn()}
      onTogglePlay={vi.fn()}
      onRestart={vi.fn()}
      onSeekBy={vi.fn()}
      clockRef={createRef<HTMLSpanElement>()}
      timeline={frise()}
      speed={1}
      onSetSpeed={vi.fn()}
      sound={makeSound()}
      capture={{
        captureImage: vi.fn(),
        recordingSupported: true,
        recording: false,
        toggleRecording: vi.fn(),
        videoExport,
      }}
      locale="fr"
      fullscreen={{ active: false, disabled: false, toggle: vi.fn() }}
      frameControl={makeFrameControl()}
    />,
  )
}

describe('ReplayTransport — fixation de la zone de lecture', () => {
  it('le HTML de la barre et de la frise à pistes est celui de la fixture, hors les icônes de cadre', () => {
    const vue = rendreTransport(null)
    // Les éléments attendus sont là : sans eux la fixture comparerait un rendu vide ou partiel.
    expect(vue.getByRole('button', { name: 'Pause' })).toBeTruthy()
    expect(vue.getByRole('button', { name: /^Recommencer/ })).toBeTruthy()
    expect(vue.getByLabelText('Joueur suivi')).toBeTruthy()
    expect(vue.getByText('Médias')).toBeTruthy()
    const { actuel, ancien } = comparables(vue.container, avantNormal)
    expect(actuel).toBe(ancien)
  })
})

describe('ReplayTransport — fixation du bouton d’export vidéo', () => {
  it('au repos : le bouton « Exporter la vidéo » est présent, sans pourcentage', () => {
    const vue = rendreTransport(exporteur({ phase: 'idle', done: 0, total: 0, pct: 0 }))
    const bouton = within(vue.container).getByRole('button', { name: 'Exporter la vidéo' })
    expect(bouton.textContent).not.toContain('%')
    const { actuel, ancien } = comparables(vue.container, avantExport)
    expect(actuel).toBe(ancien)
  })

  it('pendant l’encodage : le bouton affiche le pourcentage et les commandes de lecture sont neutralisées', () => {
    const vue = rendreTransport(exporteur({ phase: 'encode', done: 300, total: 1200, pct: 25 }))
    const bouton = within(vue.container).getByRole('button', { name: 'Exporter la vidéo' })
    expect(bouton.textContent).toContain('25 %')
    expect(within(vue.container).getByRole('button', { name: 'Pause' })).toBeDisabled()
    const { actuel, ancien } = comparables(vue.container, avantExportEnCours)
    expect(actuel).toBe(ancien)
  })
})
