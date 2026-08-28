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

import { ReplayTransport } from './ReplayTransport'
import type { ReplayTimeline } from './useReplayTimeline'
import type { ReplaySound } from './useReplaySound'

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
    endMatch: vi.fn(),
    recordingTrack: () => null,
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
    allies: [],
    dominance: [],
    allyOf: () => null,
    labelOf: () => '',
    media: [],
    showMediaTrack: true,
    tracksExpanded: true,
    onToggleTracks: vi.fn(),
    playing: true,
    onRequestPause: vi.fn(),
    startClock: '0:00',
    midClock: '2:30',
    endClock: '5:00',
    locale: 'fr',
    ...over,
  }
}

function renderTransport(over: Partial<Parameters<typeof ReplayTransport>[0]> = {}) {
  const onTogglePlay = vi.fn()
  const onRestart = vi.fn()
  const onSeekBy = vi.fn()
  const onSetSpeed = vi.fn()
  const onToggleSettings = vi.fn()
  const captureImage = vi.fn()
  const toggleRecording = vi.fn()
  const utils = render(
    <ReplayTransport
      playing
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
      }}
      locale="fr"
      settingsOpen={false}
      onToggleSettings={onToggleSettings}
      settingsButtonRef={createRef<HTMLButtonElement>()}
      {...over}
    />,
  )
  return {
    ...utils,
    onTogglePlay,
    onRestart,
    onSeekBy,
    onSetSpeed,
    onToggleSettings,
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

describe('ReplayTransport — ce qui sort du rejeu', () => {
  it('« Capturer l’image » est une icône nommée, et commande la capture', () => {
    const { captureImage } = renderTransport()
    const btn = screen.getByRole('button', { name: "Capturer l'image" })
    expect(btn.querySelector('svg')).toBeTruthy()
    fireEvent.click(btn)
    expect(captureImage).toHaveBeenCalledTimes(1)
  })

  // LES PASTILLES SONT NOMMÉES À L'ŒIL (planche 2a) : le texte court accompagne l'icône, sans
  // remplacer le nom accessible — trois icônes muettes côte à côte se ressemblent toutes.
  it('les pastilles portent leur texte court, et gardent leur nom accessible long', () => {
    renderTransport()
    expect(screen.getByRole('button', { name: "Capturer l'image" }).textContent).toContain('Image')
    expect(screen.getByRole('button', { name: 'Enregistrer la vidéo' }).textContent).toContain('REC')
  })

  it('en cours d’enregistrement, la pastille dit « Arrêter »', () => {
    renderTransport({
      capture: {
        captureImage: vi.fn(),
        recordingSupported: true,
        recording: true,
        toggleRecording: vi.fn(),
      },
    })
    expect(
      screen.getByRole('button', { name: "Arrêter l'enregistrement" }).textContent,
    ).toContain('Arrêter')
  })

  it('à l’arrêt : le bouton dit « Enregistrer la vidéo » et lance au clic', () => {
    const { toggleRecording } = renderTransport()
    const btn = screen.getByRole('button', { name: 'Enregistrer la vidéo' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(toggleRecording).toHaveBeenCalledTimes(1)
  })

  it('en cours : le nom accessible dit ce que le CLIC fera, pas l’état', () => {
    renderTransport({
      capture: {
        captureImage: vi.fn(),
        recordingSupported: true,
        recording: true,
        toggleRecording: vi.fn(),
      },
    })
    const btn = screen.getByRole('button', { name: "Arrêter l'enregistrement" })
    expect(btn).toHaveAttribute('aria-pressed', 'true')
  })

  // DÉCISION 7 : un navigateur qui ne sait pas filmer une toile n'a pas de bouton du tout. Une
  // commande grisée laisserait croire à une panne réparable — il n'y a rien à réparer.
  it('navigateur sans enregistrement : PAS de bouton vidéo, mais le bouton image reste', () => {
    renderTransport({
      capture: {
        captureImage: vi.fn(),
        recordingSupported: false,
        recording: false,
        toggleRecording: vi.fn(),
      },
    })
    expect(screen.queryByRole('button', { name: 'Enregistrer la vidéo' })).toBeNull()
    expect(screen.getByRole('button', { name: "Capturer l'image" })).toBeTruthy()
  })
})

describe('ReplayTransport — les réglages ferment la barre', () => {
  it('le bouton du tiroir est là, en icône nommée, et bascule au clic', () => {
    const { onToggleSettings } = renderTransport()
    const btn = screen.getByRole('button', { name: 'Réglages' })
    expect(btn.querySelector('svg')).toBeTruthy()
    expect(btn).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(btn)
    expect(onToggleSettings).toHaveBeenCalledTimes(1)
  })
})
