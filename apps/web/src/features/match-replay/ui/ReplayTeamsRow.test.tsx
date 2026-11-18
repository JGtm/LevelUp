/**
 * Tests — la rangée des équipes du Cinéma et du Plein cadre (`ReplayTeamsRow`).
 *
 * Ce qu'ils protègent : l'ordre gauche | score | droite et le camp allié à gauche ; les colonnes à parts égales
 * (grille posée en `style`, jamais une largeur de fiche) ; la fiche ordinaire au gabarit compact, arme comprise ;
 * le joueur suivi, le joueur mort, les places en attente ; le clic qui rend la page avec la bonne clé ; la Grande
 * équipe en emplacements de deux fiches empilées (le dernier peut n'en avoir qu'une) ; le score qu'on lui confie et
 * rien sous lui.
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { NO_ALLEGIANCE, type FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayTeamsRow } from './ReplayTeamsRow'
import { COTES_RANGEE, COTES_RANGEE_GRANDE_EQUIPE } from '../model/cardGabarit'
import { trackedPlayerOf } from '../model/playerPanelLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

type Props = Omit<ComponentProps<typeof ReplayTeamsRow>, 'players' | 'tracked' | 'allegiance' | 'score' | 'grandeEquipe'> & {
  scoreboard: MatchScoreboardRow[]
  /** Le joueur regardé par la page (clé de feuille ou de roster) ; `null` = personne. */
  watched?: string | null
  grandeEquipe?: boolean
  allegiance?: FilmAllegiance
}

const SCORE = <div data-testid="score">Score</div>

/** La rangée montée depuis une feuille, comme la page : jointure film <-> base, puis joueur suivi retrouvé au roster. */
function Rangee({ scoreboard, watched = null, grandeEquipe = false, allegiance = NO_ALLEGIANCE, ...rest }: Props) {
  const players = useMemo(() => buildPlayers(rest.doc, scoreboard), [rest.doc, scoreboard])
  const tracked = useMemo(() => trackedPlayerOf(watched, players), [watched, players])
  return (
    <ReplayTeamsRow {...rest} players={players} tracked={tracked} allegiance={allegiance} score={SCORE} grandeEquipe={grandeEquipe} />
  )
}

const T = REPLAY_TEXT.fr
const FUSIL = '0xAAAA'

/** Un match à deux camps de `parCamp` places, vitalité pleine à la lecture ; Joueur0 porte un fusil. */
function documentParCamp(parCamp: number, extra: Parameters<typeof testReplayDoc>[0] = {}) {
  const n = parCamp * 2
  return testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: Array.from({ length: n }, (_, i) => ({
      xuid: `X${i}`, filmIndex: i, seat: i % parCamp, name: `Joueur${i}`, team: i < parCamp ? 0 : 1,
    })),
    tracks: Array.from({ length: n }, (_, i) => ({
      slot: 512 + i, team: i < parCamp ? 0 : 1, xuid: `X${i}`, startFrame: 0, endFrame: 290, points: [{ t: 0, x: i, y: i, sh: 0.4, hp: 0.7 }],
    })),
    weaponLabels: { [FUSIL]: { fr: 'Fusil', en: 'Rifle' } },
    loadouts: [{ t: 0, slot: 512, w: [FUSIL] }],
    inventory: [{ t: 0, slot: 512, d: 0, am: [{ mag: 30, res: 90 }], g: [2, 0] }],
    ...extra,
  })
}

const feuille = (n: number) => Array.from({ length: n }, (_, i) => scoreboardRow(`X${i}`, `Joueur${i}`, i < n / 2 ? 'a' : 'b'))

const boutons = (c: HTMLElement) => [...c.querySelectorAll('[data-replay-player-card]')] as HTMLElement[]
const camps = (c: HTMLElement) => [...c.querySelectorAll('section')] as HTMLElement[]
const carte = (nom: string) => screen.getByRole('button', { name: `Ouvrir la fiche de ${nom}` })

describe('ReplayTeamsRow — 4 contre 4', () => {
  it('camp de gauche, score, camp de droite, dans cet ordre, une fiche par place', () => {
    const vue = render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} />)
    const rangee = vue.container.querySelector('[data-replay-teams-row]') as HTMLElement
    expect(rangee.getAttribute('data-replay-teams-row')).toBe('fiches')
    const [gauche, centre, droite] = [...rangee.children] as HTMLElement[]
    expect(gauche.tagName).toBe('SECTION')
    expect(centre.hasAttribute('data-replay-teams-row-score')).toBe(true)
    expect(droite.tagName).toBe('SECTION')
    expect(within(centre).getByTestId('score')).toBeTruthy()
    expect(boutons(gauche).map((b) => b.getAttribute('aria-label'))).toEqual(
      [0, 1, 2, 3].map((i) => `Ouvrir la fiche de Joueur${i}`),
    )
    expect(boutons(droite)).toHaveLength(4)
    expect(boutons(vue.container)).toHaveLength(8)
  })

  it('les fiches se partagent la largeur à parts égales : colonnes de grille en style, l’écart et le score de la rangée', () => {
    const vue = render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} />)
    const rangee = vue.container.querySelector('[data-replay-teams-row]') as HTMLElement
    expect(rangee.style.gridTemplateColumns).toBe(`minmax(0, 1fr) ${COTES_RANGEE.scoreCellW}px minmax(0, 1fr)`)
    expect(rangee.style.columnGap).toBe(`${COTES_RANGEE.ecartFiches}px`)
    for (const camp of camps(vue.container)) {
      expect(camp.style.gridTemplateColumns).toBe('repeat(4, minmax(0, 1fr))')
      expect(camp.style.columnGap).toBe(`${COTES_RANGEE.ecartFiches}px`)
      expect(camp.className).toContain('min-w-0')
    }
  })

  it('la fiche garde l’arme en main (gabarit compact) et ses jauges', () => {
    render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} />)
    const premier = carte('Joueur0')
    expect(within(premier).getByLabelText('Bouclier')).toBeTruthy()
    expect(within(premier).getByLabelText('Santé')).toBeTruthy()
    expect(within(premier).getByText('Fusil')).toBeTruthy()
  })

  it('le camp allié est à gauche, même quand le film le désigne en second', () => {
    const allegiance: FilmAllegiance = { ...NO_ALLEGIANCE, ofTeam: (team) => (team == null ? null : team === 1) }
    const vue = render(
      <Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} allegiance={allegiance} />,
    )
    const [gauche, droite] = camps(vue.container)
    expect(boutons(gauche)[0].getAttribute('aria-label')).toBe('Ouvrir la fiche de Joueur4')
    expect(boutons(droite)[0].getAttribute('aria-label')).toBe('Ouvrir la fiche de Joueur0')
  })

  it('un clic sur une fiche rend la page avec la clé du joueur', () => {
    const onCard = vi.fn()
    render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={onCard} />)
    fireEvent.click(carte('Joueur5'))
    expect(onCard).toHaveBeenCalledWith('X5')
  })

  it('sans geste de clic, les fiches sont inertes', () => {
    const vue = render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" />)
    expect(boutons(vue.container)).toHaveLength(0)
    expect(screen.queryAllByRole('button')).toEqual([])
  })

  it('rien n’est écrit sous le score : la cellule centrale ne porte que lui', () => {
    const vue = render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} />)
    const centre = vue.container.querySelector('[data-replay-teams-row-score]') as HTMLElement
    expect([...centre.children]).toHaveLength(1)
    expect(centre.firstElementChild).toBe(screen.getByTestId('score'))
  })
})

describe('ReplayTeamsRow — joueur suivi, joueur ouvert, joueur mort', () => {
  it('le joueur suivi a la bordure de suivi, les autres la bordure ordinaire', () => {
    render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} watched="X2" />)
    expect(carte('Joueur2').style.borderColor).toBe('var(--ring)')
    expect(carte('Joueur1').style.borderColor).toBe('var(--border)')
  })

  it('la fiche ouverte est pressée', () => {
    render(<Rangee doc={documentParCamp(4)} scoreboard={feuille(8)} frame={10} locale="fr" onCard={vi.fn()} selectedXuid="X6" />)
    expect(carte('Joueur6').getAttribute('aria-pressed')).toBe('true')
    expect(carte('Joueur0').getAttribute('aria-pressed')).toBe('false')
  })

  it('un joueur mort garde sa fiche : l’encadré éliminé prend la place des jauges, comme sur la fiche seule', () => {
    const doc = documentParCamp(4, {
      tracks: Array.from({ length: 8 }, (_, i) => ({
        slot: 512 + i, team: i < 4 ? 0 : 1, xuid: `X${i}`, startFrame: 0, endFrame: i === 1 ? 100 : 290, points: [{ t: 0, x: i, y: i, sh: 1, hp: 1 }],
      })),
    })
    const vue = render(<Rangee doc={doc} scoreboard={feuille(8)} frame={140} locale="fr" onCard={vi.fn()} />)
    const mort = carte('Joueur1')
    expect(within(mort).getByText(T.goneLabel)).toBeTruthy()
    expect(within(mort).queryByLabelText('Bouclier')).toBeNull()
    expect(within(mort).queryByLabelText('Santé')).toBeNull()
    expect(vue.container.querySelectorAll('[aria-label="Bouclier"]')).toHaveLength(7)
  })
})

describe('ReplayTeamsRow — places en attente', () => {
  /** La place 0 : Partant jusqu'à 50, VIDE de 60 à 99, puis Arrivant dès 100, qui n'apparaît qu'à 120 ; la place 1, Titulaire. */
  function documentDesPlaces() {
    const vie = (xuid: string, slot: number, team: number, debut: number, fin: number) => ({
      slot, team, xuid, startFrame: debut, endFrame: fin, points: [{ t: debut, x: 0, y: 0 }],
    })
    return testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0, presence: [{ from: 0, to: 50, toMax: 59 }] },
        { xuid: 'A', filmIndex: 9, seat: 0, seatSource: 'apparie', name: 'Arrivant', team: 0, presence: [{ from: 100, to: 199 }] },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 199 }] },
        { xuid: 'E', filmIndex: 2, seat: 2, seatSource: 'lu', name: 'Eclat', team: 1, presence: [{ from: 0, to: 199 }] },
      ],
      tracks: [vie('P', 512, 0, 0, 50), vie('A', 530, 0, 120, 199), vie('T', 513, 0, 0, 199), vie('E', 514, 1, 0, 199)],
    })
  }

  it('place libre : la boîte d’attente sans nom, la colonne tenue ; occupant pas encore apparu : son nom, sans jauges', () => {
    const vue = render(<Rangee doc={documentDesPlaces()} scoreboard={[]} frame={80} locale="fr" onCard={vi.fn()} />)
    expect(screen.getByText(T.seatVacant)).toBeTruthy()
    expect(screen.queryByText('Partant')).toBeNull()
    expect(boutons(vue.container).map((b) => b.getAttribute('aria-label'))).toEqual([
      'Ouvrir la fiche de Titulaire', 'Ouvrir la fiche de Eclat',
    ])
    vue.unmount()
    const apres = render(<Rangee doc={documentDesPlaces()} scoreboard={[]} frame={110} locale="fr" onCard={vi.fn()} />)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(boutons(apres.container).map((b) => b.getAttribute('aria-label'))).not.toContain('Ouvrir la fiche de Arrivant')
  })

  it('le camp à moins de places garde les colonnes de l’autre : les deux camps ont le même nombre de colonnes', () => {
    const vue = render(<Rangee doc={documentDesPlaces()} scoreboard={[]} frame={110} locale="fr" onCard={vi.fn()} />)
    for (const camp of camps(vue.container)) expect(camp.style.gridTemplateColumns).toBe('repeat(2, minmax(0, 1fr))')
  })
})

describe('ReplayTeamsRow — Grande équipe', () => {
  it('12 contre 12 : six emplacements de deux fiches empilées par camp, le score au centre', () => {
    const vue = render(<Rangee doc={documentParCamp(12)} scoreboard={[]} frame={10} locale="fr" onCard={vi.fn()} grandeEquipe />)
    const rangee = vue.container.querySelector('[data-replay-teams-row]') as HTMLElement
    expect(rangee.getAttribute('data-replay-teams-row')).toBe('grande-equipe')
    expect([...rangee.children].map((c) => c.tagName)).toEqual(['SECTION', 'DIV', 'SECTION'])
    for (const camp of camps(vue.container)) {
      expect(camp.style.gridTemplateColumns).toBe('repeat(6, minmax(0, 1fr))')
      expect(camp.children).toHaveLength(COTES_RANGEE_GRANDE_EQUIPE.emplacementsParCamp)
      for (const emplacement of [...camp.children] as HTMLElement[]) {
        expect(boutons(emplacement)).toHaveLength(COTES_RANGEE_GRANDE_EQUIPE.fichesParEmplacement)
        expect(emplacement.style.gap).toBe(`${COTES_RANGEE_GRANDE_EQUIPE.ecartEmplacementPx}px`)
        expect(emplacement.className).toContain('flex-col')
      }
    }
    expect(boutons(vue.container)).toHaveLength(24)
    expect(within(rangee).getByTestId('score')).toBeTruthy()
  })

  it('la fiche compacte n’a ni frags, morts, assistances ni arme', () => {
    const vue = render(<Rangee doc={documentParCamp(12)} scoreboard={feuille(24)} frame={10} locale="fr" onCard={vi.fn()} grandeEquipe />)
    expect(vue.container.textContent).not.toMatch(/\d\/\d\/\d/)
    expect(within(carte('Joueur0')).queryByText('Fusil')).toBeNull()
    expect(within(carte('Joueur0')).getByLabelText('Bouclier')).toBeTruthy()
  })

  it('7 contre 7 : quatre emplacements par camp, le dernier n’a qu’une fiche', () => {
    const vue = render(<Rangee doc={documentParCamp(7)} scoreboard={[]} frame={10} locale="fr" onCard={vi.fn()} grandeEquipe />)
    for (const camp of camps(vue.container)) {
      expect(camp.style.gridTemplateColumns).toBe('repeat(4, minmax(0, 1fr))')
      const emplacements = [...camp.children] as HTMLElement[]
      expect(emplacements.map((e) => boutons(e).length)).toEqual([2, 2, 2, 1])
    }
    expect(boutons(vue.container)).toHaveLength(14)
  })

  it('un clic sur une fiche compacte rend la page avec la clé du joueur ; le joueur suivi a la bordure de suivi', () => {
    const onCard = vi.fn()
    render(<Rangee doc={documentParCamp(12)} scoreboard={[]} frame={10} locale="fr" onCard={onCard} watched="X3" grandeEquipe />)
    fireEvent.click(carte('Joueur13'))
    expect(onCard).toHaveBeenCalledWith('X13')
    expect(carte('Joueur3').style.borderColor).toBe('var(--ring)')
  })
})
