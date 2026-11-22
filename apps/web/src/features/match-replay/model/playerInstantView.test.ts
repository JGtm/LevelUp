/**
 * playerInstantView.test.ts — LA FICHE D'UN JOUEUR À L'IMAGE LUE, section par section, sur le match écrit à la
 * main de `test/instantFixture.ts`.
 *
 * Ce qu'ils protègent : chaque section existe avec ses valeurs quand tout est lu (Alpha, au dernier instant) ;
 * elle est ABSENTE quand son module rend `null` — jamais un texte de remplacement ; un zéro mesuré s'écrit pour les
 * compteurs et les tableaux, et n'a pas de ligne pour les listes de gestes ; la fiche suit l'image lue ; un joueur à
 * lecture partielle (Echo), un bot, un joueur mort et le début de match donnent chacun la fiche que leurs lectures
 * permettent, sans rien inventer.
 */
import { describe, expect, it } from 'vitest'

import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { INSTANT_ICONS, INSTANT_LAST_FRAME, instantDoc, instantFeed, instantPlayer, instantPlayers, instantSheet } from '../test/instantFixture'
import { playerInstantViewAt, type PlayerInstantView } from './playerInstantView'
import { readInstantDocSeries, readPlayerInstantSources } from './playerInstantSources'

type Doc = ReturnType<typeof instantDoc>

/** Le lecteur d'un joueur de la fixture : `lire(frame)` rend sa fiche à cette image. */
function lecteur(who: string, over: { doc?: Doc; sheet?: ReturnType<typeof instantSheet>; locale?: 'fr' | 'en' } = {}) {
  const doc = over.doc ?? instantDoc()
  const players = instantPlayers(doc, over.sheet)
  const sources = readPlayerInstantSources({
    doc,
    player: instantPlayer(players, who),
    players,
    feed: instantFeed(),
    scoreTimeline: scoreTimelineOf(doc),
    matchStartFrame: null,
    weaponIcons: INSTANT_ICONS,
    locale: over.locale ?? 'fr',
    docSeries: readInstantDocSeries(doc, players),
  })
  return (frame: number): PlayerInstantView => playerInstantViewAt(sources, frame)
}

const FIN = INSTANT_LAST_FRAME

describe('playerInstantView — Alpha, tout est lu, à la fin du match', () => {
  const v = lecteur('A')(FIN)

  it('l’identité : les compteurs du film', () => {
    expect(v.identity).toEqual({ kills: 4, deaths: 1, assists: 1, fromFilm: true })
  })

  it('depuis le début du match : FDA du jeu, séries, tirs à la tête, parfaites, vies, bouclier, score et rang', () => {
    expect(v.since?.fda).toBeCloseTo(4 + 1 / 3 - 1, 10)
    expect(v.since).toMatchObject({
      currentStreak: 1,
      bestStreak: 3,
      headshots: 2,
      perfects: 1,
      respawns: 1,
      currentLifeMs: 22_900,
      averageLifeMs: 15_000,
      shield: { breaks: 3, recharges: 1, differential: -2, matchBreaks: 3, matchRecharges: 1, matchDifferential: -2 },
      score: 300,
      rank: { place: 2, of: 3 },
    })
  })

  it('les éliminations par arme : classes par compte au match, armes nommées, barre aux parts à l’instant', () => {
    expect(v.kills?.classes.map((c) => [c.classKey, c.atInstant, c.atMatch])).toEqual([
      ['shoulder', 2, 2],
      ['grenade', 1, 1],
      ['heavy', 1, 1],
    ])
    expect(v.kills?.classes[0].weapons).toEqual([{ name: "Fusil d'assaut", atInstant: 2, share: 0.5, atMatch: 2 }])
    // La grenade n'est pas nommée par le fil : le référentiel des pictogrammes la nomme.
    expect(v.kills?.classes[1].weapons[0].name).toBe('Fragmentation')
    expect(v.kills?.unclassified).toBeNull()
    expect(v.kills?.bar).toEqual([
      { classKey: 'shoulder', share: 0.5 },
      { classKey: 'grenade', share: 0.25 },
      { classKey: 'heavy', share: 0.25 },
    ])
  })

  it('tirs et dégâts : tirs estimés au rapport du bilan, précision sur les tirs estimés, armes nommées', () => {
    expect(v.shots).toEqual({
      accuracy: { ratio: 6 / 35, hit: 6, fired: 35 },
      damageDealt: 160,
      damageTaken: 25,
      weapons: [
        { key: '0xAAAA', name: "Fusil d'assaut", fired: 30, damage: 60 },
        { key: '0xBBBB', name: 'SPNKr', fired: 5, damage: 100 },
      ],
    })
  })

  it('tirs et dégâts : les familles de même nom font UNE ligne, additionnée, sous la clé de la première ; les totaux restent', () => {
    const base = instantDoc()
    const doc = instantDoc({ weaponLabels: { ...base.weaponLabels, '0xBBBB': { fr: "Fusil d'assaut", en: 'Assault rifle' } } })
    const s = lecteur('A', { doc })(FIN).shots
    expect(s?.weapons).toEqual([{ key: '0xAAAA', name: "Fusil d'assaut", fired: 35, damage: 160 }])
    expect(s?.weapons[0].fired).toBe(s?.accuracy?.fired)
    expect(s?.weapons[0].damage).toBe(s?.damageDealt)
    expect(s?.weapons.map((w) => w.key)).toHaveLength(new Set(s?.weapons.map((w) => w.key)).size)
  })

  it('grenades lancées par type, ramassages d’armes et de grenades', () => {
    expect(v.grenadesThrown).toEqual([{ name: 'Fragmentation', count: 3 }, { name: 'Plasma', count: 1 }])
    expect(v.pickups).toEqual({ weapons: 3, grenades: 2 })
  })

  it('équipement : une famille, un pris, un utilisé, aucun perdu', () => {
    expect(v.equipment).toEqual([{ family: 'sensor', taken: 1, used: 1, dropped: 0 }])
  })

  it('médailles : groupées avec leur compte, dans l’ordre de la première obtention, image, sprite et rareté de la table', () => {
    expect(v.medals.map((m) => [m.name, m.label, m.count])).toEqual([
      ['Double Kill', 'Doublé', 2],
      ['Perfect', 'Parfait', 1],
      ['Mystery Medal', 'Mystery Medal', 1],
    ])
    expect(v.medals[0]).toMatchObject({ imageUrl: '/medals/double.png', sprite: null, difficulty: 'Normal' })
    expect(v.medals[1]).toMatchObject({ imageUrl: '', difficulty: 'Legendary' })
    expect(v.medals[1].sprite).toEqual({ sheet: '/medals/sprites.png', left: 74, top: 148, width: 74, height: 74 })
    // Une médaille que la table ne connaît pas : son nom brut, ni image ni halo.
    expect(v.medals[2]).toMatchObject({ imageUrl: '', sprite: null, difficulty: null })
  })

  it('gestes : temps de sprint cumulé et sauts ; sans couverture, ni déplacement ni rechargements', () => {
    expect(v.gestures).toEqual({ sprintMs: 4000, moveMs: null, jumps: 3, reloads: null, reloadsMinimum: false })
  })

  it('un document qui ne déclare pas le calque des médailles : ni éliminations parfaites ni médaille, jamais un zéro', () => {
    const base = instantDoc()
    const doc = instantDoc({ coverage: { ...base.coverage, medals: undefined } as never })
    const sansCalque = lecteur('A', { doc })(FIN)
    expect(sansCalque.since).not.toBeNull()
    expect(sansCalque.since?.perfects).toBeNull()
    expect(sansCalque.medals).toEqual([])
  })
})

describe('playerInstantView — la fiche suit l’image lue', () => {
  const lire = lecteur('A')

  it('à 10 s : trois éliminations, un bris de bouclier et une recharge, un sprint terminé, pas encore de réapparition', () => {
    const v = lire(100)
    expect(v.identity).toMatchObject({ kills: 3, deaths: 0, assists: 1 })
    expect(v.since).toMatchObject({
      respawns: 0, score: 300, currentLifeMs: 10_000, averageLifeMs: null,
      shield: { breaks: 1, recharges: 1, differential: 0, matchBreaks: 3, matchRecharges: 1, matchDifferential: -2 },
    })
    expect(v.kills?.classes.map((c) => [c.classKey, c.atInstant])).toEqual([['shoulder', 2], ['grenade', 0], ['heavy', 1]])
    expect(v.grenadesThrown).toEqual([{ name: 'Fragmentation', count: 2 }, { name: 'Plasma', count: 1 }])
    expect(v.pickups).toEqual({ weapons: 2, grenades: 1 })
    expect(v.gestures).toEqual({ sprintMs: 2000, moveMs: null, jumps: 2, reloads: null, reloadsMinimum: false })
  })

  it('les valeurs ne reculent jamais d’une image à l’autre', () => {
    let prev = { kills: 0, throws: 0, medals: 0, shields: 0 }
    for (let f = 0; f <= FIN; f += 5) {
      const v = lire(f)
      const cur = {
        kills: v.kills?.classes.reduce((n, c) => n + c.atInstant, 0) ?? 0,
        throws: v.grenadesThrown.reduce((n, g) => n + g.count, 0),
        medals: v.medals.reduce((n, m) => n + m.count, 0),
        shields: v.since?.shield?.breaks ?? 0,
      }
      for (const k of ['kills', 'throws', 'medals', 'shields'] as const) expect(cur[k]).toBeGreaterThanOrEqual(prev[k])
      prev = cur
    }
  })

  it('à 16 s Alpha est mort : pas de vie en cours, la moyenne des vies terminées demeure', () => {
    const v = lire(160)
    expect(v.since?.currentLifeMs).toBeNull()
    expect(v.since?.averageLifeMs).toBe(15_000)
    expect(v.identity).toMatchObject({ deaths: 1 })
  })
})

describe('playerInstantView — le début du match : des zéros mesurés, pas de section vide', () => {
  const v = lecteur('A')(0)

  it('les compteurs et les tableaux écrivent leurs zéros', () => {
    expect(v.identity).toEqual({ kills: 0, deaths: 0, assists: 0, fromFilm: true })
    expect(v.since).toMatchObject({ fda: 0, currentStreak: 0, bestStreak: 0, perfects: 0, respawns: 0, score: 0 })
    expect(v.since?.shield).toMatchObject({ breaks: 0, recharges: 0, differential: 0, matchBreaks: 3 })
    expect(v.kills?.classes.every((c) => c.atInstant === 0 && c.share === null)).toBe(true)
    expect(v.kills?.bar).toEqual([])
    expect(v.equipment).toEqual([])
    expect(v.shots).toMatchObject({ accuracy: null, damageDealt: 0, damageTaken: 0, weapons: [] })
  })

  it('les listes de gestes et les médailles n’ont rien à écrire ; les tirs à la tête sont un zéro mesuré', () => {
    expect(v.since?.headshots).toBe(0)
    expect(v.grenadesThrown).toEqual([])
    expect(v.pickups).toBeNull()
    expect(v.medals).toEqual([])
    expect(v.gestures).toBeNull()
  })
})

describe('playerInstantView — lecture partielle (Echo) : ni score ni tirs, rien à la place', () => {
  const v = lecteur('E')(FIN)

  it('pas de tirs et dégâts', () => {
    expect(v.shots).toBeNull()
  })

  it('pas de rang quand l’un des joueurs du lobby n’a pas de score établi : le score du joueur, lui, demeure', () => {
    const sansScore = lecteur('E', { sheet: instantSheet().map((r) => (r.xuid === 'bid(7.0)' ? { ...r, score: undefined } : r)) })(FIN)
    expect(sansScore.since?.score).toBe(400)
    expect(sansScore.since?.rank).toBeNull()
  })

  it('les autres sections existent pour ce que le document permet', () => {
    expect(v.since?.score).toBe(400)
    expect(v.since?.rank).toEqual({ place: 1, of: 3 })
    expect(v.kills?.classes).toHaveLength(1)
    expect(v.medals).toEqual([])
  })
})

describe('playerInstantView — le bot', () => {
  const v = lecteur('bot:Marc')(FIN)

  it('ses compteurs viennent de la feuille (le film ne le publie pas), ni tirs ni médailles', () => {
    expect(v.identity).toEqual({ kills: 0, deaths: 2, assists: 0, fromFilm: false })
    expect(v.shots).toBeNull()
    expect(v.medals).toEqual([])
    expect(v.kills).toBeNull()
  })

  it('son score est établi comme celui des autres, et il est dernier', () => {
    expect(v.since?.score).toBe(100)
    expect(v.since?.rank).toEqual({ place: 3, of: 3 })
  })
})

describe('playerInstantView — sans ligne de feuille', () => {
  it('Alpha sans feuille : les compteurs du film, ni score ni rang (rien ne vérifie la somme)', () => {
    const v = lecteur('A', { sheet: instantSheet().filter((r) => r.xuid !== 'A') })(FIN)
    expect(v.identity).toMatchObject({ kills: 4, fromFilm: true })
    expect(v.since?.score).toBeNull()
    expect(v.since?.rank).toBeNull()
  })

  it('un joueur que le film ne publie pas et sans feuille : pas d’identité chiffrée', () => {
    const doc = instantDoc({ scoreTimeline: undefined })
    const v = lecteur('E', { doc, sheet: [] })(FIN)
    expect(v.identity).toBeNull()
  })
})

describe('playerInstantView — la langue', () => {
  it('les noms de grenade et d’arme suivent la langue de la page', () => {
    const v = lecteur('A', { locale: 'en' })(FIN)
    expect(v.grenadesThrown.map((g) => g.name)).toEqual(['Frag', 'Plasma'])
    expect(v.shots?.weapons[0].name).toBe('Assault rifle')
    expect(v.medals[0].label).toBe('Doublé')
  })
})

describe('playerInstantView — les armes sans nom', () => {
  it('la seule arme sans nom d’une classe n’a pas de ligne ; avec une voisine nommée, elle en a une, sans nom', () => {
    const feed = instantFeed()
    // Une mêlée sans arme nommée (clé vide) puis une mêlée à l'arme nommée.
    const doc = instantDoc()
    const players = instantPlayers(doc)
    const kill = (replayMs: number, key: string, label: string) =>
      ({
        key: `m${replayMs}`, replayMs, medal: null, death: null,
        kill: { xuid: 'A', replayMs, weaponKey: key, weaponClass: 'melee', weaponLabel: label, medals: [] },
      }) as unknown as (typeof feed)[number]
    const mk = (extra: typeof feed) =>
      playerInstantViewAt(
        readPlayerInstantSources({
          doc, player: instantPlayer(players, 'A'), players, feed: [...feed, ...extra],
          scoreTimeline: scoreTimelineOf(doc), matchStartFrame: null,
          weaponIcons: INSTANT_ICONS, locale: 'fr', docSeries: readInstantDocSeries(doc, players),
        }),
        FIN,
      )
    const seule = mk([kill(30_000, '', '')]).kills?.classes.find((c) => c.classKey === 'melee')
    expect(seule?.atMatch).toBe(1)
    expect(seule?.weapons).toEqual([])
    const deux = mk([kill(30_000, '', ''), kill(31_000, 'hinf_hammer', 'Marteau')]).kills?.classes.find((c) => c.classKey === 'melee')
    expect(deux?.weapons.map((w) => w.name).sort()).toEqual(['Marteau', null])
  })
})

describe('playerInstantView — rechargements et temps en commande de déplacement', () => {
  const base = instantDoc()
  const control = { holds: 2, withHole: 0, innerHoles: 0, entries: 4, published: 2, intervals: 2, noTrack: 0, noPlayer: 0 }
  const reloadsRead = {
    frames: 300, closedFrames: 300, stoppedFrames: 0, read: 4, copies: 0, withoutPlayer: 0,
    beforeOrigin: 0, afterEnd: 0, published: 4,
  }
  const reloads = [{ t: 15, place: 0 }, { t: 90, place: 0 }, { t: 200, place: 0 }, { t: 20, place: 1 }]
  const moves = [
    { slot: 512, kind: 'moveCommand', t0: 5, t1: 25 },
    { slot: 513, kind: 'moveCommand', t0: 175, t1: 185 },
  ]
  const docWith = (cov: { stances?: object; reloads?: object }) =>
    instantDoc({
      coverage: {
        ...base.coverage,
        stances: { ...base.coverage?.stances, ...cov.stances },
        ...(cov.reloads ? { reloads: cov.reloads } : {}),
      } as never,
      reloads,
      stances: [...base.stances, ...moves],
    } as never)
  const lu = docWith({ stances: { control }, reloads: reloadsRead })

  it('Alpha : les rechargements et le temps tenu, cumulés à la fin du match', () => {
    const v = lecteur('A', { doc: lu })(FIN)
    expect(v.gestures).toEqual({ sprintMs: 4000, moveMs: 3000, jumps: 3, reloads: 3, reloadsMinimum: false })
  })

  it('suivent l’image lue, sans reculer', () => {
    const lire = lecteur('A', { doc: lu })
    expect(lire(10).gestures).toMatchObject({ moveMs: 500, reloads: null })
    expect(lire(100).gestures).toMatchObject({ moveMs: 2000, reloads: 2 })
    expect(lire(FIN).gestures).toMatchObject({ moveMs: 3000, reloads: 3 })
  })

  it('une trame arrêtée fait des rechargements un minimum', () => {
    const doc = docWith({ stances: { control }, reloads: { ...reloadsRead, closedFrames: 298, stoppedFrames: 2 } })
    expect(lecteur('A', { doc })(FIN).gestures).toMatchObject({ reloads: 3, reloadsMinimum: true })
  })

  it('sans couverture des rechargements : aucune ligne de rechargements, même si le calque porte des lignes', () => {
    const doc = docWith({ stances: { control } })
    const v = lecteur('A', { doc })(FIN)
    expect(v.gestures?.reloads).toBeNull()
    expect(v.gestures?.moveMs).toBe(3000)
  })

  it('sans couverture de la commande de déplacement : aucune ligne de temps tenu', () => {
    const doc = docWith({ reloads: reloadsRead })
    const v = lecteur('A', { doc })(FIN)
    expect(v.gestures?.moveMs).toBeNull()
    expect(v.gestures?.reloads).toBe(3)
  })

  it('un joueur qui n’a ni rechargé ni tenu de commande n’a pas de ligne (zéro mesuré, jamais écrit)', () => {
    const doc = docWith({ stances: { control }, reloads: reloadsRead })
    const v = lecteur('E', { doc })(FIN)
    expect(v.gestures?.moveMs).toBeNull()
    expect(v.gestures?.reloads).toBe(1)
  })
})
