/**
 * Tests — ReplayTeams et la RÈGLE DES PLACES (lot M2.4, 2026-09-23).
 *
 * Ce que la colonne doit rendre d'une place, image par image, quand le document publie la
 * présence de ses occupants (schéma 69) : la fiche de l'occupant présent ; « pas encore apparu »
 * sous son nom quand il tient la place sans corps (Q21) ; la place VIDE, sans aucun nom, entre
 * un partant et son remplaçant (Q20). Un joueur parti n'est jamais affiché, et la colonne garde
 * une tuile par place — ni plus, ni moins.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ReplayTeams } from './ReplayTeams'
import { REPLAY_TEXT } from '../i18n/i18n'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Une vie du joueur `xuid` sur [debut, fin]. */
function vie(xuid: string, slot: number, debut: number, fin: number) {
  return { slot, team: 0, xuid, startFrame: debut, endFrame: fin, points: [{ t: debut, x: 0, y: 0 }] }
}

/**
 * Le document : la place 0 tenue par le Partant (certain jusqu'à 50, peut-être jusqu'à 59),
 * VIDE de 60 à 99, puis tenue par l'Arrivant dès 100 — qui n'apparaît qu'à 120. La place 1
 * est tenue d'un bout à l'autre par le Titulaire.
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
    ],
    tracks: [vie('P', 512, 0, 50), vie('A', 530, 120, 199), vie('T', 513, 0, 199)],
  })
}

function colonne(frame: number, locale: 'fr' | 'en' = 'fr') {
  return render(<ReplayTeams doc={documentDesPlaces()} scoreboard={[]} frame={frame} locale={locale} />)
}

describe('ReplayTeams — la règle des places', () => {
  it('le partant tient sa fiche dans sa présence', () => {
    colonne(55)
    expect(screen.getByText('Partant')).toBeTruthy()
    expect(screen.queryByText(REPLAY_TEXT.fr.seatVacant)).toBeNull()
  })

  it('Q20 : entre le partant et son remplaçant, la place est VIDE — et le parti n’est plus nommé', () => {
    colonne(80)
    expect(screen.getByText(REPLAY_TEXT.fr.seatVacant)).toBeTruthy()
    expect(screen.getByTitle(REPLAY_TEXT.fr.seatVacantHint)).toBeTruthy()
    expect(screen.queryByText('Partant')).toBeNull()
    expect(screen.queryByText('Arrivant')).toBeNull()
    expect(screen.getByText('Titulaire')).toBeTruthy()
  })

  it('Q21 : le remplaçant tient la place avant sa première apparition, « pas encore apparu »', () => {
    colonne(110)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.getByText(REPLAY_TEXT.fr.seatNotSpawned)).toBeTruthy()
    expect(screen.getByTitle(REPLAY_TEXT.fr.seatNotSpawnedHint)).toBeTruthy()
  })

  it('une fois apparu, sa fiche remplace la tuile d’attente', () => {
    colonne(150)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.queryByText(REPLAY_TEXT.fr.seatNotSpawned)).toBeNull()
    expect(screen.queryByText('Partant')).toBeNull()
  })

  it('les deux tuiles parlent anglais en anglais', () => {
    const vue = colonne(80, 'en')
    expect(vue.getByText('Open slot')).toBeTruthy()
    vue.rerender(<ReplayTeams doc={documentDesPlaces()} scoreboard={[]} frame={110} locale="en" />)
    expect(vue.getByText('Not spawned yet')).toBeTruthy()
  })

  it('une tuile par place, à chaque image : jamais plus de tuiles que de places', () => {
    for (const frame of [0, 55, 80, 110, 150, 199]) {
      const vue = colonne(frame)
      // Les tuiles sont les ENFANTS DIRECTS du conteneur des places d'un camp (la colonne qui
      // défile) — le seul endroit où leur nombre se lit sans dépendre de leur contenu.
      const conteneurs = [...vue.container.querySelectorAll('.overflow-y-auto')]
      const tuiles = conteneurs.reduce((n, c) => n + c.children.length, 0)
      expect(conteneurs.length, `image ${frame}`).toBe(1)
      expect(tuiles, `image ${frame}`).toBe(2)
      vue.unmount()
    }
  })
})

/** Le nombre de tuiles que la colonne rend : les enfants directs des conteneurs de places. */
function tuilesRendues(container: HTMLElement): number {
  return [...container.querySelectorAll('.overflow-y-auto')].reduce((n, c) => n + c.children.length, 0)
}

/**
 * UNE SECTION SANS ÉQUIPE N'EXISTE PAS (décision du 2026-10-06). Le témoin `43716616` réduit à
 * sa place 5 : Slowpoke6743 part (certain jusqu'à 118, peut-être là jusqu'à 317), le bot
 * « 343 Sandwolf » est déclaré de 248 à 281 SANS équipe écrite par le film, KernelPanic10 arrive à
 * 318 ; un titulaire tient le camp d'en face. La feuille connaît le bot : elle ne le range pas.
 */
describe('ReplayTeams — aucune section sans équipe (témoin 43716616)', () => {
  function document43716616() {
    return testReplayDoc({
      frameCount: 600,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'S', filmIndex: 5, seat: 5, seatSource: 'lu', name: 'Slowpoke6743', team: 0, presence: [{ from: 0, to: 118, toMax: 317 }] },
        { xuid: '', bot: true, filmIndex: 8, seat: 8, seatSource: 'index', name: '343 Sandwolf [bot]', presence: [{ from: 248, to: 281 }] },
        { xuid: 'K', filmIndex: 9, seat: 5, seatSource: 'tirs', name: 'KernelPanic10', team: 0, presence: [{ from: 318, to: 599 }] },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 1, presence: [{ from: 0, to: 599 }] },
      ],
      tracks: [vie('S', 512, 0, 118), { ...vie('', 540, 250, 270), bot: '343 Sandwolf [bot]' }, vie('K', 530, 330, 599), vie('T', 513, 0, 599)],
    })
  }
  const feuille = [
    scoreboardRow('S', 'Slowpoke6743', 't0'),
    scoreboardRow('bid(44.0)', '343 Sandwolf', 't0', { is_bot: true }),
    scoreboardRow('K', 'KernelPanic10', 't0'),
    scoreboardRow('T', 'Titulaire', 't1'),
  ]

  for (const locale of ['fr', 'en'] as const) {
    it(`deux colonnes à chaque image, jamais une troisième, et jamais « sans équipe » (${locale})`, () => {
      for (const frame of [0, 200, 250, 260, 281, 300, 318, 400]) {
        const vue = render(<ReplayTeams doc={document43716616()} scoreboard={feuille} frame={frame} locale={locale} />)
        expect(vue.container.querySelectorAll('.overflow-y-auto').length, `image ${frame}`).toBe(2)
        expect(tuilesRendues(vue.container), `image ${frame}`).toBe(2)
        expect(vue.container.textContent, `image ${frame}`).not.toMatch(/Sans équipe|No team|Sandwolf/)
        vue.unmount()
      }
    })
  }

  /**
   * LE TÉMOIN `859da825` : « 343 Forge Lord » (bot, index 8, sans équipe) puis SplinterCell958
   * (humain, `seatSource: index`, équipe 0) jusqu'à la fin. La provenance `index` ne retire
   * aucune tuile : seul le silence du film sur l'équipe en retire une.
   */
  it('témoin 859da825 : l’entrée `index` AVEC équipe reste rendue, la muette jamais', () => {
    const doc = testReplayDoc({
      frameCount: 600,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'T', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 599 }] },
        { xuid: '', bot: true, filmIndex: 8, seat: 8, seatSource: 'index', name: '343 Forge Lord [bot]', presence: [{ from: 170, to: 182 }] },
        { xuid: 'SC', filmIndex: 9, seat: 9, seatSource: 'index', name: 'SplinterCell958', team: 0, presence: [{ from: 183, to: 599 }] },
      ],
      tracks: [vie('T', 512, 0, 599), { ...vie('', 540, 171, 181), bot: '343 Forge Lord [bot]' }, vie('SC', 530, 190, 599)],
    })
    const feuille = [
      scoreboardRow('T', 'Titulaire', 't0'),
      scoreboardRow('bid(7.0)', '343 Forge Lord', 't0', { is_bot: true }),
      scoreboardRow('SC', 'SplinterCell958', 't0'),
    ]
    const pendantLeBot = render(<ReplayTeams doc={doc} scoreboard={feuille} frame={175} locale="fr" />)
    expect(pendantLeBot.queryByText('343 Forge Lord')).toBeNull()
    expect(tuilesRendues(pendantLeBot.container)).toBe(2) // la place de SplinterCell958 attend, libre
    pendantLeBot.unmount()
    const apres = render(<ReplayTeams doc={doc} scoreboard={feuille} frame={300} locale="fr" />)
    expect(apres.getByText('SplinterCell958')).toBeTruthy()
    expect(apres.container.querySelectorAll('.overflow-y-auto')).toHaveLength(1)
    apres.unmount()
  })

  it('les colonnes portent les noms de la feuille : Eagle, puis Cobra', () => {
    const vue = render(<ReplayTeams doc={document43716616()} scoreboard={feuille} frame={260} locale="fr" />)
    expect([...vue.container.querySelectorAll('h3')].map((h) => h.textContent)).toEqual(['Équipe Eagle', 'Équipe Cobra'])
    // Pendant la déclaration du bot, la place 5 reste à Slowpoke6743 (jusqu'à son `toMax`).
    expect(vue.getByText('Slowpoke6743')).toBeTruthy()
  })
})

describe('ReplayTeams — ce qu’une place ne rend PAS (revue M2, 2026-09-24)', () => {
  /**
   * M2-R1 : un bot que ses vies nomment sans entrée de roster (`c75f33b8`, `343 Robot Hoida`,
   * que le kill-feed n'épingle pas) avait une place à lui, VIDE tout le match hors de ses vies —
   * une tuile de plus que de places, et sans équipe une colonne de plus. Il n'a AUCUNE place.
   */
  it('un joueur sans entrée de roster ne rend aucune tuile', () => {
    const doc = testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0, presence: [{ from: 0, to: 199 }] },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 199 }] },
      ],
      tracks: [vie('P', 512, 0, 199), vie('T', 513, 0, 199), { ...vie('', 530, 120, 180), bot: 'Robot' }],
    })
    for (const frame of [0, 80, 110, 150, 190]) {
      const vue = render(<ReplayTeams doc={doc} scoreboard={[]} frame={frame} locale="fr" />)
      // Une seule colonne : le bot sans équipe n'en ouvre pas une troisième (c75f33b8).
      expect(vue.container.querySelectorAll('.overflow-y-auto').length, `image ${frame}`).toBe(1)
      expect(tuilesRendues(vue.container), `image ${frame}`).toBe(2)
      expect(vue.queryByText(REPLAY_TEXT.fr.seatVacant), `image ${frame}`).toBeNull()
      vue.unmount()
    }
  })

  /**
   * M2-R7 : sur un document qui ne publie AUCUNE présence (artefact antérieur), la place d'un
   * joueur s'affichait « libre » pendant tout le préambule, avant sa première vie. Rien n'y dit qui
   * était là au coup d'envoi : la place ne rend rien avant son premier occupant.
   */
  it('document sans présence : aucune place « libre » avant le premier occupant', () => {
    const doc = testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      originMs: 0,
      roster: [
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0 },
        { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0 },
      ],
      tracks: [vie('P', 512, 30, 199), vie('T', 513, 0, 199)],
    })
    const avant = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    expect(avant.queryByText(REPLAY_TEXT.fr.seatVacant)).toBeNull()
    expect(tuilesRendues(avant.container)).toBe(1)
    avant.unmount()
    const apres = render(<ReplayTeams doc={doc} scoreboard={[]} frame={40} locale="fr" />)
    expect(tuilesRendues(apres.container)).toBe(2)
    apres.unmount()
  })
})

/**
 * LES TUILES D'ATTENTE ONT LA BOÎTE D'UNE FICHE (retour utilisateur du 2026-10-06 : la tuile
 * « pas encore apparu » n'avait ni la largeur ni la hauteur d'une fiche, et toute la colonne
 * sautait). Dans CHACUN des deux gabarits, les trois sortes de tuile — la fiche d'un joueur
 * présent, la place libre (Q20), l'occupant pas encore apparu (Q21) — ont la même classe de
 * boîte, la même ligne du nom et le même corps à hauteur fixe ; seul l'habit tireté distingue
 * les tuiles d'attente.
 */
describe('ReplayTeams — les tuiles d’attente ont la boîte d’une fiche, dans les deux gabarits', () => {
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
      ],
      tracks: [
        vie('P', 512, 0, 199),
        vie('D', 513, 0, 50),
        vie('R', 514, 160, 199),
        { ...vie('', 515, 130, 199), bot: 'Attente [bot]' },
      ],
    })
  }

  const GABARITS = [
    {
      nom: 'normal (colonne)',
      header: { start_time: '2026-07-24T20:00:00Z' },
      boite: 'relative flex shrink-0 flex-col rounded-lg border px-2.5 py-2',
      corps: 'relative mt-[7px] h-[35px] overflow-hidden',
    },
    {
      nom: 'compact BTB (115 × 62)',
      header: { start_time: '2026-07-24T20:00:00Z', mode_category: 'BTB' },
      boite: 'relative flex shrink-0 flex-col rounded-md border px-1.5 py-1.5',
      corps: 'relative mt-[3px] h-[31px] overflow-hidden',
    },
  ] as const

  for (const g of GABARITS) {
    it(`${g.nom} : même boîte, même ligne du nom, même corps fixe pour les trois sortes de tuile`, () => {
      const vue = render(<ReplayTeams doc={documentTroisTuiles()} scoreboard={[]} frame={110} locale="fr" header={g.header} />)
      const conteneurs = [...vue.container.querySelectorAll('.overflow-y-auto')]
      expect(conteneurs).toHaveLength(1)
      const tuiles = [...conteneurs[0].children] as HTMLElement[]
      expect(tuiles).toHaveLength(3)
      // Les trois sortes sont là, chacune sur sa place.
      const fiche = vue.getByText('Présent').parentElement!.parentElement as HTMLElement
      const libre = vue.getByText(REPLAY_TEXT.fr.seatVacant).closest('[title]') as HTMLElement
      const attente = vue.getByText('Attente').parentElement!.parentElement as HTMLElement
      expect(tuiles[0]).toBe(fiche)
      expect(tuiles[1]).toBe(libre)
      expect(tuiles[2]).toBe(attente)
      expect(libre.title).toBe(REPLAY_TEXT.fr.seatVacantHint)
      expect(attente.title).toBe(REPLAY_TEXT.fr.seatNotSpawnedHint)
      // LA BOÎTE : la classe exacte de la fiche, sur les trois.
      for (const t of tuiles) expect(t.className).toBe(g.boite)
      // LE CORPS À HAUTEUR FIXE : un seul par tuile, la même classe sur les trois.
      const corps = tuiles.map((t) => [...t.children].filter((c) => c.className === g.corps))
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
  }

  it('le nom de l’occupant pas encore apparu s’écrit comme sur sa fiche : sans le suffixe « [bot] »', () => {
    const vue = render(<ReplayTeams doc={documentTroisTuiles()} scoreboard={[]} frame={110} locale="fr" />)
    expect(vue.getByText('Attente')).toBeTruthy()
    expect(vue.container.textContent).not.toContain('[bot]')
  })
})
