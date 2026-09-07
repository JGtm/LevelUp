/**
 * skullSound.test.ts — la regle PRIS / RAMASSE / LACHE du crane d'Oddball.
 *
 * Ce qui est verifie ici est ce qui peut DERIVER EN SILENCE : un rejeu qui annonce une prise
 * sur socle quand le crane roulait au sol, ou qui joue une chute que le match n'a pas eue
 * parce que le film s'est arrete pendant un portage.
 */
import { describe, expect, it } from 'vitest'

import { SKULL_SOUND_STEMS, skullSoundEvents } from './skullSound'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Un document reduit a ce que `skullSoundEvents` lit : les trois canaux et l'horloge. */
function doc(
  carries: { t0: number; t1: number; closed: boolean; xuid: string }[],
  lives: { t0: number; t1: number }[] = [],
  teams?: { teamId: number; total: { t: number; v: number }[] }[],
): ReplayDocumentReady {
  return {
    originMs: 0,
    coverage: { originResolved: true },
    skullCarries: carries,
    objectiveObjects: lives.map((l) => ({ ...l, pts: [], family: 'skull', fr: '', en: '' })),
    scoreTimeline: teams
      ? { teams: teams.map((t) => ({ ...t, rounds: [] })), players: [] }
      : undefined,
  } as unknown as ReplayDocumentReady
}

const stems = (d: ReplayDocumentReady) => skullSoundEvents(d).map((e) => e.stem)

describe('skullSoundEvents', () => {
  it('se tait sans aucune periode de portage', () => {
    expect(skullSoundEvents(doc([]))).toEqual([])
  })

  it('une prise qui suit un crane AU REPOS est un crane PRIS sur son socle', () => {
    // Une vie ou t0 === t1 est la signature du socle (cf. skullPresence.ts), et elle touche
    // la prise a la tolerance pres. La vie d'ouverture sonne l'apparition, puis le crane
    // retombe au sol (vie a 210) : c'est une chute, pas une disparition.
    const d = doc(
      [{ t0: 104, t1: 200, closed: true, xuid: 'a' }],
      [
        { t0: 100, t1: 100 },
        { t0: 210, t1: 260 },
      ],
    )
    expect(stems(d)).toEqual([
      SKULL_SOUND_STEMS.spawn,
      SKULL_SOUND_STEMS.taken,
      SKULL_SOUND_STEMS.dropped,
    ])
  })

  it('une prise qui suit une vie EN MOUVEMENT est un crane RAMASSE au sol', () => {
    const d = doc(
      [{ t0: 104, t1: 200, closed: true, xuid: 'a' }],
      [
        { t0: 60, t1: 100 },
        { t0: 210, t1: 260 },
      ],
    )
    expect(stems(d)).toEqual([
      SKULL_SOUND_STEMS.spawn,
      SKULL_SOUND_STEMS.pickup,
      SKULL_SOUND_STEMS.dropped,
    ])
  })

  it('un repos TROP ANCIEN ne decrit pas la prise en cours', () => {
    // Le crane a repose, puis le film a coule avant la prise : rien ne dit que le porteur
    // l'a prise sur ce socle-la.
    const d = doc(
      [{ t0: 150, t1: 200, closed: true, xuid: 'a' }],
      [
        { t0: 100, t1: 100 },
        { t0: 210, t1: 260 },
      ],
    )
    expect(stems(d)).toEqual([
      SKULL_SOUND_STEMS.spawn,
      SKULL_SOUND_STEMS.pickup,
      SKULL_SOUND_STEMS.dropped,
    ])
  })

  it('sans aucune vie publiee, la prise degrade vers RAMASSE', () => {
    const d = doc([{ t0: 104, t1: 200, closed: true, xuid: 'a' }])
    expect(stems(d)).toEqual([SKULL_SOUND_STEMS.pickup, SKULL_SOUND_STEMS.dropped])
  })

  it('une periode NON FERMEE ne joue pas la chute', () => {
    // `closed: false` = le FILM s'arrete pendant le portage. Personne n'a lache le crane.
    const d = doc([{ t0: 104, t1: 900, closed: false, xuid: 'a' }])
    expect(stems(d)).toEqual([SKULL_SOUND_STEMS.pickup])
  })

  it('un lacher SANS retombee est une sortie de carte, suivie du retour au socle', () => {
    // Le porteur meurt dans le vide : aucune vie ne s'ouvre derriere le lacher. Le crane
    // DISPARAIT (jamais `dropped` en meme temps), puis reapparait a la vie suivante.
    const d = doc([{ t0: 50, t1: 100, closed: true, xuid: 'a' }], [{ t0: 400, t1: 400 }])
    expect(stems(d)).toEqual([
      SKULL_SOUND_STEMS.pickup,
      SKULL_SOUND_STEMS.despawn,
      SKULL_SOUND_STEMS.spawn,
    ])
  })

  it('le nombre d apparitions est BORNE par les disparitions, pas par les vies', () => {
    // Huit vies (le crane rebondit), un seul lacher et aucune sortie de carte : une seule
    // apparition, celle de l'ouverture. C'est la garantie anti-mitraillette.
    const lives = Array.from({ length: 8 }, (_, i) => ({ t0: 200 + i * 20, t1: 215 + i * 20 }))
    const d = doc([{ t0: 50, t1: 190, closed: true, xuid: 'a' }], lives)
    expect(stems(d).filter((s) => s === SKULL_SOUND_STEMS.spawn)).toHaveLength(1)
    expect(stems(d)).not.toContain(SKULL_SOUND_STEMS.despawn)
  })

  it('chaque palier MONTANT du score est un ticket, dans le camp de son equipe', () => {
    const d = doc(
      [{ t0: 10, t1: 20, closed: false, xuid: 'a' }],
      [],
      [
        { teamId: 0, total: [{ t: 60, v: 1 }, { t: 120, v: 2 }] },
        { teamId: 1, total: [{ t: 180, v: 1 }] },
      ],
    )
    const marques = skullSoundEvents(d, 0).map((e) => e.stem).filter((s) => s.includes('tick'))
    // Le PREMIER palier de chaque serie n'a pas de precedent : il ouvre la serie. Seul le
    // palier montant qui SUIT un palier connu est un ticket.
    expect(marques).toEqual([SKULL_SOUND_STEMS.scoringTeam])
  })

  it('sans camp allie resolu, la marque se TAIT', () => {
    const d = doc(
      [{ t0: 10, t1: 20, closed: false, xuid: 'a' }],
      [],
      [{ teamId: 0, total: [{ t: 60, v: 1 }, { t: 120, v: 2 }] }],
    )
    expect(stems(d)).toEqual([SKULL_SOUND_STEMS.pickup])
  })

  it('hors Oddball (aucun portage), RIEN ne sonne — pas meme les paliers de score', () => {
    const d = doc([], [{ t0: 10, t1: 10 }], [{ teamId: 0, total: [{ t: 60, v: 1 }, { t: 120, v: 2 }] }])
    expect(skullSoundEvents(d, 0)).toEqual([])
  })

  it('date les evenements sur l horloge du rejeu, tries', () => {
    const d = doc([
      { t0: 200, t1: 260, closed: true, xuid: 'b' },
      { t0: 100, t1: 150, closed: true, xuid: 'a' },
    ])
    // L'horloge du rejeu court a 60 images/s : la prise a l'image 100 tombe a 1 666,7 ms.
    // Ce qui compte ici est l'ORDRE — les deux portages sont donnes a l'envers en entree.
    expect(skullSoundEvents(d).map((e) => Math.round(e.ms))).toEqual([1667, 2500, 3333, 4333])
  })

  it('livre toutes les variantes du RandomSequence, pas seulement la premiere', () => {
    const d = doc([{ t0: 104, t1: 200, closed: true, xuid: 'a' }])
    const chute = skullSoundEvents(d).find((e) => e.stem === SKULL_SOUND_STEMS.dropped)
    expect(chute?.variants).toHaveLength(3)
  })
})
