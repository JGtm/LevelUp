/**
 * Tests — le sélecteur de joueurs de la fiche du Standard.
 *
 * Ce qu'ils protègent :
 *  1. UN RECTANGLE PAR JOUEUR, à deux initiales, nom complet en infobulle et en nom accessible, camps l'un après
 *     l'autre avec un espace plus large entre eux, de la taille des cotes.
 *  2. LE JOUEUR OUVERT porte le double anneau et `aria-pressed` ; le choisir de nouveau ne referme rien.
 *  3. UN JOUEUR MORT À L'IMAGE LUE est estompé et le dit (`data-dead`), sans cesser d'être choisissable.
 *  4. UN CLIC OUVRE LA FICHE de ce joueur (`onCard`), sans passer par la colonne.
 *  5. EN GRANDE ÉQUIPE (12 contre 12), chaque camp se replie sur plusieurs rangées : conteneurs `flex-wrap`
 *     bornés à leur parent, rectangles qui ne rétrécissent pas. La mise en page se résout dans le
 *     navigateur : ici on lit ce que le rendu porte.
 */
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { COTES_FICHE_STANDARD, SELECTEUR_ANNEAU_TOTAL_PX } from '../model/cotesFicheStandard'
import { COTES_COLONNE_STANDARD } from '../model/cotesStandard'
import { testReplayDoc } from '../test/testDoc'
import { ReplayPlayerPicker } from './ReplayPlayerPicker'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const S = COTES_FICHE_STANDARD.selecteur

afterEach(cleanup)

/** Une vie par joueur sur [0, 100], sauf `mortAvant` : sa vie s'arrête à l'image 30. */
function docOf(parCamp: number, mortAvant?: string) {
  const roster = []
  const tracks = []
  let index = 0
  for (const team of [0, 1]) {
    for (let k = 1; k <= parCamp; k++) {
      const xuid = `${team}-${k}`
      roster.push({ xuid, filmIndex: index, name: team === 0 ? `Allie${k}` : `Rival${k}`, team })
      tracks.push({
        slot: 512 + index, team: -1, xuid, startFrame: 0, endFrame: xuid === mortAvant ? 30 : 100,
        points: [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }],
      })
      index += 1
    }
  }
  return testReplayDoc({ frameIntervalMs: 100, roster, tracks })
}

function setup(parCamp: number, over: { frame?: number; selected?: string; mortAvant?: string } = {}) {
  const doc = docOf(parCamp, over.mortAvant)
  const players = buildPlayers(doc, [])
  const onCard = vi.fn()
  render(
    <ReplayPlayerPicker
      players={players}
      frame={over.frame ?? 50}
      locale="fr"
      allegiance={filmAllegianceOf(doc, [], '0-1')}
      selectedXuid={over.selected ?? '0-1'}
      onCard={onCard}
    />,
  )
  return { onCard }
}

describe('ReplayPlayerPicker — les rectangles', () => {
  it('un par joueur, deux initiales en capitales, nom complet en infobulle et en nom accessible', () => {
    setup(4)
    const groupe = screen.getByRole('group', { name: 'Changer de joueur' })
    const boutons = within(groupe).getAllByRole('button')
    expect(boutons).toHaveLength(8)
    const premier = within(groupe).getByRole('button', { name: 'Allie1' })
    expect(premier.textContent).toBe('AL')
    expect(premier.getAttribute('title')).toBe('Allie1')
    expect(within(groupe).getByRole('button', { name: 'Rival4' }).textContent).toBe('RI')
  })

  it('la taille, le rayon et le corps viennent des cotes ; la couleur est celle du camp, en jeton', () => {
    setup(4)
    const allie = screen.getByRole('button', { name: 'Allie1' })
    const rival = screen.getByRole('button', { name: 'Rival1' })
    expect(allie.style.width).toBe(`${S.largeurPx}px`)
    expect(allie.style.height).toBe(`${S.hauteurPx}px`)
    expect(allie.style.borderRadius).toBe(`${S.rayonPx}px`)
    expect(allie.style.background).toContain('team-ally')
    expect(rival.style.background).toContain('team-enemy')
    // Aucune bordure colorée à gauche, ni aucune bordure du tout.
    expect(allie.style.borderLeft).toBe('')
  })

  it('les camps sont deux groupes, séparés par l’espace entre camps ; les rectangles d’un camp par l’écart des cotes', () => {
    setup(4)
    const camps = document.querySelectorAll<HTMLElement>('[data-replay-picker-camp]')
    expect(camps).toHaveLength(2)
    expect(within(camps[0]).getAllByRole('button').map((b) => b.getAttribute('aria-label'))).toEqual(['Allie1', 'Allie2', 'Allie3', 'Allie4'])
    expect(within(camps[1]).getAllByRole('button')).toHaveLength(4)
    const groupe = screen.getByRole('group', { name: 'Changer de joueur' })
    expect(groupe.style.columnGap).toBe(`${S.ecartCampsPx}px`)
    expect(camps[0].style.columnGap).toBe(`${S.ecartPx}px`)
    expect(S.ecartCampsPx).toBeGreaterThan(S.ecartPx)
  })
})

describe('ReplayPlayerPicker — le joueur ouvert, le joueur mort', () => {
  it('le joueur ouvert porte le double anneau et aria-pressed, lui seul', () => {
    setup(4, { selected: '0-2' })
    const ouvert = screen.getByRole('button', { name: 'Allie2' })
    expect(ouvert.getAttribute('aria-pressed')).toBe('true')
    expect(ouvert.style.boxShadow).toContain(`0 0 0 ${S.anneauFondPx}px var(--card)`)
    expect(ouvert.style.boxShadow).toContain(`0 0 0 ${SELECTEUR_ANNEAU_TOTAL_PX}px var(--foreground)`)
    const autres = screen.getAllByRole('button', { pressed: false })
    expect(autres).toHaveLength(7)
    for (const b of autres) expect(b.style.boxShadow).toBe('')
  })

  it('un joueur mort à l’image lue est estompé et le dit ; un joueur vivant ne l’est pas', () => {
    setup(4, { mortAvant: '1-2', frame: 50 })
    const mort = screen.getByRole('button', { name: 'Rival2' })
    expect(mort.getAttribute('data-dead')).toBe('true')
    expect(mort.style.opacity).toBe(String(S.mortOpacite))
    const vivant = screen.getByRole('button', { name: 'Rival3' })
    expect(vivant.hasAttribute('data-dead')).toBe(false)
    expect(vivant.style.opacity).toBe('')
  })

  it('à une image où sa vie court encore, le même joueur n’est pas estompé', () => {
    cleanup()
    setup(4, { mortAvant: '1-2', frame: 10 })
    expect(screen.getByRole('button', { name: 'Rival2' }).hasAttribute('data-dead')).toBe(false)
  })
})

describe('ReplayPlayerPicker — le clic', () => {
  it('ouvre la fiche du joueur choisi, par son identité', () => {
    const { onCard } = setup(4)
    fireEvent.click(screen.getByRole('button', { name: 'Rival3' }))
    expect(onCard).toHaveBeenCalledWith('1-3')
    expect(onCard).toHaveBeenCalledTimes(1)
  })

  it('un joueur mort reste choisissable', () => {
    const { onCard } = setup(4, { mortAvant: '1-2' })
    fireEvent.click(screen.getByRole('button', { name: 'Rival2' }))
    expect(onCard).toHaveBeenCalledWith('1-2')
  })

  it('le joueur déjà ouvert ne referme pas la fiche : aucun geste', () => {
    const { onCard } = setup(4, { selected: '0-1' })
    fireEvent.click(screen.getByRole('button', { name: 'Allie1' }))
    expect(onCard).not.toHaveBeenCalled()
  })

  it('la souris ne prend pas le focus : Espace reste « Lecture »', () => {
    setup(4)
    const evenement = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    screen.getByRole('button', { name: 'Allie2' }).dispatchEvent(evenement)
    expect(evenement.defaultPrevented).toBe(true)
  })
})

describe('ReplayPlayerPicker — Grande équipe, 12 contre 12', () => {
  it('chaque camp compte 12 rectangles, sur des conteneurs qui se replient et ne dépassent pas leur parent', () => {
    setup(12)
    const camps = document.querySelectorAll<HTMLElement>('[data-replay-picker-camp]')
    expect(camps).toHaveLength(2)
    for (const camp of camps) {
      expect(within(camp).getAllByRole('button')).toHaveLength(12)
      expect(camp.className).toContain('flex-wrap')
      expect(camp.className).toContain('max-w-full')
      expect(camp.className).toContain('min-w-0')
      expect(camp.style.rowGap).toBe(`${S.ecartRangeesPx}px`)
    }
    expect(screen.getByRole('group', { name: 'Changer de joueur' }).className).toContain('flex-wrap')
  })

  it('les rectangles gardent leur taille : aucun ne rétrécit pour tenir sur une rangée', () => {
    setup(12)
    for (const b of screen.getAllByRole('button')) {
      expect(b.className).toContain('shrink-0')
      expect(b.style.width).toBe(`${S.largeurPx}px`)
    }
  })

  it('un camp plus large que la colonne tient sur deux rangées, huit rectangles par rangée au plus', () => {
    // La colonne offre sa largeur moins ses deux marges et la barre de défilement : 8,4 rectangles et leurs écarts.
    const largeurUtile =
      COTES_COLONNE_STANDARD.largeurPx - 2 * COTES_COLONNE_STANDARD.margeLateralePx - COTES_FICHE_STANDARD.margeBarrePx
    const parRangee = Math.floor((largeurUtile + S.ecartPx) / (S.largeurPx + S.ecartPx))
    expect(parRangee).toBe(8)
    expect(Math.ceil(12 / parRangee)).toBe(2)
  })
})
