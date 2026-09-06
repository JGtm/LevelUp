/**
 * matchClock.test.ts — les conversions entre les trois axes, sur des oracles écrits à la main.
 *
 * AUCUNE VALEUR ATTENDUE N'EST RECOPIÉE DU CODE : chaque nombre ci-dessous est posé à part,
 * en repartant des grandeurs du contrat (`msFilm = event_time_ms + t0_ms − originMs`) et des
 * témoins mesurés du dépôt (e94163af : origine 39 772 ms > T0 35 238 ms ; 000d5950 : origine
 * 3 604 ms, T0 de 18 à 28 s). Un test qui réécrirait la formule du module ne prouverait rien.
 */
import { describe, expect, it } from 'vitest'

import { matchClock, type MatchClockDocument } from './matchClock'

/**
 * LE DOCUMENT TÉMOIN. Un film de 4 985 images à 100 ms (498,4 s), dont l'image zéro tombe
 * 12 s après le début du match, sur un match dont le countdown a duré 30 s. Le coup d'envoi
 * tombe donc 18 s APRÈS l'image zéro — soit à l'image 180 tout rond.
 */
const DOC: MatchClockDocument = {
  originMs: 12_000,
  t0FilmMs: 31_500,
  frameIntervalMs: 100,
  frameCount: 4985,
}
const HEADER = { t0_ms: 30_000 }

describe('matchClock — quand l’horloge s’établit, et quand elle ne s’établit pas', () => {
  it('s’établit sur un document complet', () => {
    const c = matchClock(DOC, HEADER)
    expect(c).not.toBeNull()
    expect(c?.originMs).toBe(12_000)
    expect(c?.t0Ms).toBe(30_000)
    expect(c?.frameIntervalMs).toBe(100)
    expect(c?.frameCount).toBe(4985)
  })

  it('rend null sans document', () => {
    expect(matchClock(null, HEADER)).toBeNull()
    expect(matchClock(undefined, HEADER)).toBeNull()
  })

  it('rend null SANS ORIGINE : l’écart entre les deux axes vaut alors 3,6 à 50,8 s d’inconnu', () => {
    expect(matchClock({ ...DOC, originMs: undefined }, HEADER)).toBeNull()
  })

  it('rend null sans échelle temporelle — une image ne se convertit pas en secondes', () => {
    expect(matchClock({ ...DOC, frameIntervalMs: undefined }, HEADER)).toBeNull()
    expect(matchClock({ ...DOC, frameIntervalMs: 0 }, HEADER)).toBeNull()
  })

  it('rend null à moins de deux images : rien à mettre bout à bout', () => {
    expect(matchClock({ ...DOC, frameCount: 1 }, HEADER)).toBeNull()
    expect(matchClock({ ...DOC, frameCount: 0 }, HEADER)).toBeNull()
  })

  it('porte le T0 FILM à côté de celui de l’API, sans s’y ancrer', () => {
    expect(matchClock(DOC, HEADER)?.t0FilmMs).toBe(31_500)
    expect(matchClock({ ...DOC, t0FilmMs: undefined }, HEADER)?.t0FilmMs).toBeNull()
  })
})

describe('matchClock — l’axe du gameplay, ancré sur le coup d’envoi', () => {
  it('place le coup d’envoi à l’image 180 : (30 000 − 12 000) / 100', () => {
    const c = matchClock(DOC, HEADER)!
    expect(c.gameplayMsOfFrame(180)).toBe(0)
  })

  it('rend NÉGATIF tout ce qui précède le coup d’envoi — l’image zéro, ici, 18 s avant', () => {
    const c = matchClock(DOC, HEADER)!
    expect(c.gameplayMsOfFrame(0)).toBe(-18_000)
  })

  it('date la dernière image du film à 480,4 s de gameplay (498,4 s de film − 18 s)', () => {
    const c = matchClock(DOC, HEADER)!
    expect(c.gameplayMsOfFrame(4984)).toBe(480_400)
  })

  it('convertit un instant du film sans passer par les images', () => {
    const c = matchClock(DOC, HEADER)!
    // 39,9 s de film = 51,9 s de match = 21,9 s après un coup d'envoi tombé à 30 s.
    expect(c.gameplayMsOfFilmMs(39_900)).toBe(21_900)
  })
})

describe('matchClock — l’identité qui met les deux graphes sur le même axe', () => {
  /**
   * LE CONTRAT GO, APPLIQUÉ À LA MAIN : `msFilm = event_time_ms + t0_ms − originMs`. Si
   * l'axe du gameplay est bien celui de `event_time_ms`, alors repasser un event par le film
   * puis par l'horloge doit rendre l'event INCHANGÉ. C'est cette identité, et elle seule, qui
   * autorise « Frags cumulés » (axe `event_time_ms`) et la courbe de score (axe des images) à
   * se lire l'un sous l'autre.
   */
  const filmMsOfEvent = (eventMs: number) => eventMs + HEADER.t0_ms - (DOC.originMs as number)

  it('un event repassé par le film revient à lui-même', () => {
    const c = matchClock(DOC, HEADER)!
    for (const eventMs of [0, 1, 21_900, 240_000, 480_400]) {
      expect(c.gameplayMsOfFilmMs(filmMsOfEvent(eventMs))).toBe(eventMs)
    }
  })

  it('un event à 0 tombe exactement sur l’image du coup d’envoi', () => {
    const c = matchClock(DOC, HEADER)!
    expect(filmMsOfEvent(0)).toBe(18_000)
    expect(c.gameplayMsOfFilmMs(18_000)).toBe(0)
  })
})

describe('matchClock — les témoins mesurés du dépôt', () => {
  it('e94163af : le film commence 4,5 s APRÈS le coup d’envoi, l’écart est positif', () => {
    // Origine 39 772 ms > T0 35 238 ms (replayWindow.ts:26) : le film n'a rien à montrer
    // avant sa propre image zéro, qui se lit donc déjà 4,534 s de gameplay.
    const c = matchClock({ ...DOC, originMs: 39_772 }, { t0_ms: 35_238 })!
    expect(c.gameplayMsOfFrame(0)).toBe(4_534)
  })

  it('000d5950 : origine 3 604 ms sous un countdown de 21 s, l’image zéro est 17,4 s avant', () => {
    const c = matchClock({ ...DOC, originMs: 3_604 }, { t0_ms: 21_000 })!
    expect(c.gameplayMsOfFrame(0)).toBe(-17_396)
  })

  it('T0 INCONNU (absent ou nul) : les deux axes retombent sur celui du MATCH', () => {
    // Le serveur n'a alors rien retranché aux events non plus — `event_time_ms` compte
    // depuis `start_time`, et l'axe du gameplay doit faire pareil : + originMs, rien de plus.
    const sansT0 = matchClock(DOC, {})!
    expect(sansT0.t0Ms).toBe(0)
    expect(sansT0.gameplayMsOfFrame(0)).toBe(12_000)
    expect(sansT0.gameplayMsOfFilmMs(39_900)).toBe(51_900)
    expect(matchClock(DOC, null)!.gameplayMsOfFrame(0)).toBe(12_000)
  })
})
