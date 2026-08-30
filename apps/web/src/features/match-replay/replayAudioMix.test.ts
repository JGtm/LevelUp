/**
 * replayAudioMix.test.ts — CE QUE LE MIXAGE RETIENT, ET DANS QUEL ORDRE.
 *
 * Web Audio n'existe pas sous jsdom : ce qui se teste ici est la partie DÉCIDÉE — quels sons
 * sont retenus, avec quelle variante, et lesquels le plafond de voix refuse. Le rendu lui-même
 * (`OfflineAudioContext`) et l'encodage AAC se vérifient à l'oreille sur un clip, et c'est le
 * gate de recette de l'étape E4 du plan.
 *
 * LE POINT LE PLUS IMPORTANT DE CE FICHIER est la REPRODUCTIBILITÉ : deux exports du même match
 * doivent sonner pareil, sans quoi on ne peut ni comparer deux versions d'un clip, ni le
 * re-livrer à l'identique.
 */
import { describe, expect, it } from 'vitest'

import {
  applyVoiceCap,
  mixSeed,
  planAudioMix,
  seededRandom,
  tailSeconds,
  type MixedSound,
} from './replayAudioMix'
import { SOUND_MAX_VOICES } from './replayAudio'
import type { ReplaySoundEvent } from './replaySoundVariants'

const BOUNDS = { startMs: 1000, endMs: 5000 }
const OPTS = { variationPercent: 0 }

/** Une piste avec un geste à variantes et deux gestes nus, dont un HORS de la plage. */
const TIMELINE: ReplaySoundEvent[] = [
  { ms: 500, stem: 'avant_la_plage' },
  { ms: 1200, stem: 'tir_a', variants: ['tir_a', 'tir_a_2', 'tir_a_3'] },
  { ms: 2400, stem: 'explosion' },
  { ms: 4800, stem: 'tir_a', variants: ['tir_a', 'tir_a_2', 'tir_a_3'] },
  { ms: 9000, stem: 'apres_la_plage' },
]

describe('seededRandom / mixSeed — le hasard devient reproductible', () => {
  it('deux générateurs de même graine rendent la même suite', () => {
    const a = seededRandom(1234)
    const b = seededRandom(1234)
    expect([a(), a(), a()]).toEqual([b(), b(), b()])
  })

  it('deux graines différentes divergent', () => {
    expect(seededRandom(1)()).not.toBe(seededRandom(2)())
  })

  it('la graine dépend du RANG ET du stem', () => {
    expect(mixSeed(0, 'tir_a')).not.toBe(mixSeed(1, 'tir_a'))
    expect(mixSeed(0, 'tir_a')).not.toBe(mixSeed(0, 'tir_b'))
  })

  it('la graine d’un rang donné est STABLE d’un appel à l’autre', () => {
    expect(mixSeed(7, 'explosion')).toBe(mixSeed(7, 'explosion'))
  })
})

describe('planAudioMix — ce qui entre dans le clip', () => {
  it('ne retient que les sons de la plage', () => {
    const out = planAudioMix(TIMELINE, BOUNDS, OPTS)
    expect(out).toHaveLength(3)
    expect(out.some((s) => s.stem.startsWith('avant'))).toBe(false)
    expect(out.some((s) => s.stem.startsWith('apres'))).toBe(false)
  })

  it('replace les instants sur l’axe du CLIP, pas sur celui du rejeu', () => {
    // Le premier son du match exporté tombe a 1200 - 1000 = 200 ms du debut du fichier.
    expect(planAudioMix(TIMELINE, BOUNDS, OPTS)[0].atMs).toBe(200)
  })

  it('rend EXACTEMENT le même mixage deux fois de suite', () => {
    // La garantie centrale du module : deux exports du même match sonnent pareil.
    expect(planAudioMix(TIMELINE, BOUNDS, OPTS)).toEqual(planAudioMix(TIMELINE, BOUNDS, OPTS))
  })

  it('deux occurrences du MÊME geste peuvent tirer des variantes différentes', () => {
    // C'est tout l'objet des variantes : un geste répété ne doit pas sonner comme une boucle.
    // Les deux tirs partagent le stem mais pas le rang, donc pas la graine.
    const out = planAudioMix(TIMELINE, BOUNDS, OPTS)
    const tirs = out.filter((s) => s.stem.startsWith('tir_a'))
    expect(tirs).toHaveLength(2)
    expect(mixSeed(1, 'tir_a')).not.toBe(mixSeed(3, 'tir_a'))
  })

  it('le tirage reste dans la liste des variantes déclarées', () => {
    for (const s of planAudioMix(TIMELINE, BOUNDS, OPTS)) {
      expect(['tir_a', 'tir_a_2', 'tir_a_3', 'explosion']).toContain(s.stem)
    }
  })

  it('borner la plage AUTREMENT ne change pas le tirage des sons communs', () => {
    // Le rang est celui de la piste ENTIÈRE : exporter une manche seule doit faire sonner ses
    // tirs exactement comme dans l'export du match entier.
    const large = planAudioMix(TIMELINE, { startMs: 0, endMs: 10_000 }, OPTS)
    const etroit = planAudioMix(TIMELINE, BOUNDS, OPTS)
    const communs = large.filter((s) => s.atMs >= 1000 && s.atMs <= 5000).map((s) => s.stem)
    expect(etroit.map((s) => s.stem)).toEqual(communs)
  })

  it('pose les prises de fin de partie SUR la borne de fin', () => {
    const out = planAudioMix(TIMELINE, BOUNDS, { ...OPTS, endMatchStems: ['fanfare'] })
    const fin = out.find((s) => s.stem === 'fanfare')
    expect(fin?.atMs).toBe(4000)
    // Aucune variation sur une fanfare : les fourchettes RANGED sont celles des armes.
    expect(fin?.draw).toEqual({ gainDb: 0, playbackRate: 1 })
  })

  it('rend la liste triée par instant', () => {
    const out = planAudioMix(TIMELINE, BOUNDS, { ...OPTS, endMatchStems: ['fanfare'] })
    expect(out.map((s) => s.atMs)).toEqual([...out.map((s) => s.atMs)].sort((a, b) => a - b))
  })
})

/** Fabrique N sons simultanés — de quoi saturer le plafond de voix. */
function simultanes(n: number, atMs = 0): MixedSound[] {
  return Array.from({ length: n }, (_, i) => ({
    atMs,
    stem: `s${i}`,
    draw: { gainDb: 0, playbackRate: 1 },
    family: 'sfx' as const,
  }))
}

describe('applyVoiceCap — la même comptabilité que le lecteur temps réel', () => {
  const UNE_SECONDE = () => 1

  it('laisse passer tant que le plafond n’est pas atteint', () => {
    expect(applyVoiceCap(simultanes(SOUND_MAX_VOICES), UNE_SECONDE)).toHaveLength(SOUND_MAX_VOICES)
  })

  it('refuse au-delà du plafond', () => {
    // Sans cela l'export sonnerait plus fort et plus confus que la page.
    expect(applyVoiceCap(simultanes(SOUND_MAX_VOICES + 5), UNE_SECONDE)).toHaveLength(
      SOUND_MAX_VOICES,
    )
  })

  it('libère les voix quand les sons se sont tus', () => {
    const serres = simultanes(SOUND_MAX_VOICES, 0)
    const plusTard = simultanes(3, 5000)
    expect(applyVoiceCap([...serres, ...plusTard], UNE_SECONDE)).toHaveLength(
      SOUND_MAX_VOICES + 3,
    )
  })

  it('un asset absent est un silence, et n’occupe AUCUNE voix', () => {
    const sons = [...simultanes(2, 0), { atMs: 0, stem: 'absent', draw: { gainDb: 0, playbackRate: 1 }, family: 'sfx' as const }]
    const out = applyVoiceCap(sons, (stem) => (stem === 'absent' ? null : 1))
    expect(out.map((s) => s.stem)).toEqual(['s0', 's1'])
  })
})

describe('tailSeconds — ce qui DÉPASSE la borne', () => {
  const buffers = new Map([['fanfare', { duration: 10 } as AudioBuffer]])

  it('rend le dépassement, pas la fin absolue', () => {
    // Une fanfare de 10 s posée a 4 s d'un clip de 5 s deborde de 9 s, pas de 14.
    const sons: MixedSound[] = [{ atMs: 4000, stem: 'fanfare', draw: { gainDb: 0, playbackRate: 1 }, family: 'music' }]
    expect(tailSeconds(sons, buffers, 5000)).toBe(9)
  })

  it('rend zéro quand tout tient dans la plage', () => {
    const sons: MixedSound[] = [{ atMs: 0, stem: 'fanfare', draw: { gainDb: 0, playbackRate: 1 }, family: 'music' }]
    expect(tailSeconds(sons, buffers, 60_000)).toBe(0)
  })
})
