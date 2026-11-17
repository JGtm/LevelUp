/**
 * Tests — la colonne des équipes en LIGNES (`ReplayRosterColumn`, `ReplayRosterRow`, son dépli).
 *
 * Ce qu'ils protègent : une ligne par PLACE (jamais plus de lignes que de places), le joueur mort qui
 * écrit son retour à la place des barres, les places en attente (même règle que la tuile d'attente),
 * le dépli du seul joueur suivi, la lecture ancienne qui pâlit, la part des équipes qui défile, et le
 * clic qui rend la page avec la bonne clé — au contrat des fiches (`data-replay-player-card`, gardes
 * d'appui).
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayRosterColumn } from './ReplayRosterColumn'
import { GABARIT_LIGNE } from '../model/cardGabarit'
import { trackedPlayerOf } from '../model/playerPanelLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

type Props = Omit<ComponentProps<typeof ReplayRosterColumn>, 'players' | 'tracked' | 'allegiance'> & {
  scoreboard: MatchScoreboardRow[]
  /** Le joueur regardé par la page (clé de feuille ou de roster) ; `null` = personne. */
  watched?: string | null
}

/** La colonne montée depuis une feuille, comme la page : jointure film <-> base, puis joueur suivi retrouvé au roster. */
function Colonne({ scoreboard, watched = null, ...rest }: Props) {
  const players = useMemo(() => buildPlayers(rest.doc, scoreboard), [rest.doc, scoreboard])
  const tracked = useMemo(() => trackedPlayerOf(watched, players), [watched, players])
  return <ReplayRosterColumn {...rest} players={players} tracked={tracked} allegiance={NO_ALLEGIANCE} />
}

const NOMS = ['Alpha', 'Bravo', 'Charlie', 'Delta', 'Echo', 'Foxtrot', 'Golf', 'Hotel']
const XUIDS = ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H']
const FUSIL = '0xAAAA'
const PISTOLET = '0xBBBB'

/** Une vie du joueur `i` : slot 512 + i, vitalité lue au départ (bouclier 0,4, santé 0,7). */
function vie(i: number, debut: number, fin: number, slot = 512 + i) {
  return {
    slot, team: i < 4 ? 0 : 1, xuid: XUIDS[i], startFrame: debut, endFrame: fin,
    points: [{ t: debut, x: i, y: i, sh: 0.4, hp: 0.7 }],
  }
}

/** Un 4 contre 4 : Alpha porte un fusil et un pistolet, deux types de grenade, un grappin à une charge. */
function document4v4(extra: Parameters<typeof testReplayDoc>[0] = {}) {
  return testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: NOMS.map((name, i) => ({ xuid: XUIDS[i], filmIndex: i, seat: i, name, team: i < 4 ? 0 : 1 })),
    tracks: NOMS.map((_, i) => vie(i, 0, 290)),
    weaponLabels: { [FUSIL]: { fr: 'Fusil', en: 'Rifle' }, [PISTOLET]: { fr: 'Pistolet', en: 'Pistol' } },
    loadouts: [{ t: 0, slot: 512, w: [FUSIL, PISTOLET] }],
    inventory: [{ t: 0, slot: 512, d: 0, am: [{ mag: 1, res: 2 }, { mag: 36, res: 108 }], g: [2, 0] }],
    grenadeLabels: [{ en: 'Frag', fr: 'Fragmentation' }, { en: 'Plasma', fr: 'Plasma' }],
    abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot', img: '/static/weapons-assets/halo_infinite/hud/Grapple.png', tinted: true } },
    abilities: [{ t: 0, slot: 512, r: 20, src: 'i48' }],
    ...extra,
  })
}

const FEUILLE = NOMS.map((n, i) => scoreboardRow(XUIDS[i], n, i < 4 ? 'a' : 'b', { kills: i, deaths: 2, assists: 1 }))
const T = REPLAY_TEXT.fr

/** Les lignes de joueur rendues : celles qui portent le repère de fiche. */
function lignes(container: HTMLElement) {
  return [...container.querySelectorAll('[data-replay-player-card]')] as HTMLElement[]
}

describe('ReplayRosterColumn — 4 contre 4', () => {
  it('deux camps, huit lignes, un seul en-tête de colonnes', () => {
    const vue = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} />)
    expect(vue.container.querySelectorAll('section')).toHaveLength(2)
    expect(lignes(vue.container)).toHaveLength(8)
    expect(screen.getAllByText(T.roster.colKda)).toHaveLength(1)
    for (const [court, entier] of [
      [T.roster.colWeapon, T.roster.colWeaponTitle],
      [T.roster.colGrenades, T.roster.colGrenadesTitle],
      [T.roster.colEquipment, T.roster.colEquipmentTitle],
      [T.roster.colKda, T.roster.colKdaTitle],
    ]) {
      expect(screen.getByText(court).getAttribute('title')).toBe(entier)
    }
  })

  it('la ligne dit le nom, les deux barres et le triplet F / M / A de la feuille', () => {
    render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} />)
    const bravo = screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' })
    expect(within(bravo).getByText('Bravo')).toBeTruthy()
    expect(within(bravo).getByLabelText('Bouclier')).toBeTruthy()
    expect(within(bravo).getByLabelText('Santé')).toBeTruthy()
    // Le bouclier est AU-DESSUS de la santé, et chacun garde sa valeur lue (0,4 et 0,7).
    expect((within(bravo).getByLabelText('Bouclier').firstElementChild as HTMLElement).style.width).toBe('40%')
    expect((within(bravo).getByLabelText('Santé').firstElementChild as HTMLElement).style.width).toBe('70%')
    expect(bravo.textContent).toContain('1/2/1')
  })

  it('la ligne porte la hauteur du gabarit et la colonne ne la comprime pas', () => {
    const vue = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} />)
    for (const l of lignes(vue.container)) expect(l.style.height).toBe(`${GABARIT_LIGNE.rowPx}px`)
  })
})

describe('ReplayRosterColumn — arme, grenades, équipement', () => {
  it('l’arme en main, le stock de la grenade désignée et la vignette de l’équipement tiennent leur colonne', () => {
    render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} />)
    const alpha = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(within(alpha).getByText('Fusil')).toBeTruthy()
    // Un seul type porté : désigné par déduction, son stock s'écrit.
    expect(within(alpha).getByTitle(/^Fragmentation ×2/)).toBeTruthy()
    expect(within(alpha).getByRole('img', { name: 'Grappin' })).toBeTruthy()
    // Un joueur sans lecture d'inventaire garde ses colonnes, vides.
    const bravo = screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' })
    expect(within(bravo).queryByRole('img', { name: 'Grappin' })).toBeNull()
  })
})

describe('ReplayRosterColumn — joueur mort', () => {
  /** Echo meurt à 100 et revient à 180 (nouvelle vie, autre slot). */
  const docMort = () =>
    document4v4({ tracks: [...NOMS.map((_, i) => (i === 4 ? vie(i, 0, 100) : vie(i, 0, 290))), vie(4, 180, 290, 600)] })

  it('« Réapparition dans … » prend la place des barres, et le nom est éteint', () => {
    const vue = render(<Colonne doc={docMort()} scoreboard={FEUILLE} frame={140} locale="fr" onCard={vi.fn()} />)
    const echo = screen.getByRole('button', { name: 'Ouvrir la fiche de Echo' })
    expect(within(echo).getByText(/^Réapparition dans \d/)).toBeTruthy()
    expect(within(echo).queryByLabelText('Bouclier')).toBeNull()
    expect(within(echo).queryByLabelText('Santé')).toBeNull()
    expect((within(echo).getByText('Echo') as HTMLElement).style.opacity).toBe(String(GABARIT_LIGNE.deadOpacity))
    // Les sept autres vivent : leurs barres sont là.
    expect(vue.container.querySelectorAll('[aria-label="Bouclier"]')).toHaveLength(7)
  })

  it('les effets de la fiche sont gardés à la hauteur de la ligne : éclat du coup fatal, puis éclat de réapparition', () => {
    const mort = render(<Colonne doc={docMort()} scoreboard={FEUILLE} frame={105} locale="fr" onCard={vi.fn()} />)
    const echo = screen.getByRole('button', { name: 'Ouvrir la fiche de Echo' })
    const couche = echo.querySelector('.replay-card-fx') as HTMLElement
    expect(couche.className).toContain('replay-flash-death')
    expect(couche.className).toContain(GABARIT_LIGNE.layerRadius)
    mort.unmount()
    render(<Colonne doc={docMort()} scoreboard={FEUILLE} frame={185} locale="fr" onCard={vi.fn()} />)
    const revenu = screen.getByRole('button', { name: 'Ouvrir la fiche de Echo' })
    expect((revenu.querySelector('.replay-card-fx') as HTMLElement).className).toContain('replay-flash-respawn')
  })

  it('sans vie suivante, le joueur ne revient plus : le retour n’est jamais deviné', () => {
    render(<Colonne doc={document4v4({ tracks: NOMS.map((_, i) => (i === 4 ? vie(i, 0, 100) : vie(i, 0, 290))) })} scoreboard={FEUILLE} frame={140} locale="fr" onCard={vi.fn()} />)
    const echo = screen.getByRole('button', { name: 'Ouvrir la fiche de Echo' })
    expect(within(echo).getByText(`${T.goneLabel} · ${T.goneValue}`)).toBeTruthy()
    expect(within(echo).queryByText(/Réapparition dans/)).toBeNull()
  })

  it('la ligne vivante n’écrit pas de retour', () => {
    render(<Colonne doc={docMort()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} />)
    expect(screen.queryByText(/Réapparition dans/)).toBeNull()
  })
})

/** La place 0 tenue par le Partant (jusqu'à 50), vide de 60 à 99, puis par l'Arrivant dès 100 — qui n'apparaît qu'à 120. */
function documentDesPlaces() {
  const vieP = (xuid: string, slot: number, debut: number, fin: number) => ({
    slot, team: 0, xuid, startFrame: debut, endFrame: fin, points: [{ t: debut, x: 0, y: 0 }],
  })
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    originMs: 0,
    roster: [
      { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0, presence: [{ from: 0, to: 50, toMax: 59 }] },
      { xuid: 'A', filmIndex: 9, seat: 0, seatSource: 'apparie', name: 'Arrivant', team: 0, presence: [{ from: 100, to: 199 }] },
      { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 199 }] },
    ],
    tracks: [vieP('P', 512, 0, 50), vieP('A', 530, 120, 199), vieP('T', 513, 0, 199)],
  })
}

describe('ReplayRosterColumn — places en attente', () => {
  const rendre = (frame: number) => render(<Colonne doc={documentDesPlaces()} scoreboard={[]} frame={frame} locale="fr" onCard={vi.fn()} />)

  it('place libre : aucun nom, son état à la place des barres, jamais cliquable', () => {
    const vue = rendre(80)
    expect(screen.getByText(T.seatVacant)).toBeTruthy()
    expect(screen.getByTitle(T.seatVacantHint)).toBeTruthy()
    expect(screen.queryByText('Partant')).toBeNull()
    expect(screen.queryByText('Arrivant')).toBeNull()
    // Une ligne de joueur (Titulaire) et une ligne d'attente : deux lignes, une place chacune.
    expect(lignes(vue.container)).toHaveLength(1)
    expect(screen.getByTitle(T.seatVacantHint).hasAttribute('data-replay-player-card')).toBe(false)
    expect(screen.getByTitle(T.seatVacantHint).getAttribute('role')).toBeNull()
  })

  it('occupant pas encore apparu : son nom et « pas encore apparu »', () => {
    rendre(110)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.getByText(T.seatNotSpawned)).toBeTruthy()
    expect(screen.getByTitle(T.seatNotSpawnedHint)).toBeTruthy()
  })

  it('apparu : sa ligne remplace la ligne d’attente ; le partant n’est jamais nommé', () => {
    const vue = rendre(150)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.queryByText(T.seatNotSpawned)).toBeNull()
    expect(screen.queryByText('Partant')).toBeNull()
    expect(lignes(vue.container)).toHaveLength(2)
  })

  it('la ligne d’attente a la hauteur d’une ligne de joueur', () => {
    rendre(80)
    expect((screen.getByTitle(T.seatVacantHint) as HTMLElement).style.height).toBe(`${GABARIT_LIGNE.rowPx}px`)
  })

  it('les deux états parlent anglais en anglais', () => {
    const vue = render(<Colonne doc={documentDesPlaces()} scoreboard={[]} frame={80} locale="en" />)
    expect(vue.getByText('Open slot')).toBeTruthy()
    vue.rerender(<Colonne doc={documentDesPlaces()} scoreboard={[]} frame={110} locale="en" />)
    expect(vue.getByText('Not spawned yet')).toBeTruthy()
  })
})

describe('ReplayRosterColumn — joueur suivi et dépli', () => {
  const rendre = (watched: string | null) =>
    render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} watched={watched} />)

  it('la ligne du joueur suivi se déplie : munitions, arme en réserve, grenades, mention « suivi »', () => {
    const vue = rendre('A')
    const depli = vue.container.querySelector('[data-replay-roster-detail]') as HTMLElement
    expect(depli).toBeTruthy()
    // Sous SA ligne, pas sous une autre.
    const alpha = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(alpha.nextElementSibling).toBe(depli)
    expect(within(depli).getByText('Fusil')).toBeTruthy()
    expect(within(depli).getByText('Pistolet')).toBeTruthy()
    expect(depli.textContent).toContain('1/2')
    expect(within(depli).getByText('Fragmentation ×2')).toBeTruthy()
    expect(within(alpha).getByText(T.roster.followed)).toBeTruthy()
    expect(alpha.querySelector('b')?.className).toContain('font-bold')
  })

  it('un seul dépli : les autres joueurs ne se déplient pas et n’ont pas de mention', () => {
    const vue = rendre('A')
    expect(vue.container.querySelectorAll('[data-replay-roster-detail]')).toHaveLength(1)
    expect(screen.getAllByText(T.roster.followed)).toHaveLength(1)
    const bravo = screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' })
    expect(bravo.nextElementSibling?.hasAttribute('data-replay-roster-detail')).toBe(false)
  })

  it('personne de suivi : aucun dépli, aucune mention', () => {
    const vue = rendre(null)
    expect(vue.container.querySelector('[data-replay-roster-detail]')).toBeNull()
    expect(screen.queryByText(T.roster.followed)).toBeNull()
  })

  it('un joueur suivi mort n’a pas de dépli', () => {
    const doc = document4v4({ tracks: NOMS.map((_, i) => (i === 0 ? vie(i, 0, 100) : vie(i, 0, 290))) })
    const vue = render(<Colonne doc={doc} scoreboard={FEUILLE} frame={140} locale="fr" onCard={vi.fn()} watched="A" />)
    expect(vue.container.querySelector('[data-replay-roster-detail]')).toBeNull()
  })
})

describe('ReplayRosterColumn — lecture ancienne', () => {
  it('une vitalité lue il y a longtemps pâlit ; une lecture fraîche garde sa pleine encre', () => {
    const frais = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={1} locale="fr" onCard={vi.fn()} />)
    const opaciteFrais = Number((frais.getAllByLabelText('Bouclier')[0] as HTMLElement).style.opacity)
    frais.unmount()
    const ancien = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={200} locale="fr" onCard={vi.fn()} />)
    const opaciteAncien = Number((ancien.getAllByLabelText('Bouclier')[0] as HTMLElement).style.opacity)
    expect(opaciteFrais).toBeGreaterThan(0.95)
    expect(opaciteAncien).toBeCloseTo(0.5, 5)
  })

  it('l’arme en main pâlit avec l’âge de sa lecture', () => {
    const ancien = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={250} locale="fr" onCard={vi.fn()} />)
    const cellule = within(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })).getByText('Fusil').closest('[title]') as HTMLElement
    expect(Number(cellule.style.opacity)).toBeCloseTo(0.5, 5)
    ancien.unmount()
  })
})

describe('ReplayRosterColumn — bot', () => {
  const docBot = () =>
    testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
        { xuid: '', bot: true, filmIndex: 2, seat: 2, name: 'Marc', team: 1 },
      ],
      tracks: [
        { slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 1, y: 1 }] },
        { slot: 514, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 3, y: 3 }] },
      ] as never,
    })

  it('un bot a sa ligne, et le clic rend sa clé de roster', () => {
    const onCard = vi.fn()
    render(<Colonne doc={docBot()} scoreboard={[scoreboardRow('A', 'Alpha', 'a')]} frame={10} locale="fr" onCard={onCard} />)
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Marc' }))
    expect(onCard).toHaveBeenCalledWith('bot:Marc')
  })
})

describe('ReplayRosterColumn — clic', () => {
  it('un clic rend la page avec la clé du joueur ; Entrée et Espace aussi', () => {
    const onCard = vi.fn()
    render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={onCard} />)
    const bravo = screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' })
    fireEvent.click(bravo)
    fireEvent.keyDown(bravo, { key: 'Enter' })
    fireEvent.keyDown(bravo, { key: ' ' })
    fireEvent.keyDown(bravo, { key: 'a' })
    expect(onCard.mock.calls).toEqual([['B'], ['B'], ['B']])
  })

  it('même contrat d’appui que les fiches : repère, focus clavier, souris sans focus, Espace gardé', () => {
    render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} selectedXuid="C" />)
    const charlie = screen.getByRole('button', { name: 'Ouvrir la fiche de Charlie' })
    expect(charlie.hasAttribute('data-replay-player-card')).toBe(true)
    expect(charlie.tabIndex).toBe(0)
    expect(charlie.getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Delta' }).getAttribute('aria-pressed')).toBe('false')
    expect(fireEvent.mouseDown(charlie)).toBe(false)
    const global = vi.fn()
    window.addEventListener('keydown', global)
    fireEvent.keyDown(charlie, { key: ' ' })
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })

  it('sans `onCard`, les lignes sont inertes', () => {
    const vue = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" />)
    expect(screen.queryAllByRole('button')).toEqual([])
    expect(lignes(vue.container)).toHaveLength(0)
  })
})

describe('ReplayRosterColumn — Grande équipe', () => {
  const NB = 24
  const docGrande = () =>
    testReplayDoc({
      frameCount: 300,
      frameIntervalMs: 100,
      originMs: 0,
      roster: Array.from({ length: NB }, (_, i) => ({ xuid: `X${i}`, filmIndex: i, seat: i % 12, name: `Joueur${i}`, team: i < 12 ? 0 : 1 })),
      tracks: Array.from({ length: NB }, (_, i) => ({
        slot: 512 + i, team: i < 12 ? 0 : 1, xuid: `X${i}`, startFrame: 0, endFrame: 290, points: [{ t: 0, x: i, y: i, sh: 1, hp: 1 }],
      })),
    })

  it('vingt-quatre lignes ordinaires dans UNE zone qui défile ; les en-têtes de camp restent collés en haut', () => {
    const vue = render(<Colonne doc={docGrande()} scoreboard={[]} frame={10} locale="fr" onCard={vi.fn()} />)
    const defile = vue.container.querySelectorAll('.overflow-y-auto')
    expect(defile).toHaveLength(1)
    expect(defile[0].hasAttribute('data-replay-roster-column')).toBe(true)
    expect(lignes(defile[0] as HTMLElement)).toHaveLength(NB)
    for (const l of lignes(vue.container)) expect(l.style.height).toBe(`${GABARIT_LIGNE.rowPx}px`)
    const entetes = vue.container.querySelectorAll('h3')
    expect(entetes).toHaveLength(2)
    for (const h of entetes) expect(h.className).toContain('sticky')
  })

  it('le même gabarit qu’un 4 contre 4 : aucune forme compacte', () => {
    const vue = render(<Colonne doc={document4v4()} scoreboard={FEUILLE} frame={10} locale="fr" onCard={vi.fn()} />)
    expect(vue.container.querySelectorAll('.overflow-y-auto')).toHaveLength(1)
    expect((lignes(vue.container)[0] as HTMLElement).style.height).toBe(`${GABARIT_LIGNE.rowPx}px`)
  })
})

describe('ReplayRosterColumn — chacun pour soi', () => {
  it('un camp sans désignateur n’a ni en-tête ni pastille', () => {
    const doc = testReplayDoc({
      frameCount: 100,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [{ xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: -1 }],
      tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 90, points: [{ t: 0, x: 0, y: 0 }] }],
    })
    const vue = render(<Colonne doc={doc} scoreboard={[]} frame={10} locale="fr" onCard={vi.fn()} />)
    expect(vue.container.querySelectorAll('h3')).toHaveLength(0)
    expect(lignes(vue.container)).toHaveLength(1)
  })
})
