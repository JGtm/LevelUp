/**
 * Tests — ReplayAntagonistBand : la bande des titres au-dessus de la rangée du joueur regardé (C.2, D-13).
 *
 * Ils protègent :
 *  1. UNE ICÔNE-BOUTON PAR PRISE DE TITRE, AUCUNE POUR UNE ÉGALITÉ ;
 *  2. LE NOM ACCESSIBLE « titre. phrase » ET L'INFOBULLE « titre », saut de ligne, « phrase » (le
 *     même couple que les boutons des tuiles), en français et en anglais ;
 *  3. LE CLIC POSE LE CURSEUR à l'image de la prise (sans pause : c'est `onSeekFrame` des portes) ;
 *  4. LA GÉOMÉTRIE DES MARQUES : l'abscisse est celle de `trackLeft`, jamais recopiée ;
 *  5. UNE BANDE VIDE RESTE UNE BANDE : sans point de vue, ou sans prise, la rangée garde sa hauteur
 *     et ne porte aucun texte — passer d'un joueur à l'autre ne décale pas la frise ;
 *  6. L'ENCRE EST CELLE DU CAMP DU TENANT (jetons), et la souris ne prend pas le focus.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { Antagonists, AntagonistChange } from '@/lib/antagonists/antagonists'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayAntagonists } from '../model/replayAntagonists'
import { trackLeft } from '../model/replayTimelineTracksLogic'
import { ReplayAntagonistBand } from './ReplayAntagonistBand'

const PLAYERS: ReplayPlayer[] = [
  { xuid: 'me-1', filmName: 'JGtm', lives: [] },
  { xuid: 'foe-1', filmName: 'Rival', lives: [] },
  { xuid: 'foe-2', filmName: 'Fantome', lives: [] },
]

function change(over: Partial<AntagonistChange>): AntagonistChange {
  return { t: 10_000, role: 'nemesis', kind: 'prise', holder: 'foe-1', previousHolder: null, count: 1, previousCount: 0, ...over }
}

const CHANGES: AntagonistChange[] = [
  change({ t: 10_000 }),
  change({ t: 20_000, role: 'victim', holder: 'foe-2' }),
  change({ t: 30_000, kind: 'egalite', holder: 'foe-2', previousHolder: 'foe-1', count: 1, previousCount: 1 }),
  change({ t: 40_000, holder: 'foe-2', previousHolder: 'foe-1', count: 2, previousCount: 1 }),
]

/** Une fabrique qui rend ces changements au point de vue `me-1` et rien à personne d'autre. */
function fabrique(changes: readonly AntagonistChange[]): ReplayAntagonists {
  const index: Antagonists = { changes, at: () => ({ nemesis: null, victim: null }) }
  const vide: Antagonists = { changes: [], at: () => ({ nemesis: null, victim: null }) }
  return {
    forSubject: (key) => (key === 'me-1' ? index : vide),
    forViewpoint: (viewpoint) => (viewpoint === 'me-1' ? index : vide),
  }
}

function monter(over: Partial<Parameters<typeof ReplayAntagonistBand>[0]> = {}) {
  const onSeekFrame = vi.fn()
  const utils = render(
    // La bande vit dans la grille de la frise : un conteneur en grille à deux colonnes la reçoit.
    <div style={{ display: 'grid', gridTemplateColumns: '100px 1fr' }}>
      <ReplayAntagonistBand
        antagonists={fabrique(CHANGES)}
        viewpoint="me-1"
        players={PLAYERS}
        allegiance={{ ofXuid: (key) => key === 'me-1' }}
        playWindow={null}
        frameCount={1001}
        frameIntervalMs={100}
        locale="fr"
        onSeekFrame={onSeekFrame}
        {...over}
      />
    </div>,
  )
  return { ...utils, onSeekFrame }
}

describe('ReplayAntagonistBand — une icône-bouton par prise', () => {
  it('pose trois boutons pour trois prises et aucun pour l égalité', () => {
    monter()
    expect(screen.getAllByRole('button')).toHaveLength(3)
  })

  it('le nom accessible est « titre. phrase » (français)', () => {
    monter()
    expect(screen.getByRole('button', { name: 'Némésis : Rival. Premier adversaire à éliminer JGtm.' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Souffre-douleur : Fantome. Premier adversaire éliminé par JGtm.' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Némésis : Fantome. A éliminé JGtm 2 fois et passe devant Rival (1).' })).toBeTruthy()
  })

  it('l infobulle est « titre », un saut de ligne, « phrase » (français)', () => {
    monter()
    const bouton = screen.getByRole('button', { name: /^Némésis : Rival\./ })
    expect(bouton.getAttribute('title')).toBe('Némésis : Rival\nPremier adversaire à éliminer JGtm.')
  })

  it('le nom accessible et l infobulle sont ceux de l anglais en anglais', () => {
    monter({ locale: 'en' })
    const bouton = screen.getByRole('button', { name: 'Bully target: Fantome. First opponent eliminated by JGtm.' })
    expect(bouton.getAttribute('title')).toBe('Bully target: Fantome\nFirst opponent eliminated by JGtm.')
    expect(screen.getByRole('button', { name: 'Nemesis: Fantome. Eliminated JGtm 2 times and moves ahead of Rival (1).' })).toBeTruthy()
  })

  it('chaque bouton porte l icône de son rôle', () => {
    monter()
    const [premier, deuxieme] = screen.getAllByRole('button')
    expect(premier.querySelector('[data-antagonist-icon="nemesis"]')).toBeTruthy()
    expect(deuxieme.querySelector('[data-antagonist-icon="victim"]')).toBeTruthy()
  })
})

describe('ReplayAntagonistBand — le clic, la place, l encre', () => {
  it('le clic pose le curseur à l image de la prise', () => {
    const { onSeekFrame } = monter()
    fireEvent.click(screen.getByRole('button', { name: /^Souffre-douleur : Fantome/ }))
    expect(onSeekFrame).toHaveBeenCalledTimes(1)
    expect(onSeekFrame).toHaveBeenCalledWith(200)
  })

  it('l abscisse est celle de trackLeft', () => {
    monter()
    const bouton = screen.getByRole('button', { name: /^Némésis : Rival/ })
    // jsdom range le calc() à sa façon : on compare à une sonde qui reçoit la même valeur.
    const sonde = document.createElement('span')
    sonde.style.left = trackLeft(0.1)
    expect(bouton.style.left).toBe(sonde.style.left)
    expect(bouton.style.left).not.toBe('')
  })

  it('l icône prend l encre du camp du tenant (adversaire : encre adverse)', () => {
    monter()
    const bouton = screen.getByRole('button', { name: /^Némésis : Rival/ })
    expect(bouton.style.color).toContain('--ac-team-enemy')
  })

  it('la souris ne prend pas le focus : Espace reste « Lecture »', () => {
    monter()
    const bouton = screen.getByRole('button', { name: /^Némésis : Rival/ })
    expect(fireEvent.mouseDown(bouton)).toBe(false)
  })

  it('le bouton reçoit le pointeur : la bande n est pas pointer-events-none', () => {
    const { container } = monter()
    expect(container.querySelector('[data-replay-antagonist-band]')?.className).not.toContain('pointer-events-none')
  })
})

describe('ReplayAntagonistBand — une bande vide reste une bande', () => {
  it('sans point de vue : la rangée existe, sans bouton ni texte', () => {
    const { container } = monter({ viewpoint: null })
    const bande = container.querySelector('[data-replay-antagonist-band]')
    expect(bande).toBeTruthy()
    expect(bande?.textContent).toBe('')
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('un point de vue sans changement : la rangée existe, sans bouton ni texte', () => {
    const { container } = monter({ antagonists: fabrique([]) })
    const bande = container.querySelector('[data-replay-antagonist-band]')
    expect(bande).toBeTruthy()
    expect(bande?.textContent).toBe('')
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('la rangée garde ses quatorze pixels', () => {
    const { container } = monter({ antagonists: fabrique([]) })
    expect(container.querySelector('[data-replay-antagonist-band]')?.className).toContain('h-[14px]')
  })
})
