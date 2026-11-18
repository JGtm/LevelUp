/**
 * playerInstantEstimates.test.ts — LE SCORE, LE RANG ET LES ESTIMATIONS DE TIR À L'IMAGE LUE.
 *
 * Un match écrit à la main (grille de 100 ms) donne les valeurs exactes : trois gardes du score (verdict du
 * document, horloge, somme égale à la feuille), le rang et ses égalités, la suffisance des tirs, les cumuls par
 * arme. Les documents publiés par Go (`fixtures/go/`) prouvent sur des films réels que les cumuls ne reculent
 * jamais et ne dépassent jamais les totaux de match du document.
 */
import { describe, expect, it } from 'vitest'

import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc, type TestReplayDocOverrides } from '../test/testDoc'

import {
  playerInstantEstimatesAt,
  readInstantEstimatesSeries,
  type InstantEstimatesSeries,
  type PlayerInstantEstimates,
} from './playerInstantEstimates'

const COMPLETE = { scoreActions: { complete: true } } as never
const INCOMPLETE = { scoreActions: { complete: false } } as never

const action = (t: number, xuid: string, points: number) => ({ t, xuid, name: 'killPersonalScore', points })

/**
 * Les points par action : A et B finissent à 300 (égalité en tête), C à 200, le bot à 100 (publié sous le xuid de sa
 * ligne de feuille). B marque 100 de plus que A à l'image 60 puis A le rejoint à 70.
 */
const ACTIONS = [
  action(10, 'A', 100), action(20, 'B', 100), action(30, 'A', 100), action(40, 'B', 100),
  action(45, 'bid(7.0)', 100), action(50, 'C', 100), action(60, 'B', 100), action(70, 'A', 100), action(80, 'C', 100),
]

const SHOT_STAT = {
  coverage: 0.9, damageDealt: 0, damageTaken: 0, duplicates: 1, hits: 0, messages: 0, shotEvents: 0, shots: 0,
}
const stat = (xuid: string, sufficient: boolean) => ({ ...SHOT_STAT, xuid, sufficient })

const shot = (t: number, slot: number, w?: string) => ({ t, slot, x: 0, y: 0, ...(w ? { w } : {}) })
const hit = (t: number, shooter: string, victim: string) => ({ t, shooter, victim, effect: 'e', w: 'W1' })
const dmg = (t: number, attacker: string | undefined, victim: string, shield: number, body: number, w?: string) => ({
  t, attacker, victim, effect: 'e', shield, body, ...(w ? { w } : {}),
})

const SHOTS = [
  shot(5, 512, 'W1'), shot(15, 512, 'W1'), shot(25, 512, 'W2'), shot(35, 512), shot(10, 513, 'W1'),
  // Le slot 512 est libre après la vie de A : ce tir n'est celui de personne.
  shot(190, 512, 'W1'),
]
const HITS = [hit(6, 'A', 'B'), hit(16, 'A', 'B'), hit(17, 'A', 'A'), hit(11, 'B', 'A')]
const DAMAGE = [
  dmg(6, 'A', 'B', 20, 0, 'W1'),
  dmg(16, 'A', 'B', 10.3, 5, 'W1'),
  dmg(26, 'A', 'B', 0, 40, 'W2'),
  dmg(11, 'B', 'A', 25, 0, 'W1'),
  // Du décor (pas d'auteur) : subi par A.
  dmg(12, undefined, 'A', 0, 30),
  // Un dégât à soi-même n'est ni infligé ni subi.
  dmg(18, 'A', 'A', 99, 0, 'W1'),
]

function docWith(over: TestReplayDocOverrides = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    frameCount: 200,
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
      { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
      { xuid: 'C', filmIndex: 2, name: 'Charlie', team: 1 },
      { xuid: '', bot: true, filmIndex: 3, name: 'Marc', team: 1 },
    ],
    tracks: [
      { slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] },
      { slot: 513, team: 1, xuid: 'B', startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }] },
      { slot: 514, team: 1, xuid: 'C', startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }] },
      { slot: 515, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }] },
    ],
    coverage: COMPLETE,
    scoreActions: ACTIONS,
    shotStats: [stat('A', true), stat('B', true), stat('C', false)],
    shots: SHOTS,
    hits: HITS,
    damage: DAMAGE,
    ...over,
  })
}

const SHEET = { A: 300, B: 300, C: 200, BOT: 100 }

function lobby(
  doc = docWith(),
  sheet: Partial<Record<keyof typeof SHEET, number | null>> = {},
): { players: ReplayPlayer[]; series: InstantEstimatesSeries; at: (who: number, frame: number) => PlayerInstantEstimates } {
  const score = (k: keyof typeof SHEET) => (k in sheet ? sheet[k] : SHEET[k])
  const rows = [
    ['A', scoreboardRow('A', 'Alpha', 'Eagle', { score: score('A') ?? undefined })],
    ['B', scoreboardRow('B', 'Bravo', 'Cobra', { score: score('B') ?? undefined })],
    ['C', scoreboardRow('C', 'Charlie', 'Cobra', { score: score('C') ?? undefined })],
    ['BOT', scoreboardRow('bid(7.0)', 'Marc', 'Cobra', { is_bot: true, score: score('BOT') ?? undefined })],
  ] as const
  const players = buildPlayers(doc, rows.filter(([k]) => score(k) !== null).map(([, r]) => r))
  const series = readInstantEstimatesSeries({ doc, players })
  return { players, series, at: (who, frame) => playerInstantEstimatesAt(series, players[who], frame) }
}

const [A, B, C, BOT] = [0, 1, 2, 3]

describe('playerInstantEstimates — score : les trois gardes', () => {
  it('score par action incomplet : aucun score ni rang, quelle que soit la feuille', () => {
    const { at } = lobby(docWith({ coverage: INCOMPLETE }))
    for (const who of [A, B, C, BOT]) {
      const e = at(who, 100)
      expect(e.score).toBeNull()
      expect(e.absences.score).toBe('score-incomplet')
      expect(e.rank).toBeNull()
      expect(e.absences.rank).toBe('lobby-incomplet')
    }
  })

  it('document sans couverture du score par action : incomplet', () => {
    const e = lobby(docWith({ coverage: undefined })).at(A, 100)
    expect(e.score).toBeNull()
    expect(e.absences.score).toBe('score-incomplet')
  })

  it("horloge du film non recalée : pas de score, les points seraient datés à un écart inconnu", () => {
    const doc = docWith({ coverage: { scoreActions: { complete: true }, originResolved: false } as never })
    const e = lobby(doc).at(A, 100)
    expect(e.score).toBeNull()
    expect(e.absences.score).toBe('horloge-non-recalee')
  })

  it("horloge : une origine servie même avec le drapeau à faux garde le score", () => {
    const doc = docWith({ coverage: { scoreActions: { complete: true }, originResolved: false } as never, originMs: 500 })
    expect(lobby(doc).at(A, 100).score).toBe(300)
  })

  it('somme différente du score de la feuille : ce joueur est sans score, et personne n a de rang', () => {
    const { at } = lobby(docWith(), { B: 310 })
    const b = at(B, 100)
    expect(b.score).toBeNull()
    expect(b.absences.score).toBe('somme-differente')
    // Les autres joueurs gardent leur score ; leur rang, lui, dépend de tout le lobby.
    expect(at(A, 100).score).toBe(300)
    for (const who of [A, B, C, BOT]) {
      expect(at(who, 100).rank).toBeNull()
      expect(at(who, 100).absences.rank).toBe('lobby-incomplet')
    }
  })

  it('la somme est comparée à la FIN du match, pas à l image lue', () => {
    // A 300 à la fin : à l'image 10 elle vaut 100, et le score est quand même établi.
    expect(lobby().at(A, 10).score).toBe(100)
  })

  it('joueur sans ligne de feuille : rien ne permet de vérifier la somme, score absent', () => {
    const { at } = lobby(docWith(), { C: null })
    const c = at(C, 100)
    expect(c.score).toBeNull()
    expect(c.absences.score).toBe('feuille-absente')
    expect(at(A, 100).rank).toBeNull()
  })

  it('une feuille dont le score n est pas renseigné est une feuille absente', () => {
    const { players } = lobby()
    const sansScore = { ...players[A], board: { ...players[A].board!, score: undefined } }
    const series = readInstantEstimatesSeries({ doc: docWith(), players: [sansScore, ...players.slice(1)] })
    expect(playerInstantEstimatesAt(series, sansScore, 100).absences.score).toBe('feuille-absente')
  })
})

describe('playerInstantEstimates — score établi', () => {
  const { at } = lobby()

  it('début de match : zéro mesuré, pas une absence', () => {
    const e = at(A, 0)
    expect(e.score).toBe(0)
    expect(e.absences.score).toBeNull()
  })

  it('cumule les points par action jusqu à l image comprise', () => {
    const frames = [0, 9, 10, 29, 30, 69, 70, 199]
    expect(frames.map((f) => at(A, f).score)).toEqual([0, 0, 100, 100, 200, 200, 300, 300])
    expect(at(B, 20).score).toBe(100)
    expect(at(C, 79).score).toBe(100)
    expect(at(C, 80).score).toBe(200)
  })

  it('un bot est lu sous le xuid de sa ligne de feuille', () => {
    expect(at(BOT, 44).score).toBe(0)
    expect(at(BOT, 45).score).toBe(100)
  })

  it('ne recule jamais quand le document ne porte aucun point négatif', () => {
    for (const who of [A, B, C, BOT]) {
      let prev = 0
      for (let f = 0; f <= 199; f++) {
        const s = at(who, f).score as number
        expect(s).toBeGreaterThanOrEqual(prev)
        prev = s
      }
    }
  })

  it('recule quand le document porte une pénalité : la somme est le score, signe compris', () => {
    const doc = docWith({ scoreActions: [action(10, 'A', 100), action(20, 'A', -25), action(30, 'A', 100)] })
    const l = lobby(doc, { A: 175, B: 0, C: 0, BOT: 0 })
    expect([10, 20, 30].map((f) => l.at(A, f).score)).toEqual([100, 75, 175])
  })

  it('plusieurs actions à la même image se cumulent toutes', () => {
    const doc = docWith({ scoreActions: [action(10, 'A', 100), action(10, 'A', 50)] })
    expect(lobby(doc, { A: 150, B: 0, C: 0, BOT: 0 }).at(A, 10).score).toBe(150)
  })
})

describe('playerInstantEstimates — rang', () => {
  const { at } = lobby()
  const places = (frame: number) => [A, B, C, BOT].map((w) => at(w, frame).rank?.place)

  it('début de match : tout le monde est à égalité, premier', () => {
    expect(places(0)).toEqual([1, 1, 1, 1])
  })

  it('un seul meneur, les autres à égalité : 1 puis 2, 2, 2', () => {
    expect(places(10)).toEqual([1, 2, 2, 2])
  })

  it('classement strict : 1, 2, puis des égalités qui sautent le rang suivant', () => {
    // A 200, B 100, C 0, bot 0 à l'image 35 : C et le bot sont troisièmes ex aequo.
    expect(places(35)).toEqual([1, 2, 3, 3])
  })

  it('égalité en tête : 1, 1, 3, puis 4', () => {
    expect(places(100)).toEqual([1, 1, 3, 4])
  })

  it('porte la taille du lobby, bots compris', () => {
    expect(at(A, 100).rank).toEqual({ place: 1, of: 4 })
    expect(at(BOT, 100).rank).toEqual({ place: 4, of: 4 })
  })

  it('le rang suit le score à l image : B passe devant A à l image 60, A le rejoint à 70', () => {
    expect(at(A, 60).rank?.place).toBe(2)
    expect(at(B, 60).rank?.place).toBe(1)
    expect(at(A, 70).rank?.place).toBe(1)
  })

  it('un joueur étranger au lobby lu est absent partout', () => {
    const l = lobby()
    const foreign = buildPlayers(docWith(), [scoreboardRow('Z', 'Zed', 'Eagle', { score: 0 })])[0]
    const e = playerInstantEstimatesAt(l.series, { ...foreign, xuid: 'Z' }, 100)
    expect([e.score, e.rank, e.shots]).toEqual([null, null, null])
  })
})

describe('playerInstantEstimates — tirs : suffisance', () => {
  it('joueur insuffisant : aucune section, la cause est nommée', () => {
    const e = lobby().at(C, 100)
    expect(e.shots).toBeNull()
    expect(e.absences.shots).toBe('lecture-insuffisante')
  })

  it('un bot n a pas de bilan de tirs : non publié', () => {
    const e = lobby().at(BOT, 100)
    expect(e.shots).toBeNull()
    expect(e.absences.shots).toBe('non-publie')
  })

  it('document sans aucun bilan : non publié pour tous, le score reste lu', () => {
    const l = lobby(docWith({ shotStats: [] }))
    for (const who of [A, B, C, BOT]) {
      expect(l.at(who, 100).shots).toBeNull()
      expect(l.at(who, 100).absences.shots).toBe('non-publie')
    }
    expect(l.at(A, 100).score).toBe(300)
  })

  it('un joueur insuffisant n est pas compté même s il a des tirs datés', () => {
    const doc = docWith({ shots: [shot(5, 514, 'W1')], shotStats: [stat('C', false)] })
    expect(lobby(doc).at(C, 100).shots).toBeNull()
  })

  it('l absence de tirs ne dépend ni du score ni du rang', () => {
    const e = lobby(docWith({ coverage: INCOMPLETE })).at(A, 100)
    expect(e.score).toBeNull()
    expect(e.shots?.estimated).toBe(true)
  })
})

describe('playerInstantEstimates — tirs : valeurs', () => {
  const { at } = lobby()

  it('début de match : zéro mesuré, précision absente', () => {
    const s = at(A, 0).shots!
    expect(s).toMatchObject({ estimated: true, fired: 0, hit: 0, accuracy: null, damageDealt: 0, damageTaken: 0 })
    expect(s.byWeapon).toEqual([])
  })

  it('un tir sans touche : précision zéro, pas absente', () => {
    const s = at(A, 5).shots!
    expect([s.fired, s.hit, s.accuracy]).toEqual([1, 0, 0])
  })

  it('fin de match : tirs, touches, précision, dégâts infligés et subis', () => {
    const s = at(A, 100).shots!
    expect(s.fired).toBe(4)
    // Les touches sur soi-même ne comptent pas.
    expect(s.hit).toBe(2)
    expect(s.accuracy).toBeCloseTo(0.5, 10)
    // Les dégâts à soi-même ne comptent ni infligés ni subis : 20 + 15,3 + 40 infligés, 25 + 30 (décor) subis.
    expect(s.damageDealt).toBeCloseTo(75.3, 10)
    expect(s.damageTaken).toBeCloseTo(55, 10)
  })

  it('par arme : tirs et dégâts, arme non résolue comprise, du plus tiré au moins tiré', () => {
    const s = at(A, 100).shots!
    expect(s.byWeapon).toEqual([
      { weapon: 'W1', fired: 2, firedEstimated: 2, damage: 35.3 },
      { weapon: 'W2', fired: 1, firedEstimated: 1, damage: 40 },
      { weapon: null, fired: 1, firedEstimated: 1, damage: 0 },
    ])
  })

  it('la somme des armes égale le total, à chaque image', () => {
    for (let f = 0; f <= 199; f += 3) {
      const s = at(A, f).shots!
      expect(s.byWeapon.reduce((n, w) => n + w.fired, 0)).toBe(s.fired)
      expect(s.byWeapon.reduce((n, w) => n + w.damage, 0)).toBeCloseTo(s.damageDealt, 6)
    }
  })

  it('un tir daté après la vie du joueur, sur un slot libre, n est le tir de personne', () => {
    expect(at(A, 199).shots!.fired).toBe(4)
    expect(at(B, 199).shots!.fired).toBe(1)
  })

  it('un autre joueur a ses propres valeurs : B a un tir et subit les dégâts de A', () => {
    const s = at(B, 100).shots!
    expect([s.fired, s.hit, s.accuracy]).toEqual([1, 1, 1])
    expect(s.damageDealt).toBeCloseTo(25, 10)
    expect(s.damageTaken).toBeCloseTo(75.3, 10)
  })

  it('aucun tir sur tout le match : précision absente, touches et dégâts lus', () => {
    const doc = docWith({ shots: [], shotStats: [stat('A', true)] })
    const s = lobby(doc).at(A, 199).shots!
    expect(s.fired).toBe(0)
    expect(s.accuracy).toBeNull()
    expect(s.hit).toBe(2)
    expect(s.damageDealt).toBeCloseTo(75.3, 10)
  })

  it('tirs, touches et dégâts ne reculent jamais d une image à l autre', () => {
    for (const who of [A, B]) {
      let prev = { fired: 0, hit: 0, dealt: 0, taken: 0 }
      for (let f = 0; f <= 199; f++) {
        const s = at(who, f).shots!
        const cur = { fired: s.fired, hit: s.hit, dealt: s.damageDealt, taken: s.damageTaken }
        for (const k of ['fired', 'hit', 'dealt', 'taken'] as const) expect(cur[k]).toBeGreaterThanOrEqual(prev[k])
        prev = cur
      }
    }
  })
})

describe('playerInstantEstimates — tirs estimés (rapport du bilan)', () => {
  /** A a écrit 4 tirs dans le film pour 20 tirs du jeu : le rapport est 5. */
  const bilan = { ...stat('A', true), shots: 20, shotEvents: 4 }
  const doc = docWith({ shotStats: [bilan, stat('B', true), stat('C', false)] })
  const { at } = lobby(doc)

  it('à la dernière image, les tirs estimés égalent ceux du bilan et la précision est touches sur tirs estimés du bilan', () => {
    const s = at(A, 199).shots!
    expect(s.fired).toBe(bilan.shotEvents)
    expect(s.firedEstimated).toBe(bilan.shots)
    expect(s.accuracy).toBeCloseTo(s.hit / bilan.shots, 10)
    expect(s.accuracy).toBeCloseTo(0.1, 10)
  })

  it('à l instant : les tirs écrits au rapport du bilan, arrondis à l entier', () => {
    expect(at(A, 5).shots!).toMatchObject({ fired: 1, firedEstimated: 5, hit: 0, accuracy: 0 })
    expect(at(A, 16).shots!).toMatchObject({ fired: 2, firedEstimated: 10, hit: 2 })
    expect(at(A, 16).shots!.accuracy).toBeCloseTo(0.2, 10)
  })

  it('début de match : aucun tir estimé, précision absente', () => {
    expect(at(A, 0).shots!).toMatchObject({ fired: 0, firedEstimated: 0, accuracy: null })
  })

  it('par arme : le même rapport s applique aux tirs de chaque arme', () => {
    expect(at(A, 100).shots!.byWeapon.map((w) => [w.weapon, w.fired, w.firedEstimated])).toEqual([
      ['W1', 2, 10],
      ['W2', 1, 5],
      [null, 1, 5],
    ])
  })

  it('un rapport non entier est arrondi', () => {
    const rapport = lobby(docWith({ shotStats: [{ ...stat('A', true), shots: 10, shotEvents: 4 }, stat('B', true)] })).at
    // 1 tir écrit x 2,5 = 2,5 -> 3 ; 4 tirs écrits x 2,5 = 10.
    expect(rapport(A, 5).shots!.firedEstimated).toBe(3)
    expect(rapport(A, 199).shots!.firedEstimated).toBe(10)
  })

  it('un bilan sans tirs écrits ne donne pas de rapport : les tirs estimés sont les tirs écrits', () => {
    const sans = lobby(docWith({ shotStats: [{ ...stat('A', true), shots: 20, shotEvents: 0 }, stat('B', true)] })).at
    expect(sans(A, 199).shots).toMatchObject({ fired: 4, firedEstimated: 4 })
  })
})

/** La feuille d'un match de fixture : le score de chaque joueur est la somme de ses points par action. */
function sheetFromActions(doc: ReturnType<typeof testReplayDoc>) {
  const totals = new Map<string, number>()
  for (const a of doc.scoreActions) totals.set(a.xuid, (totals.get(a.xuid) ?? 0) + a.points)
  return [...new Set(doc.roster.map((r) => r.xuid).filter((x) => x !== ''))].map((x) =>
    scoreboardRow(x, x, null, { score: totals.get(x) ?? 0 }),
  )
}

describe('playerInstantEstimates — films réels (fixtures Go)', () => {
  for (const entry of goFixtureEntries()) {
    it(`${entry.film} (${entry.build}) : cumuls monotones et bornés par les totaux du document`, () => {
      const doc = testReplayDoc(loadGoFixture(entry).doc)
      const players = buildPlayers(doc, sheetFromActions(doc))
      const series = readInstantEstimatesSeries({ doc, players })
      const last = doc.frameCount - 1
      for (const p of players) {
        const stat = doc.shotStats.find((s) => s.xuid === p.xuid)
        const end = playerInstantEstimatesAt(series, p, last)
        // Les gardes : sans bilan suffisant, pas de section ; avec, des valeurs cohérentes avec le bilan.
        expect(end.shots === null).toBe(stat?.sufficient !== true)
        if (!end.shots || !stat) continue
        expect(end.shots.fired).toBeLessThanOrEqual(stat.shotEvents)
        // Les tirs estimés ne dépassent pas les tirs du jeu du bilan.
        expect(end.shots.firedEstimated).toBeLessThanOrEqual(Math.max(stat.shots, stat.shotEvents) + 1)
        expect(end.shots.hit).toBeLessThanOrEqual(stat.hits)
        expect(end.shots.damageDealt).toBeLessThanOrEqual(stat.damageDealt + 0.1)
        expect(end.shots.damageTaken).toBeLessThanOrEqual(stat.damageTaken + 0.1)
        expect(end.shots.byWeapon.reduce((n, w) => n + w.fired, 0)).toBe(end.shots.fired)
        let prev = { fired: 0, hit: 0, dealt: 0, taken: 0 }
        for (let f = 0; f <= last; f += 25) {
          const s = playerInstantEstimatesAt(series, p, f).shots!
          expect(s.fired).toBeGreaterThanOrEqual(prev.fired)
          expect(s.hit).toBeGreaterThanOrEqual(prev.hit)
          expect(s.damageDealt).toBeGreaterThanOrEqual(prev.dealt)
          expect(s.damageTaken).toBeGreaterThanOrEqual(prev.taken)
          prev = { fired: s.fired, hit: s.hit, dealt: s.damageDealt, taken: s.damageTaken }
        }
      }
    })

    it(`${entry.film} (${entry.build}) : score et rang suivent le verdict du document`, () => {
      const doc = testReplayDoc(loadGoFixture(entry).doc)
      const players = buildPlayers(doc, sheetFromActions(doc))
      const series = readInstantEstimatesSeries({ doc, players })
      const last = doc.frameCount - 1
      const complete = doc.coverage?.scoreActions?.complete === true
      for (const p of players) {
        const e = playerInstantEstimatesAt(series, p, last)
        if (!complete) {
          expect(e.score).toBeNull()
          expect(e.absences.score).toBe('score-incomplet')
          expect(e.rank).toBeNull()
          continue
        }
        // La feuille est ici la somme des points : un score établi est cette somme ; un joueur sans ligne de
        // feuille (un bot) est absent par la feuille, jamais par un zéro.
        if (e.score === null) {
          expect(e.absences.score).toBe('feuille-absente')
          expect(e.rank).toBeNull()
          continue
        }
        const sum = doc.scoreActions.filter((a) => a.xuid === p.xuid).reduce((n, a) => n + a.points, 0)
        expect(e.score).toBe(sum)
      }
      const established = players.every((p) => playerInstantEstimatesAt(series, p, last).score !== null)
      for (const p of players) {
        const rank = playerInstantEstimatesAt(series, p, last).rank
        expect(rank === null).toBe(!complete || !established)
        if (rank) {
          expect(rank.of).toBe(players.length)
          expect(rank.place).toBeGreaterThanOrEqual(1)
          expect(rank.place).toBeLessThanOrEqual(players.length)
        }
      }
    })
  }
})
