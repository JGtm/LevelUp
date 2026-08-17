/**
 * replaySound.test.ts — les règles de la piste sonore qui ne s'entendent pas à l'oreille :
 * l'horloge des kills est celle du fil (jamais l'horloge brute), une clé sans fichier est
 * un silence (jamais le son d'une voisine), et le curseur ne rejoue rien deux fois — ni
 * après un scrub, ni au rebouclage.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { ReplayDocument, ReplayGrenade } from '@/lib/api/types'

import { SOUND_CUT_MAX_S, SOUND_FADE_S, soundEnvelope } from './replayAudio'
import {
  advanceSoundCursor,
  buildSoundTimeline,
  killSound,
  killSourceSpriteStem,
  resyncSoundCursor,
  SOUND_RESYNC_JUMP_MS,
  type ReplaySoundEvent,
  type SoundCategoryFilter,
} from './replaySound'
import { testReplayDoc } from './test/testDoc'

/** URL de vignette telle que le backend la compose (adapter_asset_urls.go + static.URL). */
function vignette(sprite: string): string {
  return `/static/weapons-assets/halo_infinite/jeu/${sprite}.png`
}

/** Un kill minimal (même patron que killFx.test.ts) ; tMs est sur l'horloge gameplay. */
function kill(over: Partial<KillEvent> = {}): KillEvent {
  return {
    tMs: 2_000,
    xuid: 'K',
    ally: true,
    teamID: 0,
    weaponKey: 'hinf_br75',
    weaponLabel: 'BR75',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: 'V',
    victimGamertag: 'Victime',
    victimTeamID: 1,
    ...over,
  }
}

function grenade(over: Partial<ReplayGrenade> = {}): ReplayGrenade {
  return { i: 0, rank: 0, s: '', slot: 0, t: 10, x: 0, y: 0, ...over }
}

/** Document 10 Hz : vie du tueur (0-100) et de la victime (0-20) — fin de vie à 2 000 ms. */
function docWithCouple(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      {
        slot: 1,
        team: -1,
        xuid: 'K',
        points: [
          { t: 0, x: 0, y: 0 },
          { t: 100, x: 10, y: 0 },
        ],
        startFrame: 0,
        endFrame: 100,
      },
      {
        slot: 2,
        team: -1,
        xuid: 'V',
        points: [
          { t: 0, x: 5, y: 5 },
          { t: 20, x: 5, y: 4 },
        ],
        startFrame: 0,
        endFrame: 20,
      },
    ],
    ...over,
  })
}

/** Document 10 Hz portant `n` tirs consécutifs d'une arme du pack sonore (BR75). */
function docAvecTirs(n: number) {
  return testReplayDoc({
    frameIntervalMs: 100,
    shots: Array.from({ length: n }, (_, i) => ({ slot: 1, t: i, x: 0, y: 0, w: '0x2B1824D5' })),
    weaponLabels: { '0x2B1824D5': { en: 'BR75', fr: 'BR75', fx: 'ballistic', key: 'hinf_br75' } },
  })
}

describe('buildSoundTimeline', () => {
  it("pose le kill sur l'horloge du FIL (fin de vie de la victime), pas l'horloge brute", () => {
    // Kill servi 3 s après la fin de vie (décalage d'origine, même mesure que killFx) :
    // l'alignement le pose à 2 000 ms — là où le fil et la fiche le montrent.
    const tl = buildSoundTimeline(docWithCouple(), [kill({ tMs: 5_000 })], 0)
    expect(tl).toEqual([{ ms: 2_000, stem: 'hinf_br75' }])
  })

  it('sans weapon_key ou clé hors manifeste : silence — jamais le son d une voisine', () => {
    const tl = buildSoundTimeline(
      docWithCouple(),
      [kill({ weaponKey: '' }), kill({ weaponKey: 'hinf_mutilator' })],
      0,
    )
    expect(tl).toEqual([])
  })

  it("un kill À LA grenade sonne l'explosion DE SON TYPE (pas le geste du lancer)", () => {
    // La grenade à pointes n'a PAS de weapon_key (killicon GGGL 3) : sa vignette est la
    // seule donnée qui la nomme, et elle suffit — c'est tout l'objet de la jointure.
    const tl = buildSoundTimeline(
      docWithCouple(),
      [kill({ weaponKey: '', weaponImageUrl: vignette('killfeed-49') })],
      0,
    )
    expect(tl.map((e) => e.stem)).toEqual(['explosion_spike'])
  })

  it('un kill À LA MÊLÉE sonne le coup qui a tué (classe MELEE, sans arme nommée)', () => {
    const tl = buildSoundTimeline(
      docWithCouple(),
      [kill({ weaponKey: '', weaponLabel: '', weaponImageUrl: vignette('killfeed-65') })],
      0,
    )
    expect(tl.map((e) => e.stem)).toEqual(['melee_kill'])
  })

  it('les lancers sonnent par TYPE (rang -> stem), un rang hors table reste muet', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        grenades: [
          grenade({ t: 10, rank: 0 }),
          grenade({ t: 20, rank: 1 }),
          grenade({ t: 30, rank: 2 }),
          grenade({ t: 40, rank: 3 }),
          grenade({ t: 50, rank: 4 }),
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([
      { ms: 1_000, stem: 'throw_frag' },
      { ms: 2_000, stem: 'throw_plasma' },
      { ms: 3_000, stem: 'throw_dynamo' },
      { ms: 4_000, stem: 'throw_spike' },
    ])
  })

  it('kills et lancers fusionnent en une piste TRIÉE', () => {
    const tl = buildSoundTimeline(
      docWithCouple({ grenades: [grenade({ t: 50, rank: 0 })] }),
      [kill()],
      0,
    )
    expect(tl.map((e) => e.ms)).toEqual([2_000, 5_000])
  })

  it('CHAQUE tir sonne son arme — aucun filtrage de densité (décision du 2026-08-15)', () => {
    // Six tirs de la même arme en 600 ms : les six sonnent. Le seul plafond est technique
    // (voix simultanées, replayAudio.ts), et il ne vit pas dans cette table.
    const tl = buildSoundTimeline(docAvecTirs(6), [], 0)
    expect(tl.map((e) => e.stem)).toEqual(Array(6).fill('hinf_br75'))
    expect(tl.map((e) => e.ms)).toEqual([0, 100, 200, 300, 400, 500])
  })

  it('un tir dont l arme n a ni clé ni fichier reste MUET, jamais le son d une voisine', () => {
    // Trois cas de silence : arme sans identifiant, identifiant hors table de libellés,
    // et arme du registre absente du pack sonore (Bandit — mesure du lot 5).
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      shots: [
        { slot: 1, t: 0, x: 0, y: 0 },
        { slot: 1, t: 10, x: 0, y: 0, w: '0xINCONNU' },
        { slot: 1, t: 20, x: 0, y: 0, w: '0xMUT1L4' },
      ],
      weaponLabels: { '0xMUT1L4': { en: 'Mutilator', fr: 'Mutilator', key: 'hinf_mutilator' } },
    })
    expect(buildSoundTimeline(doc, [], 0)).toEqual([])
  })

  it('un libellé SANS clé ne sonne pas : la clé est posée à la requête, jamais devinée', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      shots: [{ slot: 1, t: 0, x: 0, y: 0, w: '0x2B1824D5' }],
      weaponLabels: { '0x2B1824D5': { en: 'BR75', fr: 'BR75', fx: 'ballistic' } },
    })
    expect(buildSoundTimeline(doc, [], 0)).toEqual([])
  })

  it('tirs, kills et lancers cohabitent sur UNE piste triée', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        shots: [{ slot: 1, t: 5, x: 0, y: 0, w: '0x2B1824D5' }],
        weaponLabels: { '0x2B1824D5': { en: 'BR75', fr: 'BR75', key: 'hinf_br75' } },
        grenades: [grenade({ t: 50, rank: 0 })],
      }),
      [kill()],
      0,
    )
    expect(tl.map((e) => e.ms)).toEqual([500, 2_000, 5_000])
  })

  it("un épisode d'équipement sonne l'activation au début et la désactivation à sa fin MESURÉE", () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentEpisodes: [
          { slot: 1, fam: 'camo', t0: 10, t1: 30, endRead: true },
          { slot: 1, fam: 'overshield', t0: 40, t1: 60, endRead: true },
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([
      { ms: 1_000, stem: 'camo_activate' },
      { ms: 3_000, stem: 'camo_deactivate' },
      { ms: 4_000, stem: 'overshield_activate' },
      { ms: 6_000, stem: 'overshield_deactivate' },
    ])
  })

  it('un épisode fermé par la MORT ne sonne PAS de désactivation : rien ne l’a mesurée', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentEpisodes: [{ slot: 2, fam: 'camo', t0: 5, t1: 20 }], // endRead absent = mort
      }),
      [],
      0,
    )
    expect(tl).toEqual([{ ms: 500, stem: 'camo_activate' }])
  })

  it("une famille d'équipement hors table reste muette — jamais le son d'une voisine", () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentEpisodes: [{ slot: 1, fam: 'grapple', t0: 10, t1: 30, endRead: true }],
      }),
      [],
      0,
    )
    expect(tl).toEqual([])
  })

  it('une POSE sonne SA famille au geste (t0), et rien à la disparition de l objet', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentPlacements: [
          { t0: 10, t1: 90, x: 0, y: 0, family: 'wall', id: '0x528fce46', owner: 1, origin: 'deployed' },
          { t0: 40, t1: 95, x: 1, y: 1, family: 'sensor', id: '0x72199cba', owner: 1, origin: 'deployed' },
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([
      { ms: 1_000, stem: 'wall_activate' },
      { ms: 4_000, stem: 'sensor_activate' },
    ])
  })

  it('une pose de famille sans fichier (objet non identifié) reste MUETTE', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentPlacements: [
          { t0: 10, t1: 90, x: 0, y: 0, family: 'other', id: '0x4396db42', owner: -1, origin: 'deployed' },
          { t0: 20, t1: 90, x: 0, y: 0, family: 'famille_future', id: '0x00528fce', owner: 2, origin: 'deployed' },
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([])
  })

  it('un objet LÂCHÉ À LA MORT ne sonne pas : ce n est pas un geste de pose', () => {
    // 88,6 % des poses du corpus sont des lâchers. Les sonner ferait partir un « mur déployé »
    // à chaque mort tenant un mur — la même règle que le calque, et elle est PARTAGÉE.
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentPlacements: [
          { t0: 10, t1: 90, x: 0, y: 0, family: 'wall', id: '0x528fce46', owner: 1, origin: 'dropped' },
          { t0: 20, t1: 90, x: 0, y: 0, family: 'sensor', id: '0x72199cba', owner: 1, origin: 'unknown' },
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([])
  })

  it("une pose SANS origine (artefact antérieur au schéma 10) ne sonne pas non plus", () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentPlacements: [
          { t0: 10, t1: 90, x: 0, y: 0, family: 'wall', id: '0x528fce46', owner: 1 },
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([])
  })

  it('un mur déployé sonne UNE fois : l appareil se tait, les panneaux sonnent', () => {
    // L'appareil qui vole et ses panneaux sont DEUX poses `deployed` du même mur.
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentPlacements: [
          { t0: 10, t1: 40, x: 0, y: 0, family: 'wall', id: '0x8e2dc574', owner: 1, origin: 'deployed' },
          { t0: 14, t1: 60, x: 2, y: 0, family: 'wall', id: '0x686b40c9', owner: 1, origin: 'deployed' },
        ],
      }),
      [],
      0,
    )
    expect(tl).toEqual([{ ms: 1_400, stem: 'wall_activate' }])
  })
})

describe('buildSoundTimeline — filtre par catégorie (tiroir de réglages, phase 2)', () => {
  const ALL_ON: SoundCategoryFilter = { weapon: true, grenade: true, melee: true, equipment: true }

  it('sans 4e paramètre, comportement INCHANGÉ : les quatre catégories sonnent', () => {
    const tl = buildSoundTimeline(docAvecTirs(2), [], 0)
    expect(tl.map((e) => e.stem)).toEqual(['hinf_br75', 'hinf_br75'])
  })

  it('catégorie ARMES coupée : ni le tir ni le kill d arme ne sonnent, la grenade reste', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        shots: [{ slot: 1, t: 5, x: 0, y: 0, w: '0x2B1824D5' }],
        weaponLabels: { '0x2B1824D5': { en: 'BR75', fr: 'BR75', key: 'hinf_br75' } },
        grenades: [grenade({ t: 50, rank: 0 })],
      }),
      [kill()], // kill à l'arme (hinf_br75)
      0,
      { ...ALL_ON, weapon: false },
    )
    expect(tl.map((e) => e.stem)).toEqual(['throw_frag'])
  })

  it('catégorie GRENADES coupée : ni le lancer ni l explosion ne sonnent, l arme reste', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        shots: [{ slot: 1, t: 5, x: 0, y: 0, w: '0x2B1824D5' }],
        weaponLabels: { '0x2B1824D5': { en: 'BR75', fr: 'BR75', key: 'hinf_br75' } },
        grenades: [grenade({ t: 50, rank: 0 })],
      }),
      [kill({ weaponKey: '', weaponImageUrl: vignette('killfeed-49') })], // kill à la grenade à pointes
      0,
      { ...ALL_ON, grenade: false },
    )
    expect(tl.map((e) => e.stem)).toEqual(['hinf_br75'])
  })

  it('catégorie MÊLÉE coupée : le coup fatal ne sonne plus', () => {
    const tl = buildSoundTimeline(
      docWithCouple(),
      [kill({ weaponKey: '', weaponLabel: '', weaponImageUrl: vignette('killfeed-65') })],
      0,
      { ...ALL_ON, melee: false },
    )
    expect(tl).toEqual([])
  })

  it('catégorie ÉQUIPEMENTS coupée : ni activation, ni désactivation, ni POSE ne sonnent', () => {
    const tl = buildSoundTimeline(
      docWithCouple({
        equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 30, endRead: true }],
        equipmentPlacements: [
          { t0: 12, t1: 90, x: 0, y: 0, family: 'wall', id: '0x528fce46', owner: 1, origin: 'deployed' },
        ],
      }),
      [],
      0,
      { ...ALL_ON, equipment: false },
    )
    expect(tl).toEqual([])
  })

  it('toutes catégories coupées : silence total, jamais une erreur', () => {
    const tl = buildSoundTimeline(
      docWithCouple({ grenades: [grenade({ t: 50, rank: 0 })] }),
      [kill()],
      0,
      { weapon: false, grenade: false, melee: false, equipment: false },
    )
    expect(tl).toEqual([])
  })
})

describe('killSourceSpriteStem', () => {
  it("rend le stem de la vignette, et rien quand il n'y a rien à lire", () => {
    expect(killSourceSpriteStem(vignette('killfeed-46'))).toBe('killfeed-46')
    expect(killSourceSpriteStem('killfeed-65.png')).toBe('killfeed-65')
    expect(killSourceSpriteStem(vignette('killfeed-46') + '?v=2')).toBe('killfeed-46')
    expect(killSourceSpriteStem('')).toBe('')
    expect(killSourceSpriteStem(undefined)).toBe('')
    expect(killSourceSpriteStem('/static/sounds/halo_infinite/melee_kill.wav')).toBe('')
  })
})

// killSound remplace la façade `killSoundStem` (supprimée le 2026-08-16, elle n'avait plus
// d'appelant de production) : MÊMES quatre cas, plus la CATÉGORIE, qui est ce que le filtre
// du tiroir coupe. Sans elle, une grenade rangée en « weapon » se tairait avec les armes.
describe('killSound', () => {
  it('les QUATRE grenades sonnent chacune la sienne (ordre du dépôt : frag/plasma/dynamo/spike)', () => {
    const sounds = ['killfeed-46', 'killfeed-47', 'killfeed-48', 'killfeed-49'].map((s) =>
      killSound({ weaponKey: '', weaponImageUrl: vignette(s) }),
    )
    expect(sounds).toEqual([
      { stem: 'explosion_frag', category: 'grenade' },
      { stem: 'explosion_plasma', category: 'grenade' },
      { stem: 'explosion_dynamo', category: 'grenade' },
      { stem: 'explosion_spike', category: 'grenade' },
    ])
  })

  it('la VIGNETTE prime sur la clé : une grenade qui a les deux ne sonne pas deux vérités', () => {
    expect(
      killSound({
        weaponKey: 'hinf_frag_grenade',
        weaponImageUrl: vignette('killfeed-46'),
      }),
    ).toEqual({ stem: 'explosion_frag', category: 'grenade' })
    // Et la clé de grenade, seule, ne sonne plus rien : le pack n'a pas d'explosion partagée.
    expect(killSound({ weaponKey: 'hinf_frag_grenade', weaponImageUrl: '' })).toBeUndefined()
  })

  it("une ARME garde son son : sa vignette n'est pas dans la table, la clé répond", () => {
    expect(killSound({ weaponKey: 'hinf_br75', weaponImageUrl: vignette('killfeed-00') })).toEqual({
      stem: 'hinf_br75',
      category: 'weapon',
    })
  })

  it('source sans vignette ET sans clé (étiquette AMBIGU) : silence, jamais une voisine', () => {
    expect(killSound({ weaponKey: '', weaponImageUrl: '' })).toBeUndefined()
  })

  it('vignette hors table sonore et sans clé (le répulseur) : SILENCE, pas une voisine', () => {
    // Le cas du lot R2.4 : l'atlas du jeu porte bien `killfeed-56` (repulsor), mais aucune
    // source de dégât mesurée n'y mène et aucun son ne lui est associé. Si une vignette
    // arrivait un jour sans entrée dans la table, elle doit se taire — jamais retomber sur
    // l'explosion ou la mêlée d'à côté.
    expect(killSound({ weaponKey: '', weaponImageUrl: vignette('killfeed-56') })).toBeUndefined()
  })
})

const TL: ReplaySoundEvent[] = [
  { ms: 1_000, stem: 'a' },
  { ms: 1_200, stem: 'b' },
  { ms: 5_000, stem: 'c' },
]

describe('curseur sonore', () => {
  it('resync pose le curseur APRÈS les événements déjà passés', () => {
    expect(resyncSoundCursor(TL, 0)).toEqual({ ms: 0, idx: 0 })
    expect(resyncSoundCursor(TL, 1_000)).toEqual({ ms: 1_000, idx: 1 })
    expect(resyncSoundCursor(TL, 9_000)).toEqual({ ms: 9_000, idx: 3 })
  })

  it('lecture continue : chaque événement part UNE fois, à son passage', () => {
    let cur = resyncSoundCursor(TL, 900)
    const step1 = advanceSoundCursor(TL, cur, 1_100)
    expect(step1.fire.map((e) => e.stem)).toEqual(['a'])
    cur = step1.cursor
    const step2 = advanceSoundCursor(TL, cur, 1_200) // borne incluse
    expect(step2.fire.map((e) => e.stem)).toEqual(['b'])
    const step3 = advanceSoundCursor(TL, step2.cursor, 1_900)
    expect(step3.fire).toEqual([])
  })

  it('saut long en avant (scrub) : recalage SILENCIEUX, rien ne part', () => {
    const cur = resyncSoundCursor(TL, 0)
    const jumped = advanceSoundCursor(TL, cur, 1_500) // saut > SOUND_RESYNC_JUMP_MS
    expect(jumped.fire).toEqual([])
    expect(jumped.cursor.idx).toBe(2) // a (1 000) et b (1 200) enjambés, jamais rejoués
  })

  it('un saut JUSTE sous le seuil reste une lecture continue : ce qui est passé sonne', () => {
    const cur = resyncSoundCursor(TL, 0)
    const step = advanceSoundCursor(TL, cur, SOUND_RESYNC_JUMP_MS)
    expect(step.fire.map((e) => e.stem)).toEqual(['a'])
  })

  it('recul (rebouclage, scrub arrière) : recalage silencieux, puis la suite rejoue', () => {
    const back = advanceSoundCursor(TL, resyncSoundCursor(TL, 6_000), 100)
    expect(back.fire).toEqual([])
    const next = advanceSoundCursor(TL, back.cursor, 1_050)
    expect(next.fire.map((e) => e.stem)).toEqual(['a'])
  })
})

describe('soundEnvelope', () => {
  it('un son joue jusqu au bout de son fichier, fondu sur le dernier quart de seconde', () => {
    // 3,33 s : la durée de l explosion de fragmentation livrée (lot R2.3).
    expect(soundEnvelope(3.33)).toEqual({ fadeStartS: 3.33 - SOUND_FADE_S, stopS: 3.33 })
  })

  it('au-delà du plafond de sûreté, la coupe reprend la main', () => {
    expect(soundEnvelope(30)).toEqual({
      fadeStartS: SOUND_CUT_MAX_S - SOUND_FADE_S,
      stopS: SOUND_CUT_MAX_S,
    })
  })

  it('un son court joue entier, le fondu est borné à sa moitié (pas de claquement)', () => {
    expect(soundEnvelope(0.3)).toEqual({ fadeStartS: 0.15, stopS: 0.3 })
  })
})
