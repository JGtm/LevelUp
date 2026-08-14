/**
 * killFeedLogic.test.ts — le recalage des deux horloges, le fil PERMANENT, et le
 * rattachement des médailles.
 *
 * Les chiffres viennent du match 000d5950 : T0 = 18 465 ms (real_start_time −
 * start_time_utc), premier kill à 35 306 ms sur l'horloge BRUTE, donc 16 841 ms sur
 * l'horloge gameplay que sert la Match View. Rattachement médaille→kill mesuré sur ce
 * même match : 42/44 à ≤ 500 ms, 2 médailles seules.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchHighlightEvent } from '@/lib/api/types'

import {
  alignFeed,
  alignFeedByOrigin,
  alignFeedToTracks,
  buildFeedEntries,
  collectMedalEvents,
  feedAt,
  toReplayKills,
  type MedalEvent,
} from './killFeedLogic'
import { testReplayDoc } from './test/testDoc'

const T0 = 18_465

function kill(tMs: number, xuid = 'x1', victimXuid = ''): KillEvent {
  return {
    tMs,
    xuid,
    ally: true,
    teamID: 0,
    weaponKey: 'hinf_br75',
    weaponLabel: 'BR75',
    weaponImageUrl: '/static/weapons/br75.png',
    weaponTinted: true,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid,
    victimGamertag: '',
    victimTeamID: null,
  }
}

function medal(tMs: number, xuid = 'x1', name = 'No Scope'): MedalEvent {
  return {
    tMs,
    xuid,
    gamertag: 'GT',
    teamID: 0,
    name,
    label: 'Sans lunette',
    description: 'Desc',
    imageUrl: '/static/medals/1.png',
  }
}

describe('toReplayKills', () => {
  it('remet le countdown que la Match View avait retranché', () => {
    // 16 841 ms sur l'horloge gameplay = 35 306 ms sur celle du film, la seule que le
    // rejeu connaisse.
    const [k] = toReplayKills([kill(16_841)], T0)
    expect(k.replayMs).toBe(35_306)
  })

  it("n'ajoute rien quand le countdown est inconnu", () => {
    // T0 nul veut dire que la correction n'a pas eu lieu côté events non plus : les deux
    // horloges coïncident déjà, et ajouter quoi que ce soit les décalerait.
    expect(toReplayKills([kill(16_841)], 0)[0].replayMs).toBe(16_841)
    expect(toReplayKills([kill(16_841)], Number.NaN)[0].replayMs).toBe(16_841)
  })

  it('trie chronologiquement — feedAt en dépend', () => {
    const out = toReplayKills([kill(30_000), kill(10_000), kill(20_000)], T0)
    expect(out.map((k) => k.tMs)).toEqual([10_000, 20_000, 30_000])
  })
})

/**
 * Un document 10 Hz, frameCount 200 (horizon 19 900 ms) :
 *  - la victime V meurt deux fois (frames 20 et 80, soit 2 000 et 8 000 ms) ;
 *  - W meurt une fois (frame 50, 5 000 ms) sans qu'aucun kill ne le revendique ;
 *  - S survit (sa vie couvre l'horizon).
 */
function docSpec() {
  const track = (xuid: string, startFrame: number, endFrame: number) => ({
    slot: 1,
    team: -1,
    xuid,
    points: [{ t: startFrame, x: 0, y: 0 }],
    startFrame,
    endFrame,
  })
  return {
    frameIntervalMs: 100,
    tracks: [track('V', 0, 20), track('V', 40, 80), track('W', 0, 50), track('S', 0, 199)],
  }
}

function docWithLives() {
  return testReplayDoc(docSpec())
}

describe('alignFeedToTracks — le fil sur le référentiel des pistes', () => {
  // Les kills arrivent avec le décalage d'origine de l'artefact : +500 ms ici, +3 678 ms
  // sur le témoin 000d5950 (mesure 2026-08-14, cf. en-tête du module).
  const kills = [kill(2_500, 'k1', 'V'), kill(8_500, 'k2', 'V')]

  it('mesure le décalage du document et pose chaque kill SUR la fin de vie de sa victime', () => {
    const out = alignFeedToTracks(kills, 0, docWithLives())
    expect(out.offsetMs).toBe(500)
    // L'instant de la ligne est EXACTEMENT celui du flash de fiche (fin de piste).
    expect(out.kills.map((k) => k.replayMs)).toEqual([2_000, 8_000])
  })

  it('corrige un kill inappariable du décalage MESURÉ — jamais d\'une constante', () => {
    const out = alignFeedToTracks([...kills, kill(12_500, 'k3')], 0, docWithLives())
    expect(out.kills.map((k) => k.replayMs)).toEqual([2_000, 8_000, 12_000])
  })

  it('sans aucune paire mesurable, l\'horloge brute reste — rien n\'est inventé', () => {
    const out = alignFeedToTracks([kill(2_500, 'k1')], 0, docWithLives())
    expect(out.offsetMs).toBeNull()
    expect(out.kills[0].replayMs).toBe(2_500)
  })

  it('la mort sans kill fait une MORT NEUTRE à l\'instant de la piste ; jamais de doublon', () => {
    const out = alignFeedToTracks(kills, 0, docWithLives())
    // W (5 000 ms) n'a pas de ligne de kill -> mort neutre. Les deux fins de vie de V
    // sont consommées par leurs kills : AUCUNE ligne neutre pour elles.
    expect(out.deaths).toEqual([{ replayMs: 5_000, xuid: 'W', kind: '', img: '', tinted: false }])
  })

  it('un survivant de fin de partie ne meurt pas dans le fil', () => {
    const out = alignFeedToTracks(kills, 0, docWithLives())
    expect(out.deaths.some((d) => d.xuid === 'S')).toBe(false)
  })

  it('un kill SANS victime au voisinage met son VETO sur la mort neutre (anti-doublon)', () => {
    // Le kill non attribué à 5 400 ms peut être celui de W : pas de ligne neutre.
    const out = alignFeedToTracks([...kills, kill(5_900, 'k3')], 0, docWithLives())
    expect(out.deaths).toEqual([])
  })

  it('sans appariement mesuré, AUCUNE mort neutre — on ne sait pas dédoublonner', () => {
    const out = alignFeedToTracks([kill(5_500, 'k1')], 0, docWithLives())
    expect(out.deaths).toEqual([])
  })
})

describe("alignFeedByOrigin — le fil sur l'origine publiée par l'artefact", () => {
  // Même montage que le repli : l'artefact publie 500 ms d'origine au lieu de la faire
  // mesurer au navigateur.
  const docAvecOrigine = () => testReplayDoc({ ...docSpec(), originMs: 500 })
  const kills = [kill(2_500, 'k1', 'V'), kill(8_500, 'k2', 'V')]

  it("retranche l'origine à TOUS les kills — une soustraction, pas un appariement", () => {
    const out = alignFeedByOrigin(kills, 0, docAvecOrigine(), 500)
    expect(out.kills.map((k) => k.replayMs)).toEqual([2_000, 8_000])
    expect(out.offsetMs).toBe(500)
  })

  it('recale AUSSI les kills sans victime nommée — ce que le repli ne savait pas faire', () => {
    // Le cas du témoin 606d9844 : 35 kills, 0 victime nommée. L'appariement laissait
    // l'horloge brute ; l'origine les corrige comme les autres.
    const out = alignFeedByOrigin([kill(2_500, 'k1'), kill(8_500, 'k2')], 0, docAvecOrigine(), 500)
    expect(out.kills.map((k) => k.replayMs)).toEqual([2_000, 8_000])
  })

  it('remet le countdown retranché par la Match View avant de retrancher l\'origine', () => {
    const out = alignFeedByOrigin([kill(16_841)], T0, docAvecOrigine(), 500)
    expect(out.kills[0].replayMs).toBe(34_806) // 16 841 + 18 465 − 500
  })

  it('la mort sans kill fait une MORT NEUTRE ; une fin de vie revendiquée n\'en fait pas', () => {
    const out = alignFeedByOrigin(kills, 0, docAvecOrigine(), 500)
    expect(out.deaths).toEqual([{ replayMs: 5_000, xuid: 'W', kind: '', img: '', tinted: false }])
  })

  it('un survivant de fin de partie ne meurt pas dans le fil', () => {
    const out = alignFeedByOrigin(kills, 0, docAvecOrigine(), 500)
    expect(out.deaths.some((d) => d.xuid === 'S')).toBe(false)
  })

  it('un kill SANS victime au voisinage met son VETO sur la mort neutre (anti-doublon)', () => {
    const out = alignFeedByOrigin([...kills, kill(5_900, 'k3')], 0, docAvecOrigine(), 500)
    expect(out.deaths).toEqual([])
  })
})

describe('alignFeed — origine publiée, appariement en repli', () => {
  it("emploie l'origine dès que l'artefact la porte", () => {
    // Origine 500 ms ET aucune victime nommée : seul le chemin par origine sait recaler.
    const doc = testReplayDoc({ ...docSpec(), originMs: 500 })
    expect(alignFeed([kill(2_500, 'k1')], 0, doc).kills[0].replayMs).toBe(2_000)
  })

  it("retombe sur l'appariement quand l'artefact n'a pas d'origine (schéma antérieur)", () => {
    const out = alignFeed([kill(2_500, 'k1', 'V'), kill(8_500, 'k2', 'V')], 0, docWithLives())
    expect(out.offsetMs).toBe(500)
    expect(out.kills.map((k) => k.replayMs)).toEqual([2_000, 8_000])
  })

  it("une origine de ZÉRO est une mesure, pas une absence", () => {
    const doc = testReplayDoc({ ...docSpec(), originMs: 0 })
    expect(alignFeed([kill(2_500, 'k1')], 0, doc).kills[0].replayMs).toBe(2_500)
  })
})

/**
 * Le TYPE des morts neutres (lot 7.1). Rappel du montage : W meurt à 5 000 ms sur l'axe du
 * rejeu, et le document date ses morts sur l'horloge du FIL — donc à 5 500 ms quand l'origine
 * vaut 500 ms. C'est précisément ce décalage que la jointure doit absorber.
 */
describe('le type des morts sans revendication', () => {
  const chute = { xuid: 'W', feedMs: 5_500, kind: 'environment', img: '/s/env.png', tinted: true }
  const docTypé = (over: Record<string, unknown> = {}) =>
    testReplayDoc({ ...docSpec(), originMs: 500, neutralDeaths: [chute], ...over })
  const kills = [kill(2_500, 'k1', 'V'), kill(8_500, 'k2', 'V')]

  it("pose le type sur la ligne, en absorbant le décalage du fil", () => {
    const out = alignFeed(kills, 0, docTypé())
    expect(out.deaths).toEqual([
      { replayMs: 5_000, xuid: 'W', kind: 'environment', img: '/s/env.png', tinted: true },
    ])
  })

  it('absorbe AUSSI le décalage MESURÉ quand l\'artefact ne publie pas d\'origine', () => {
    // Repli : l'offset est mesuré à 500 ms par appariement, la table du document est datée
    // sur la même horloge du fil. Le résultat doit être identique au chemin nominal.
    const doc = testReplayDoc({ ...docSpec(), neutralDeaths: [chute] })
    expect(alignFeed(kills, 0, doc).deaths[0].kind).toBe('environment')
  })

  it("ne pose RIEN quand le document ne parle pas de ce joueur — jamais le type d'une autre mort", () => {
    const doc = docTypé({ neutralDeaths: [{ ...chute, xuid: 'V' }] })
    expect(alignFeed(kills, 0, doc).deaths[0]).toMatchObject({ xuid: 'W', kind: '', img: '' })
  })

  it("ne pose RIEN quand l'entrée est trop loin dans le temps (autre vie du même joueur)", () => {
    // 5 500 + 3 000 = au-delà de la tolérance d'appariement : c'est une autre mort.
    const doc = docTypé({ neutralDeaths: [{ ...chute, feedMs: 5_500 + 3_000 }] })
    expect(alignFeed(kills, 0, doc).deaths[0].kind).toBe('')
  })

  it("ne sert JAMAIS deux fois la même entrée", () => {
    // Deux morts neutres du même joueur, une seule entrée typée : la seconde reste neutre.
    const spec = docSpec()
    const doc = testReplayDoc({
      ...spec,
      tracks: [...spec.tracks, { ...spec.tracks[2], startFrame: 60, endFrame: 100 }],
      originMs: 500,
      neutralDeaths: [chute],
    })
    const typés = alignFeed(kills, 0, doc).deaths.filter((d) => d.kind !== '')
    expect(typés).toHaveLength(1)
  })
})

describe('collectMedalEvents', () => {
  const ev = (over: Partial<MatchHighlightEvent>): MatchHighlightEvent => ({
    event_type: 'medal',
    event_time_ms: 1_000,
    actor_xuid: 'x1',
    target_xuid: null,
    weapon_id: null,
    medal_name: 'No Scope',
    ...over,
  })

  it("lit les events medal avec leur identité résolue", () => {
    const [m] = collectMedalEvents([
      ev({
        actor_gamertag: 'JGtm',
        actor_team_id: 1,
        medal_label: 'Sans lunette',
        medal_description: 'Desc',
        medal_image_url: '/static/medals/1.png',
      }),
    ])
    expect(m).toMatchObject({
      tMs: 1_000,
      xuid: 'x1',
      gamertag: 'JGtm',
      teamID: 1,
      name: 'No Scope',
      label: 'Sans lunette',
      imageUrl: '/static/medals/1.png',
    })
  })

  it('écarte ce qui ne se montre pas : mauvais type, sans acteur, sans instant, sans nom', () => {
    expect(
      collectMedalEvents([
        ev({ event_type: 'kill' }),
        ev({ actor_xuid: null }),
        ev({ event_time_ms: null }),
        ev({ medal_name: null }),
      ]),
    ).toEqual([])
  })
})

describe('buildFeedEntries — rattachement des médailles', () => {
  it('rattache une médaille au kill du MÊME acteur à ≤ 500 ms', () => {
    const entries = buildFeedEntries([kill(10_000)], [medal(10_240)], 0)
    expect(entries).toHaveLength(1)
    expect(entries[0].kill?.medals.map((m) => m.name)).toEqual(['No Scope'])
  })

  it("une médaille d'un AUTRE acteur au même instant fait sa propre ligne", () => {
    const entries = buildFeedEntries([kill(10_000, 'x1')], [medal(10_000, 'x2')], 0)
    expect(entries).toHaveLength(2)
    expect(entries.find((e) => e.medal)?.medal?.xuid).toBe('x2')
  })

  it('au-delà de la tolérance, la médaille reste seule — jamais forcée', () => {
    const entries = buildFeedEntries([kill(10_000)], [medal(11_000)], 0)
    expect(entries).toHaveLength(2)
    expect(entries[0].kill?.medals).toEqual([])
    expect(entries[1].medal?.name).toBe('No Scope')
  })

  it('choisit le kill le PLUS PROCHE quand deux se disputent la médaille', () => {
    const entries = buildFeedEntries([kill(10_000), kill(10_400)], [medal(10_390)], 0)
    const porteur = entries.find((e) => (e.kill?.medals.length ?? 0) > 0)
    expect(porteur?.kill?.tMs).toBe(10_400)
  })

  it('la médaille seule est recalée sur la même horloge que les kills', () => {
    const entries = buildFeedEntries([], [medal(16_841)], T0)
    expect(entries[0].replayMs).toBe(35_306)
  })
})

describe('buildFeedEntries — avec document : pistes, morts neutres, une seule horloge', () => {
  const kills = [kill(2_500, 'k1', 'V'), kill(8_500, 'k2', 'V')]

  it('assemble kills alignés ET morts neutres, triés chronologiquement', () => {
    const entries = buildFeedEntries(kills, [], 0, docWithLives())
    expect(entries.map((e) => [e.replayMs, e.kill ? 'kill' : 'death'])).toEqual([
      [2_000, 'kill'],
      [5_000, 'death'],
      [8_000, 'kill'],
    ])
    expect(entries[1].death?.xuid).toBe('W')
  })

  it('la médaille seule suit la MÊME correction de décalage que les kills', () => {
    const entries = buildFeedEntries(kills, [medal(12_000, 'autre')], 0, docWithLives())
    const seule = entries.find((e) => e.medal)
    expect(seule?.replayMs).toBe(11_500)
  })
})

describe('feedAt — le fil est PERMANENT', () => {
  const entries = buildFeedEntries(
    [kill(10_000), kill(12_000), kill(14_000), kill(60_000)],
    [],
    0,
  )

  it("n'affiche RIEN de ce qui n'est pas encore arrivé", () => {
    const vus = feedAt(entries, 11_000)
    expect(vus.map((e) => e.replayMs)).toEqual([10_000])
  })

  it('GARDE TOUT ce qui est survenu, le plus récent en tête — aucune fenêtre', () => {
    // Le point du verdict user 2026-08-13 : à 59 s de rejeu, les kills de 10/12/14 s
    // sont TOUJOURS là. L'ancien feed les faisait disparaître après 8 s.
    const vus = feedAt(entries, 59_000)
    expect(vus.map((e) => e.replayMs)).toEqual([14_000, 12_000, 10_000])
  })

  it('fil complet en fin de match', () => {
    expect(feedAt(entries, 120_000)).toHaveLength(4)
  })

  it('vide avant le premier événement', () => {
    expect(feedAt(entries, 0)).toEqual([])
  })
})
