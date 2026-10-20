/**
 * Tests — presenceTrackLogic (l'ombrage de présence de la frise, lot L4 du 2026-09-07).
 *
 * Ce qu'ils protègent :
 *  1. LE SENS DE L'OMBRE. Un arrivant s'ombre en TÊTE, un partant en QUEUE. L'inversion serait
 *     invisible à la relecture (deux bandes grises se ressemblent) et fausse à l'écran : elle
 *     dirait présent celui qui manquait.
 *  2. LA FRONTIÈRE (`edge`) EST LE BORD QUI TOUCHE LE JEU — c'est elle qui porte le glyphe.
 *  3. LE SILENCE EST UNE RÉPONSE. Aucun événement de présence = aucune ombre (le joueur était
 *     là de bout en bout), et un fil sans horloge n'en porte aucun : l'ombrage est alors ABSENT,
 *     pas faux.
 *  4. LES PALIERS DE LA PISTE COÉQUIPIERS comptent l'effectif MANQUANT sur un dénominateur fixe,
 *     fondent les tranches voisines de même compte et n'émettent jamais un palier à zéro.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayFeedEntry } from './killFeedLogic'
import type { PresenceEvent } from './presenceFeed'
import { presenceShades, teammatesAbsence } from './presenceTrackLogic'
import type { TrackScale } from './replayTimelineTracksLogic'

/** 20 images par seconde de film : un instant en ms se lit directement en images. */
const FRAME_MS = 50
/** Une frise de 200 images (10 s) ouverte à l'image 0 : un ratio r vaut 200·r images. */
const SCALE: TrackScale = { from: 0, span: 200 }
const clockOf = (ms: number) => `t${ms}`

function ligne(over: Partial<PresenceEvent> & { replayMs: number }): ReplayFeedEntry {
  const { replayMs, ...rest } = over
  const presence: PresenceEvent = {
    kind: 'joined',
    xuid: 'me',
    name: 'JGtm',
    bot: false,
    source: 'api',
    ...rest,
  }
  return {
    key: `p-${presence.kind}-${presence.xuid}-${replayMs}`,
    replayMs,
    kill: null,
    medal: null,
    death: null,
    presence,
  }
}

/** Une ligne de fil ORDINAIRE (ni entrée ni sortie) : l'ombrage doit l'ignorer. */
function autreLigne(replayMs: number): ReplayFeedEntry {
  return { key: `k-${replayMs}`, replayMs, kill: null, medal: null, death: null }
}

describe('presenceShades — l’ombre d’un joueur', () => {
  it('un arrivant s’ombre en TÊTE, du coup d’envoi à son entrée', () => {
    const [ombre] = presenceShades([ligne({ replayMs: 2_500 })], 'me', FRAME_MS, SCALE, clockOf)
    expect(ombre).toMatchObject({ from: 0, to: 0.25, edge: 0.25, kind: 'joined', source: 'api' })
  })

  it('un partant s’ombre en QUEUE, de son départ à la fin', () => {
    const entries = [ligne({ replayMs: 7_500, kind: 'left' })]
    const [ombre] = presenceShades(entries, 'me', FRAME_MS, SCALE, clockOf)
    expect(ombre).toMatchObject({ from: 0.75, to: 1, edge: 0.75, kind: 'left' })
  })

  it('l’image du glyphe et son horloge accompagnent l’ombre — le clic n’a rien à recalculer', () => {
    const [ombre] = presenceShades([ligne({ replayMs: 2_500 })], 'me', FRAME_MS, SCALE, clockOf)
    expect(ombre.frame).toBe(50)
    expect(ombre.clock).toBe('t2500')
  })

  it('arrivé PUIS parti : deux ombres, chacune sur sa frontière', () => {
    const entries = [
      ligne({ replayMs: 2_000 }),
      ligne({ replayMs: 8_000, kind: 'left' }),
    ]
    const ombres = presenceShades(entries, 'me', FRAME_MS, SCALE, clockOf)
    expect(ombres.map((o) => [o.from, o.to])).toEqual([
      [0, 0.2],
      [0.8, 1],
    ])
  })

  it('la source du film voyage jusqu’au rendu — c’est elle qui dégradera le bord', () => {
    const entries = [ligne({ replayMs: 2_000, source: 'film' })]
    expect(presenceShades(entries, 'me', FRAME_MS, SCALE, clockOf)[0].source).toBe('film')
  })

  it('aucun événement de présence = aucune ombre : le joueur était là de bout en bout', () => {
    expect(presenceShades([autreLigne(3_000)], 'me', FRAME_MS, SCALE, clockOf)).toEqual([])
  })

  it('les lignes des AUTRES joueurs n’ombrent pas cette piste', () => {
    const entries = [ligne({ replayMs: 2_000, xuid: 'autre' })]
    expect(presenceShades(entries, 'me', FRAME_MS, SCALE, clockOf)).toEqual([])
  })

  it('sans point de vue résolu, aucune ombre — on n’affirme rien sur personne', () => {
    expect(presenceShades([ligne({ replayMs: 2_000 })], null, FRAME_MS, SCALE, clockOf)).toEqual([])
  })

  it('une arrivée au coup d’envoi exact n’ombre rien : une bande de largeur nulle est écartée', () => {
    expect(presenceShades([ligne({ replayMs: 0 })], 'me', FRAME_MS, SCALE, clockOf)).toEqual([])
  })

  it('une échelle dégénérée n’ombre rien (film d’une image, fenêtre vide)', () => {
    const degenere: TrackScale = { from: 0, span: 0 }
    expect(presenceShades([ligne({ replayMs: 2_000 })], 'me', FRAME_MS, degenere, clockOf)).toEqual([])
  })
})

describe('teammatesAbsence — les paliers de la piste Coéquipiers', () => {
  const QUATRE = ['a', 'b', 'c', 'd']

  it('quatre coéquipiers, un partant à 0:03 et un arrivant à 0:06 : TROIS paliers', () => {
    // Le partant s'en va avant que l'arrivant n'arrive : l'effectif descend à 2 manquants entre
    // les deux instants, puis remonte. C'est le cas qui distingue un vrai escalier d'une somme.
    const entries = [
      ligne({ replayMs: 3_000, kind: 'left', xuid: 'a' }),
      ligne({ replayMs: 6_000, xuid: 'b' }),
    ]
    expect(teammatesAbsence(entries, QUATRE, FRAME_MS, SCALE)).toMatchObject([
      { from: 0, to: 0.3, absent: 1, total: 4 },
      { from: 0.3, to: 0.6, absent: 2, total: 4 },
      { from: 0.6, to: 1, absent: 1, total: 4 },
    ])
  })

  it('deux absences disjointes : le milieu à effectif complet n’est PAS un palier', () => {
    const entries = [
      ligne({ replayMs: 2_000, xuid: 'a' }),
      ligne({ replayMs: 8_000, kind: 'left', xuid: 'b' }),
    ]
    expect(teammatesAbsence(entries, QUATRE, FRAME_MS, SCALE)).toMatchObject([
      { from: 0, to: 0.2, absent: 1 },
      { from: 0.8, to: 1, absent: 1 },
    ])
  })

  it('deux absences qui se touchent au même compte fondent en UN palier', () => {
    // `a` arrive à 0:04, `b` part à 0:04 : un absent avant, un absent après, sans discontinuité.
    const entries = [
      ligne({ replayMs: 4_000, xuid: 'a' }),
      ligne({ replayMs: 4_000, kind: 'left', xuid: 'b' }),
    ]
    expect(teammatesAbsence(entries, QUATRE, FRAME_MS, SCALE)).toMatchObject([
      { from: 0, to: 1, absent: 1, total: 4 },
    ])
  })

  it('le dénominateur est l’effectif de référence, pas le nombre d’absents', () => {
    const entries = [ligne({ replayMs: 5_000, xuid: 'a' })]
    expect(teammatesAbsence(entries, ['a', 'b'], FRAME_MS, SCALE)[0]).toMatchObject({
      absent: 1,
      total: 2,
    })
  })

  it('un joueur hors de l’effectif regardé n’ombre pas la piste', () => {
    const entries = [ligne({ replayMs: 5_000, xuid: 'inconnu' })]
    expect(teammatesAbsence(entries, QUATRE, FRAME_MS, SCALE)).toEqual([])
  })

  it('sans coéquipier, aucun palier — il n’y a pas d’effectif à comparer', () => {
    const entries = [ligne({ replayMs: 5_000, xuid: 'a' })]
    expect(teammatesAbsence(entries, [], FRAME_MS, SCALE)).toEqual([])
  })

  it('effectif complet de bout en bout : aucun palier', () => {
    expect(teammatesAbsence([autreLigne(3_000)], QUATRE, FRAME_MS, SCALE)).toEqual([])
  })
})

/**
 * AJOUT DU 2026-09-07 (revue F3) — LE BOT SÉLECTIONNÉ A SON OMBRE.
 *
 * Le cas RÉEL du match témoin `4ecdf3e7` : le remplaçant est un BOT. Le menu de point de vue
 * rend son xuid de BASE (`bid(N.0)`, cf. `viewpointOptions` — le piège des bots y est traité),
 * alors que la ligne de présence portait la clé du FILM (`bot:<nom>`). Les deux ne se
 * rencontraient jamais : la piste du bot sélectionné restait vierge d'ombre et de porte pendant
 * que le fil affichait « a rejoint ». La correction est dans `presenceFeed` ; ce test-ci vérifie
 * que l'appariement ABOUTIT, c'est-à-dire que le défaut ne peut pas revenir par ce chemin.
 */
describe('presenceShades — appariement d’un BOT sur le xuid de base', () => {
  it('un bot joint, sélectionné par son xuid de BASE : son ombre et sa porte existent', () => {
    const entries = [ligne({ replayMs: 2_500, xuid: 'bid(1.0)', name: '343 Oscar', bot: true })]
    const [ombre] = presenceShades(entries, 'bid(1.0)', FRAME_MS, SCALE, clockOf)
    expect(ombre).toMatchObject({ from: 0, to: 0.25, edge: 0.25, xuid: 'bid(1.0)' })
  })

  it('la clé du film ne s’apparie plus à rien, et c’est exactement le défaut corrigé', () => {
    const entries = [ligne({ replayMs: 2_500, xuid: 'bid(1.0)', bot: true })]
    expect(presenceShades(entries, 'bot:343 Oscar', FRAME_MS, SCALE, clockOf)).toEqual([])
  })

  it('un bot COÉQUIPIER compte dans les paliers d’absence de la piste Coéquipiers', () => {
    const entries = [ligne({ replayMs: 5_000, xuid: 'bid(1.0)', bot: true })]
    const paliers = teammatesAbsence(entries, ['bid(1.0)', 'pote'], FRAME_MS, SCALE)
    expect(paliers).toEqual([{ key: 'a0-0.5', from: 0, to: 0.5, absent: 1, total: 2 }])
  })
})
