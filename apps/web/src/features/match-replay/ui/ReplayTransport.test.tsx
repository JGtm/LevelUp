/**
 * Tests — ReplayTransport (la barre de lecture : icônes, sauts, vitesse, son, frise).
 *
 * Ce qu'ils protègent : les boutons en ICÔNE gardent leur libellé accessible (un symbole
 * sans nom serait une régression, pas une simplification), les SAUTS ±10 s portent leur durée
 * dans leur nom, les pastilles de sortie sont NOMMÉES, et le son n'affiche aucune commande
 * quand le match n'en a aucun.
 *
 * LA FRISE EST DEVENUE UN OBJET (planche 2a du 2026-08-28) : `timeline` porte le curseur, ses
 * bornes et les quatre pistes, exactement comme `sound` et `capture` portent les leurs. Ce
 * n'est pas un détail de forme — c'est ce qui permet au canvas, qui vit sous un cliquet de
 * taille, d'ajouter des pistes sans y gagner une prop de plus.
 */
import { createRef } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { ReplayTransport } from './ReplayTransport'
import { makeFrameControl } from '../test/frameControlStub'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplaySound } from '../sound/useReplaySound'
import type { ReplayExport } from '../export/useReplayExport'

function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
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
    ...over,
  }
}

/** La frise servie par le canvas (cf. useReplayTimeline) : vide de pistes par défaut. */
function makeTimeline(over: Partial<ReplayTimeline> = {}): ReplayTimeline {
  return {
    sliderRef: createRef<HTMLInputElement>(),
    minFrame: 0,
    maxFrame: 100,
    onScrub: vi.fn(),
    own: [],
    teammates: [],
    shades: [],
    absence: [],
    allegiance: NO_ALLEGIANCE,
    onSeekFrame: vi.fn(),
    viewpoint: null,
    viewpointGroups: [],
    onSelectViewpoint: vi.fn(),
    dominance: [],
    score: null,
    allyOf: () => null,
    labelOf: () => '',
    media: [],
    showMediaTrack: true,
    tracksExpanded: true,
    onToggleTracks: vi.fn(),
    playing: true,
    onRequestPause: vi.fn(),
    locale: 'fr',
    ...over,
  }
}

function renderTransport(over: Partial<Parameters<typeof ReplayTransport>[0]> = {}) {
  const onTogglePlay = vi.fn()
  const onRestart = vi.fn()
  const onSeekBy = vi.fn()
  const onSetSpeed = vi.fn()
  const captureImage = vi.fn()
  const toggleRecording = vi.fn()
  const utils = render(
    <ReplayTransport
      playing
      autoPlay={false}
      onToggleAutoPlay={vi.fn()}
      onTogglePlay={onTogglePlay}
      onRestart={onRestart}
      onSeekBy={onSeekBy}
      clockRef={createRef<HTMLSpanElement>()}
      timeline={makeTimeline()}
      speed={1}
      onSetSpeed={onSetSpeed}
      sound={makeSound()}
      capture={{
        captureImage,
        recordingSupported: true,
        recording: false,
        toggleRecording,
        videoExport: null,
      }}
      locale="fr"
      fullscreen={{ active: false, disabled: false, toggle: vi.fn() }}
      frameControl={makeFrameControl()}
      {...over}
    />,
  )
  return {
    ...utils,
    onTogglePlay,
    onRestart,
    onSeekBy,
    onSetSpeed,
    captureImage,
    toggleRecording,
  }
}

describe('ReplayTransport — lecture en icônes', () => {
  it('en lecture : le bouton dit « Pause » et bascule au clic', () => {
    const { onTogglePlay } = renderTransport({ playing: true })
    const btn = screen.getByRole('button', { name: 'Pause' })
    expect(btn.querySelector('svg')).toBeTruthy()
    fireEvent.click(btn)
    expect(onTogglePlay).toHaveBeenCalledTimes(1)
  })

  it('à l’arrêt : le bouton dit « Lecture »', () => {
    renderTransport({ playing: false })
    expect(screen.getByRole('button', { name: 'Lecture' }).querySelector('svg')).toBeTruthy()
  })

  // LE NOM RAPPELLE LE RACCOURCI (planche 2a) : « Recommencer (R) ». Le raccourci se découvre
  // là où on cherche la commande — pas dans une page d'aide que personne n'ouvre.
  it('« Recommencer » est une icône nommée qui rappelle sa touche, et appelle onRestart', () => {
    const { onRestart } = renderTransport()
    const btn = screen.getByRole('button', { name: /^Recommencer/ })
    expect(btn.getAttribute('aria-label')).toBe('Recommencer (R)')
    expect(btn.querySelector('svg')).toBeTruthy()
    fireEvent.click(btn)
    expect(onRestart).toHaveBeenCalledTimes(1)
  })

  it('LA FRISE NE SORT PAS DE LA FENÊTRE DE GAMEPLAY : un scrub est borné aux deux bouts', () => {
    renderTransport({ timeline: makeTimeline({ minFrame: 149, maxFrame: 4_929 }) })
    const frise = screen.getAllByLabelText('Temps de match').find((el) => el.tagName === 'INPUT')
    expect(frise).toBeTruthy()
    expect(frise).toHaveAttribute('min', '149')
    expect(frise).toHaveAttribute('max', '4929')
    // Le curseur démarre AU coup d'envoi, pas au premier paquet du film.
    expect((frise as HTMLInputElement).value).toBe('149')
  })
})

/**
 * LES SAUTS ±10 s (planche 2a) — le geste le plus fréquent d'un rejeu qu'on analyse, et il
 * n'existait qu'en tirant la frise à la main. Leur nom PORTE la durée : « Avancer » seul
 * laisserait deviner de combien.
 */
describe('ReplayTransport — les sauts encadrent la lecture', () => {
  it('les deux boutons portent leur durée dans leur nom accessible', () => {
    renderTransport()
    expect(screen.getByRole('button', { name: /Reculer de 10 s/ })).toBeTruthy()
    expect(screen.getByRole('button', { name: /Avancer de 10 s/ })).toBeTruthy()
  })

  it('reculer demande un saut NÉGATIF, avancer un saut positif', () => {
    const { onSeekBy } = renderTransport()
    fireEvent.click(screen.getByRole('button', { name: /Reculer de 10 s/ }))
    expect(onSeekBy).toHaveBeenCalledWith(-10)
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    expect(onSeekBy).toHaveBeenCalledWith(10)
  })
})

describe('ReplayTransport — la vitesse est un menu', () => {
  // Les quatre boutons occupaient la place de quatre commandes pour un seul réglage. Ce qui
  // reste dans la barre : le déclencheur, qui AFFICHE la valeur courante — l'information que
  // les quatre boutons donnaient d'un coup d'œil. Le menu lui-même est testé chez lui
  // (ReplaySpeedMenu.test.tsx).
  it('le déclencheur montre la vitesse courante', () => {
    renderTransport({ speed: 2 })
    expect(screen.getByRole('button', { name: 'Vitesse' }).textContent).toContain('2×')
  })

  it('fermé, aucun multiplicateur n’occupe la barre', () => {
    renderTransport({ speed: 1 })
    expect(screen.queryByRole('button', { name: '4×' })).toBeNull()
  })
})

describe('ReplayTransport — le son au niveau de la lecture', () => {
  it('l’interrupteur du son est dans la barre quand le match a des sons', () => {
    renderTransport()
    expect(screen.getByRole('button', { name: 'Son' })).toBeTruthy()
  })

  it('aucune commande de son quand le match n’en a aucun', () => {
    renderTransport({ sound: makeSound({ available: false }) })
    expect(screen.queryByRole('button', { name: 'Son' })).toBeNull()
  })

  it('son allumé : le curseur porte le volume réglé, et il est actionnable', () => {
    renderTransport({ sound: makeSound({ on: true, volume: 0.4 }) })
    const slider = screen.getByLabelText('Volume des sons') as HTMLInputElement
    expect(slider.value).toBe('40')
    expect(slider.disabled).toBe(false)
  })

  // DEMANDE UTILISATEUR DU 2026-08-25 : couper le son ne doit PLUS faire disparaître la barre
  // de volume. Ce cas tient les trois faits qui la composent — elle reste à l'écran, elle
  // montre zéro, et elle n'agit plus tant que le son est coupé.
  it('son coupé : le curseur RESTE, à zéro et inerte', () => {
    renderTransport({ sound: makeSound({ on: false, volume: 0.7 }) })
    const slider = screen.getByLabelText('Volume des sons') as HTMLInputElement
    expect(slider.value).toBe('0')
    expect(slider.disabled).toBe(true)
    expect(slider.getAttribute('title')).toMatch(/Son coupé/)
  })

  // LE NIVEAU RÉGLÉ N'EST PAS PERDU : le zéro affiché pendant la coupure est un AFFICHAGE, pas
  // une écriture. Le composant ne touche jamais `sound.volume` en basculant — la preuve est ici,
  // sur les deux rendus du même volume 0,7.
  it('rallumer le son rend le volume précédent — la coupure n’écrit rien', () => {
    const setVolume = vi.fn()
    const { unmount } = renderTransport({ sound: makeSound({ on: false, volume: 0.7, setVolume }) })
    expect(setVolume).not.toHaveBeenCalled()
    unmount()
    renderTransport({ sound: makeSound({ on: true, volume: 0.7, setVolume }) })
    expect((screen.getByLabelText('Volume des sons') as HTMLInputElement).value).toBe('70')
  })
})

describe('ReplayTransport — le tiroir de réglages a quitté la barre', () => {
  // LES RÉGLAGES D'AFFICHAGE VIVENT SOUS LE BOUTON « VUE » DE LA CARTE : la barre n'en porte plus.
  it('aucun bouton Réglages ni Vue dans la barre : le plein écran la ferme', () => {
    renderTransport()
    expect(screen.queryByRole('button', { name: 'Réglages' })).toBeNull()
    expect(screen.queryByRole('button', { name: /^Vue/ })).toBeNull()
    expect(screen.getByRole('button', { name: 'Plein écran' })).toBeTruthy()
  })
})

/**
 * LA LECTURE AUTOMATIQUE A QUITTÉ LE TIROIR le 2026-09-02 (« on a carrément la place pour un
 * bouton comme YouTube ») : ces tests ont déménagé depuis l’ancien test du tiroir de réglages, avec
 * leur raison d'être. Le point le plus important est le DERNIER — ce bouton ne met ni en lecture
 * ni en pause, et rien à l'écran ne le distingue de « Lecture » sinon son nom et son infobulle.
 */
describe('ReplayTransport — lecture automatique', () => {
  it('est un interrupteur, reflète son état et appelle SON callback', () => {
    const onToggleAutoPlay = vi.fn()
    const { onTogglePlay } = renderTransport({ autoPlay: false, onToggleAutoPlay })
    const sw = screen.getByRole('switch', { name: 'Lecture automatique' })
    expect(sw).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(sw)
    expect(onToggleAutoPlay).toHaveBeenCalledTimes(1)
    // La commande VOISINE est celle qu'on risque de toucher : les deux parlent de lecture.
    expect(onTogglePlay).not.toHaveBeenCalled()
  })

  it('dit en clair qu il ne met ni en lecture ni en pause le rejeu ouvert', () => {
    renderTransport()
    expect(screen.getByRole('switch', { name: 'Lecture automatique' })).toHaveAttribute(
      'title',
      expect.stringContaining('ni en lecture ni en pause'),
    )
  })
})

describe('ReplayTransport — ce qui sort du rejeu et les cadres sont à droite', () => {
  it('le cartouche de capture et les icônes de cadre sont dans la barre', () => {
    renderTransport()
    expect(screen.getByRole('button', { name: "Capturer l'image" })).toBeTruthy()
    expect(screen.getByRole('group', { name: 'Cadre du lecteur' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Plein écran' })).toBeTruthy()
  })

  it('pendant un export, la lecture est neutralisée dès la préparation, pas seulement à l’encodage', () => {
    const videoExport = {
      supported: true,
      state: { phase: 'prepare', done: 0, total: 1200, pct: 0 },
      defaultBounds: () => ({ startFrame: 0, endFrame: 100 }),
      run: vi.fn(async () => {}),
      cancel: vi.fn(),
      clockOf: () => '0:00',
      lengthClock: () => '1:40',
      zoomLevel: 1,
    } satisfies ReplayExport
    renderTransport({
      capture: { captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(), videoExport },
    })
    // Le rejeu est en lecture dans ce montage : le bouton porte donc « Pause ».
    expect(screen.getByRole('button', { name: 'Pause' })).toBeDisabled()
  })
})
