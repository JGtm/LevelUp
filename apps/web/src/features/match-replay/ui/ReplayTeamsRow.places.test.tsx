/**
 * Tests — la rangée des équipes (`ReplayTeamsRow`, forme « fiches ») et la RÈGLE DES PLACES.
 *
 * Ce que la rangée doit rendre d'une place, image par image, quand le document publie la présence de ses
 * occupants (schéma 69) : la fiche de l'occupant présent ; « pas encore apparu » sous son nom quand il tient
 * la place sans corps (Q21) ; la place VIDE, sans aucun nom, entre un partant et son remplaçant (Q20). Un
 * joueur parti n'est jamais affiché, et la rangée garde une tuile par place — ni plus, ni moins. Une place qui
 * ne rend rien à cette image garde sa colonne (une cellule vide), sans quoi les suivantes glisseraient.
 *
 * La rangée n'admet que deux camps : chaque document pose un titulaire au camp d'en face (`Eclat`, camp 1).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { ReplayRowFromBoard } from '../test/replayRowFromBoard'
import { REPLAY_TEXT } from '../i18n/i18n'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Une vie du joueur `xuid` sur [debut, fin]. */
function vie(xuid: string, slot: number, debut: number, fin: number, team = 0) {
  return { slot, team, xuid, startFrame: debut, endFrame: fin, points: [{ t: debut, x: 0, y: 0 }] }
}

const ECLAT = { xuid: 'E', filmIndex: 2, seat: 2, seatSource: 'lu', name: 'Eclat', team: 1, presence: [{ from: 0, to: 199 }] }

/**
 * Le document : la place 0 tenue par le Partant (certain jusqu'à 50, peut-être jusqu'à 59), VIDE de 60 à 99,
 * puis tenue par l'Arrivant dès 100 — qui n'apparaît qu'à 120. La place 1 est tenue d'un bout à l'autre par
 * le Titulaire ; Eclat tient le camp d'en face.
 */
function documentDesPlaces() {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    originMs: 0,
    roster: [
      { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0, presence: [{ from: 0, to: 50, toMax: 59 }] },
      { xuid: 'A', filmIndex: 9, seat: 0, seatSource: 'apparie', name: 'Arrivant', team: 0, presence: [{ from: 100, to: 199 }] },
      { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 199 }] },
      ECLAT,
    ],
    tracks: [vie('P', 512, 0, 50), vie('A', 530, 120, 199), vie('T', 513, 0, 199), vie('E', 514, 0, 199, 1)],
  })
}

type Vue = ReturnType<typeof render>

function rangee(frame: number, locale: 'fr' | 'en' = 'fr') {
  return render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={documentDesPlaces()} scoreboard={[]} frame={frame} locale={locale} />)
}

const camps = (c: HTMLElement) => [...c.querySelectorAll('section')] as HTMLElement[]
/** Les cellules des camps : une par place, vide (`aria-hidden`) quand la place ne rend rien à cette image. */
const cellules = (c: HTMLElement) => camps(c).flatMap((camp) => [...camp.children] as HTMLElement[])
/** Les tuiles rendues : les cellules qui ne sont pas la cellule vide d'une place sans rendu. */
const tuiles = (c: HTMLElement) => cellules(c).filter((e) => !e.hasAttribute('aria-hidden'))

describe('ReplayTeamsRow — la règle des places', () => {
  it('le partant tient sa fiche dans sa présence', () => {
    rangee(55)
    expect(screen.getByText('Partant')).toBeTruthy()
    expect(screen.queryByText(REPLAY_TEXT.fr.seatVacant)).toBeNull()
  })

  it('Q20 : entre le partant et son remplaçant, la place est VIDE — et le parti n’est plus nommé', () => {
    rangee(80)
    expect(screen.getByText(REPLAY_TEXT.fr.seatVacant)).toBeTruthy()
    expect(screen.getByTitle(REPLAY_TEXT.fr.seatVacantHint)).toBeTruthy()
    expect(screen.queryByText('Partant')).toBeNull()
    expect(screen.queryByText('Arrivant')).toBeNull()
    expect(screen.getByText('Titulaire')).toBeTruthy()
  })

  it('Q21 : le remplaçant tient la place avant sa première apparition, « pas encore apparu »', () => {
    rangee(110)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.getByText(REPLAY_TEXT.fr.seatNotSpawned)).toBeTruthy()
    expect(screen.getByTitle(REPLAY_TEXT.fr.seatNotSpawnedHint)).toBeTruthy()
  })

  it('une fois apparu, sa fiche remplace la tuile d’attente', () => {
    rangee(150)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.queryByText(REPLAY_TEXT.fr.seatNotSpawned)).toBeNull()
    expect(screen.queryByText('Partant')).toBeNull()
  })

  it('les deux tuiles parlent anglais en anglais', () => {
    const vue = rangee(80, 'en')
    expect(vue.getByText('Open slot')).toBeTruthy()
    vue.rerender(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={documentDesPlaces()} scoreboard={[]} frame={110} locale="en" />)
    expect(vue.getByText('Not spawned yet')).toBeTruthy()
  })

  it('une cellule par place, à chaque image : jamais plus de tuiles que de places', () => {
    for (const frame of [0, 55, 80, 110, 150, 199]) {
      const vue = rangee(frame)
      // Trois places (Partant/Arrivant, Titulaire, Eclat) : le nombre de cellules est celui des places.
      expect(cellules(vue.container), `image ${frame}`).toHaveLength(3)
      expect(tuiles(vue.container), `image ${frame}`).toHaveLength(3)
      vue.unmount()
    }
  })
})

/**
 * UNE SECTION SANS ÉQUIPE N'EXISTE PAS. Le témoin `43716616` réduit à sa place 5 : Joueur900007 part (certain
 * jusqu'à 118, peut-être là jusqu'à 317), le bot « 343 Sandwolf » est déclaré de 248 à 281 SANS équipe écrite
 * par le film, Joueur9000002 arrive à 318 ; un titulaire tient le camp d'en face. La feuille connaît le bot :
 * elle ne le range pas.
 */
describe('ReplayTeamsRow — aucune section sans équipe (témoin 43716616)', () => {
  function document43716616() {
    return testReplayDoc({
      frameCount: 600,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'S', filmIndex: 5, seat: 5, seatSource: 'lu', name: 'Joueur900007', team: 0, presence: [{ from: 0, to: 118, toMax: 317 }] },
        { xuid: '', bot: true, filmIndex: 8, seat: 8, seatSource: 'index', name: '343 Sandwolf [bot]', presence: [{ from: 248, to: 281 }] },
        { xuid: 'K', filmIndex: 9, seat: 5, seatSource: 'tirs', name: 'Joueur9000002', team: 0, presence: [{ from: 318, to: 599 }] },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 1, presence: [{ from: 0, to: 599 }] },
      ],
      tracks: [vie('S', 512, 0, 118), { ...vie('', 540, 250, 270), bot: '343 Sandwolf [bot]' }, vie('K', 530, 330, 599), vie('T', 513, 0, 599, 1)],
    })
  }
  const feuille = [
    scoreboardRow('S', 'Joueur900007', 't0'),
    scoreboardRow('bid(44.0)', '343 Sandwolf', 't0', { is_bot: true }),
    scoreboardRow('K', 'Joueur9000002', 't0'),
    scoreboardRow('T', 'Titulaire', 't1'),
  ]

  for (const locale of ['fr', 'en'] as const) {
    it(`deux camps à chaque image, jamais un troisième, et jamais « sans équipe » (${locale})`, () => {
      for (const frame of [0, 200, 250, 260, 281, 300, 318, 400]) {
        const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={document43716616()} scoreboard={feuille} frame={frame} locale={locale} />)
        expect(camps(vue.container).length, `image ${frame}`).toBe(2)
        expect(cellules(vue.container), `image ${frame}`).toHaveLength(2)
        expect(vue.container.textContent, `image ${frame}`).not.toMatch(/Sans équipe|No team|Sandwolf/)
        vue.unmount()
      }
    })
  }

  /**
   * LE TÉMOIN `859da825` : « 343 Forge Lord » (bot, index 8, sans équipe) puis Joueur90000000a (humain,
   * `seatSource: index`, équipe 0) jusqu'à la fin. La provenance `index` ne retire aucune tuile : seul le
   * silence du film sur l'équipe en retire une.
   */
  it('témoin 859da825 : l’entrée `index` AVEC équipe reste rendue, la muette jamais', () => {
    const doc = testReplayDoc({
      frameCount: 600,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'T', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 599 }] },
        { xuid: '', bot: true, filmIndex: 8, seat: 8, seatSource: 'index', name: '343 Forge Lord [bot]', presence: [{ from: 170, to: 182 }] },
        { xuid: 'SC', filmIndex: 9, seat: 9, seatSource: 'index', name: 'Joueur90000000a', team: 0, presence: [{ from: 183, to: 599 }] },
        { xuid: 'E', filmIndex: 2, seat: 2, seatSource: 'lu', name: 'Eclat', team: 1, presence: [{ from: 0, to: 599 }] },
      ],
      tracks: [vie('T', 512, 0, 599), { ...vie('', 540, 171, 181), bot: '343 Forge Lord [bot]' }, vie('SC', 530, 190, 599), vie('E', 514, 0, 599, 1)],
    })
    const feuille = [
      scoreboardRow('T', 'Titulaire', 't0'),
      scoreboardRow('bid(7.0)', '343 Forge Lord', 't0', { is_bot: true }),
      scoreboardRow('SC', 'Joueur90000000a', 't0'),
      scoreboardRow('E', 'Eclat', 't1'),
    ]
    const pendantLeBot = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={feuille} frame={175} locale="fr" />)
    expect(pendantLeBot.queryByText('343 Forge Lord')).toBeNull()
    expect(tuiles(pendantLeBot.container)).toHaveLength(3) // la place de Joueur90000000a attend, libre
    pendantLeBot.unmount()
    const apres = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={feuille} frame={300} locale="fr" />)
    expect(apres.getByText('Joueur90000000a')).toBeTruthy()
    expect(camps(apres.container)).toHaveLength(2)
  })

  it('les camps portent les noms de la feuille : Eagle, puis Cobra', () => {
    const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={document43716616()} scoreboard={feuille} frame={260} locale="fr" />)
    expect(camps(vue.container).map((c) => c.getAttribute('aria-label'))).toEqual(['Équipe Eagle', 'Équipe Cobra'])
    // Pendant la déclaration du bot, la place 5 reste à Joueur900007 (jusqu'à son `toMax`).
    expect(vue.getByText('Joueur900007')).toBeTruthy()
  })
})

describe('ReplayTeamsRow — ce qu’une place ne rend PAS', () => {
  /**
   * Un bot que ses vies nomment sans entrée de roster (`c75f33b8`, `343 Robot Hoida`, que le fil des éliminations
   * n'épingle pas) avait une place à lui, VIDE tout le match hors de ses vies — une tuile de plus que de places,
   * et sans équipe un camp de plus. Il n'a AUCUNE place.
   */
  it('un joueur sans entrée de roster ne rend aucune tuile', () => {
    const doc = testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0, presence: [{ from: 0, to: 199 }] },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 199 }] },
        ECLAT,
      ],
      tracks: [vie('P', 512, 0, 199), vie('T', 513, 0, 199), vie('E', 514, 0, 199, 1), { ...vie('', 530, 120, 180), bot: 'Robot' }],
    })
    for (const frame of [0, 80, 110, 150, 190]) {
      const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={[]} frame={frame} locale="fr" />)
      // Deux camps : le bot sans équipe n'en ouvre pas un troisième (c75f33b8).
      expect(camps(vue.container).length, `image ${frame}`).toBe(2)
      expect(tuiles(vue.container), `image ${frame}`).toHaveLength(3)
      expect(vue.queryByText(REPLAY_TEXT.fr.seatVacant), `image ${frame}`).toBeNull()
      vue.unmount()
    }
  })

  /**
   * Sur un document qui ne publie AUCUNE présence (artefact antérieur), la place d'un joueur s'affichait
   * « libre » pendant tout le préambule, avant sa première vie. Rien n'y dit qui était là au coup d'envoi : la
   * place ne rend rien avant son premier occupant — et garde sa cellule vide, pour que les suivantes ne glissent pas.
   */
  it('document sans présence : aucune place « libre » avant le premier occupant', () => {
    const doc = testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0 },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0 },
        { xuid: 'E', filmIndex: 2, seat: 2, seatSource: 'lu', name: 'Eclat', team: 1 },
      ],
      tracks: [vie('P', 512, 30, 199), vie('T', 513, 0, 199), vie('E', 514, 0, 199, 1)],
    })
    const avant = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    expect(avant.queryByText(REPLAY_TEXT.fr.seatVacant)).toBeNull()
    expect(cellules(avant.container)).toHaveLength(3)
    expect(tuiles(avant.container)).toHaveLength(2)
    avant.unmount()
    const apres = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={[]} frame={40} locale="fr" />)
    expect(tuiles(apres.container)).toHaveLength(3)
    apres.unmount()
  })
})

/**
 * LES TUILES D'ATTENTE ONT LA BOÎTE D'UNE FICHE (retour utilisateur du 2026-10-06 : la tuile « pas encore
 * apparu » n'avait ni la largeur ni la hauteur d'une fiche, et toute la colonne sautait). Les trois sortes de
 * tuile — la fiche d'un joueur présent, la place libre (Q20), l'occupant pas encore apparu (Q21) — ont la même
 * classe de boîte, la même ligne du nom et le même corps à hauteur fixe ; seul l'habit tireté distingue les
 * tuiles d'attente.
 */
describe('ReplayTeamsRow — les tuiles d’attente ont la boîte d’une fiche', () => {
  /** À l'image 110 : Présent est en vie, la place 1 est libre, Attente (un bot) n'a pas encore de corps. */
  function documentTroisTuiles() {
    return testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Présent', team: 0, presence: [{ from: 0, to: 199 }] },
        { xuid: 'D', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Parti', team: 0, presence: [{ from: 0, to: 50, toMax: 59 }] },
        { xuid: 'R', filmIndex: 9, seat: 1, seatSource: 'apparie', name: 'Remplaçant', team: 0, presence: [{ from: 150, to: 199 }] },
        { xuid: '', bot: true, filmIndex: 2, seat: 2, seatSource: 'lu', name: 'Attente [bot]', team: 0, presence: [{ from: 100, to: 199 }] },
        { xuid: 'E', filmIndex: 3, seat: 3, seatSource: 'lu', name: 'Eclat', team: 1, presence: [{ from: 0, to: 199 }] },
      ],
      tracks: [
        vie('P', 512, 0, 199),
        vie('D', 513, 0, 50),
        vie('R', 514, 160, 199),
        { ...vie('', 515, 130, 199), bot: 'Attente [bot]' },
        vie('E', 516, 0, 199, 1),
      ],
    })
  }

  it('même boîte, même ligne du nom, même corps fixe pour les trois sortes de tuile', () => {
    const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={documentTroisTuiles()} scoreboard={[]} frame={110} locale="fr" />)
    const camp = camps(vue.container)[0]
    const lesTuiles = [...camp.children] as HTMLElement[]
    expect(lesTuiles).toHaveLength(3)
    // Les trois sortes sont là, chacune sur sa place.
    const fiche = vue.getByText('Présent').parentElement!.parentElement as HTMLElement
    const libre = vue.getByText(REPLAY_TEXT.fr.seatVacant).closest('[title]') as HTMLElement
    const attente = vue.getByText('Attente').parentElement!.parentElement as HTMLElement
    expect(lesTuiles[0]).toBe(fiche)
    expect(lesTuiles[1]).toBe(libre)
    expect(lesTuiles[2]).toBe(attente)
    expect(libre.title).toBe(REPLAY_TEXT.fr.seatVacantHint)
    expect(attente.title).toBe(REPLAY_TEXT.fr.seatNotSpawnedHint)
    // LA BOÎTE : la classe exacte de la fiche, sur les trois.
    const boite = 'relative flex shrink-0 flex-col rounded-md border px-1.5 py-1.5'
    for (const t of lesTuiles) expect(t.className).toBe(boite)
    // LE CORPS À HAUTEUR FIXE : un seul par tuile, la même classe sur les trois.
    const corps = lesTuiles.map((t) => [...t.children].filter((c) => c.className === 'relative mt-[3px] h-[31px] overflow-hidden'))
    expect(corps.map((c) => c.length)).toEqual([1, 1, 1])
    // LA LIGNE DU NOM : la même classe, et un nom à la classe de celui d'une fiche.
    const lignes = [vue.getByText('Présent'), libre.firstElementChild!.firstElementChild!, vue.getByText('Attente')]
    expect(new Set(lignes.map((n) => n.parentElement!.className)).size).toBe(1)
    expect(new Set(lignes.map((n) => n.className.replace('text-foreground', 'text-muted-foreground'))).size).toBe(1)
    // L'HABIT seul distingue les tuiles d'attente : la bordure tiretée, aucun littéral de couleur.
    expect([libre.style.borderStyle, attente.style.borderStyle]).toEqual(['dashed', 'dashed'])
    expect(fiche.style.borderStyle).toBe('')
    expect(libre.style.borderColor).toBe('var(--border)')
  })

  it('le nom de l’occupant pas encore apparu s’écrit comme sur sa fiche : sans le suffixe « [bot] »', () => {
    const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={documentTroisTuiles()} scoreboard={[]} frame={110} locale="fr" />)
    expect(vue.getByText('Attente')).toBeTruthy()
    expect(vue.container.textContent).not.toContain('[bot]')
  })
})

/**
 * LES SIÈGES RELAYÉS : le compte de places ne bouge pas, la fiche suit l'occupant. LE RELAIS VIENT DU
 * DOCUMENT : Nord12 et Zulu portent le MÊME `seat`, et c'est le film qui le dit — le web ne l'apparie plus sur
 * la participation API. Les deux présences sont disjointes : Nord12 sort à l'image 100, Zulu entre à l'image 100.
 */
describe('ReplayTeamsRow — sièges relayés dans une Grande équipe', () => {
  const noms = (prefixe: string) => Array.from({ length: 12 }, (_, i) => `${prefixe}${String(i + 1).padStart(2, '0')}`)
  const NORD = noms('Nord')
  const SUD = noms('Sud')

  it('le compte de sièges ne bouge pas (24), la fiche suit l’occupant', () => {
    const tous = [...NORD, ...SUD, 'Zulu']
    const doc = testReplayDoc({
      frameCount: 300,
      frameIntervalMs: 100,
      originMs: 0,
      roster: tous.map((nom, i) => ({
        xuid: nom,
        filmIndex: i,
        name: nom,
        team: nom.startsWith('Sud') ? 1 : 0,
        seat: nom === 'Zulu' ? 11 : i,
        seatSource: nom === 'Zulu' ? 'apparie' : 'lu',
      })),
      tracks: tous.map((nom, i) => ({
        slot: 512 + i,
        team: -1,
        xuid: nom,
        // Nord12 (siège 11) cesse d'apparaître à l'image 100 ; Zulu y entre.
        startFrame: nom === 'Zulu' ? 100 : 0,
        endFrame: nom === 'Nord12' ? 99 : 300,
        points: [{ t: nom === 'Zulu' ? 100 : 0, x: 1, y: 1, sh: 0.7, hp: 1 }],
      })),
    })
    const board = [...NORD.map((n) => scoreboardRow(n, n, 't0')), ...SUD.map((n) => scoreboardRow(n, n, 't1'))]
    const arbre = (frame: number) => <ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={board} frame={frame} locale="fr" />
    const corps = (vue: Vue) => vue.container.querySelectorAll('.h-\\[31px\\]').length
    const vue = render(arbre(50))
    // Avant le relais : le partant tient la fiche, le remplaçant n'est nulle part.
    expect(corps(vue)).toBe(24)
    expect(vue.getByText('Nord12')).toBeTruthy()
    expect(vue.queryByText('Zulu')).toBeNull()
    expect(cellules(vue.container)).toHaveLength(24)
    // Après : le même siège, le même compte, l'autre occupant.
    vue.rerender(arbre(150))
    expect(corps(vue)).toBe(24)
    expect(vue.getByText('Zulu')).toBeTruthy()
    expect(vue.queryByText('Nord12')).toBeNull()
    expect(cellules(vue.container)).toHaveLength(24)
  })

  it('effectifs inégaux (12 contre 9) : la rangée garde les colonnes du plus grand camp, toutes les fiches au même gabarit', () => {
    const sud = SUD.slice(0, 9)
    const tous = [...NORD, ...sud]
    const doc = testReplayDoc({
      frameCount: 300,
      frameIntervalMs: 100,
      originMs: 0,
      roster: tous.map((nom, i) => ({ xuid: nom, filmIndex: i, name: nom, team: nom.startsWith('Sud') ? 1 : 0 })),
      tracks: tous.map((nom, i) => ({ slot: 512 + i, team: -1, xuid: nom, startFrame: 0, endFrame: 300, points: [{ t: 0, x: 1, y: 1, sh: 0.7, hp: 1 }] })),
    })
    const board = [...NORD.map((n) => scoreboardRow(n, n, 't0')), ...sud.map((n) => scoreboardRow(n, n, 't1'))]
    const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={board} frame={50} locale="fr" />)
    const [nord, sudCamp] = camps(vue.container)
    expect(nord.children).toHaveLength(12)
    expect(sudCamp.children).toHaveLength(9)
    expect(nord.style.gridTemplateColumns).toBe('repeat(12, minmax(0, 1fr))')
    expect(sudCamp.style.gridTemplateColumns).toBe('repeat(12, minmax(0, 1fr))')
    expect(vue.container.querySelectorAll('.h-\\[31px\\]')).toHaveLength(21)
    expect(vue.container.querySelectorAll('.rounded-md.border')).toHaveLength(21)
  })
})
