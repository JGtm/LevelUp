/**
 * replaySound.test.ts — les règles de la piste sonore qui ne s'entendent pas à l'oreille :
 * l'horloge des kills est celle du fil (jamais l'horloge brute), une clé sans fichier est
 * un silence (jamais le son d'une voisine), et le curseur ne rejoue rien deux fois — ni
 * après un scrub, ni au rebouclage.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { ReplayDocument, ReplayGrenade } from '@/lib/api/types'

import { SOUND_CUT_S, SOUND_FADE_S, soundEnvelope } from './replayAudio'
import {
  advanceSoundCursor,
  buildSoundTimeline,
  resyncSoundCursor,
  SOUND_RESYNC_JUMP_MS,
  type ReplaySoundEvent,
} from './replaySound'
import { testReplayDoc } from './test/testDoc'

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
      [kill({ weaponKey: '' }), kill({ weaponKey: 'hinf_bandit' })],
      0,
    )
    expect(tl).toEqual([])
  })

  it("un kill À LA grenade sonne l'explosion (c'est elle qui a tué, pas le lancer)", () => {
    const tl = buildSoundTimeline(docWithCouple(), [kill({ weaponKey: 'hinf_frag_grenade' })], 0)
    expect(tl.map((e) => e.stem)).toEqual(['explosion'])
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
        { slot: 1, t: 20, x: 0, y: 0, w: '0xB4ND1T' },
      ],
      weaponLabels: { '0xB4ND1T': { en: 'M392 Bandit', fr: 'Bandit EVO', key: 'hinf_bandit' } },
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
  it('un son long est coupé à ~1 s, fondu entamé un quart de seconde avant', () => {
    expect(soundEnvelope(3)).toEqual({ fadeStartS: SOUND_CUT_S - SOUND_FADE_S, stopS: SOUND_CUT_S })
  })

  it('un son court joue entier, le fondu est borné à sa moitié (pas de claquement)', () => {
    expect(soundEnvelope(0.3)).toEqual({ fadeStartS: 0.15, stopS: 0.3 })
  })
})
