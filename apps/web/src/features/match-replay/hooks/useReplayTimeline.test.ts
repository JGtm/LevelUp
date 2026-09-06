/**
 * Tests — reduceFeed (à qui appartient une ligne du fil, vu des pistes de la frise).
 *
 * POURQUOI CETTE FONCTION MÉRITE SES PROPRES CAS (revue R1) : c'est le seul endroit qui décide
 * de l'ACTEUR d'une ligne, et une erreur y est invisible partout ailleurs. Inverser tueur et
 * victime laisserait les deux pistes peuplées — avec les mauvais événements ; retirer la garde
 * « la victime est le joueur regardé » remplirait sa piste des morts de toute la partie. Ni la
 * relecture ni l'écran ne rattrapent ça : seul un test le tient.
 *
 * CE QUI EST TESTÉ ICI EST L'ATTRIBUTION, PAS LE PLACEMENT. Le tri par piste (point de vue,
 * coéquipiers) et le calcul de position vivent dans `buildEventTracks`
 * (replayTimelineTracksLogic.test.ts) ; ici on vérifie seulement que chaque ligne part avec le
 * bon xuid et la bonne clé.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDeath, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import { reduceFeed } from './useReplayTimeline'

/**
 * LE POINT DE VUE, ET RIEN D'AUTRE (2026-09-07, lot L3). Cette fonction lisait la marque `me`
 * de `playerMarks` pour reconnaître « ma » mort ; elle lit maintenant le point de vue
 * directement. C'était déjà le même joueur — la marque `me` suit le point de vue depuis L2b —
 * mais par un détour qui laissait croire que la frise dépendait des marques d'identité.
 */
const VIEWPOINT = 'me'

/** Une ligne de kill du fil, réduite à ce que `reduceFeed` lit. */
function killEntry(over: Partial<ReplayKill> & { key?: string; replayMs?: number } = {}): ReplayFeedEntry {
  const { key = 'k1', replayMs = 20_000, ...kill } = over
  return {
    key,
    replayMs,
    kill: { xuid: 'me', victimXuid: '', ...kill } as ReplayKill,
    medal: null,
    death: null,
  }
}

/** Une ligne de mort NEUTRE (suicide, chute, sortie) : personne n'est crédité. */
function deathEntry(xuid: string, key = 'd1', replayMs = 25_000): ReplayFeedEntry {
  return { key, replayMs, kill: null, medal: null, death: { xuid } as ReplayDeath }
}

/** Une médaille seule : ni tueur ni défunt. */
function medalEntry(): ReplayFeedEntry {
  return { key: 'm1', replayMs: 21_000, kill: null, medal: { xuid: 'me' } as never, death: null }
}

describe('reduceFeed — une élimination appartient à son TUEUR', () => {
  it('range le kill sous le xuid du tueur, pas sous celui de la victime', () => {
    const { kills, deaths } = reduceFeed(
      [killEntry({ xuid: 'pote', victimXuid: 'ennemi' })],
      VIEWPOINT,
    )
    expect(kills).toEqual([{ key: 'k1', replayMs: 20_000, xuid: 'pote' }])
    expect(deaths).toEqual([])
  })

  it('garde la clé et l’instant de la ligne du fil — la frise pointe la ligne qu’on lit', () => {
    const { kills } = reduceFeed([killEntry({ key: 'k-42', replayMs: 33_000 })], VIEWPOINT)
    expect(kills[0].key).toBe('k-42')
    expect(kills[0].replayMs).toBe(33_000)
  })

  it('un acteur INCONNU passe quand même — c’est `buildEventTracks` qui filtre', () => {
    // La séparation compte : reduceFeed dit QUI, buildEventTracks dit SUR QUELLE PISTE. Les
    // confondre ferait deux endroits où oublier une règle.
    const { kills } = reduceFeed([killEntry({ xuid: 'inconnu' })], VIEWPOINT)
    expect(kills.map((k) => k.xuid)).toEqual(['inconnu'])
  })
})

describe('reduceFeed — une mort, et ses deux formes', () => {
  it('une mort NEUTRE part avec le xuid du défunt', () => {
    const { kills, deaths } = reduceFeed([deathEntry('me')], VIEWPOINT)
    expect(deaths).toEqual([{ key: 'd1', replayMs: 25_000, xuid: 'me' }])
    expect(kills).toEqual([])
  })

  it('LA MORT NEUTRE D’UN AUTRE passe aussi : le filtrage n’a pas lieu ici', () => {
    const { deaths } = reduceFeed([deathEntry('pote')], VIEWPOINT)
    expect(deaths.map((d) => d.xuid)).toEqual(['pote'])
  })

  // LA GARDE QUI COMPTE : une élimination dont JE suis la victime est MA mort, et c'est la forme
  // que prend la majorité des morts d'un match. Sans elle, la piste « Toi » n'en montrerait
  // presque aucune ; élargie à tout le monde, elle déverserait les morts de la partie entière.
  it('un kill dont la victime est « moi » produit AUSSI une mort, sous une clé distincte', () => {
    const { kills, deaths } = reduceFeed(
      [killEntry({ key: 'k7', xuid: 'ennemi', victimXuid: 'me' })],
      VIEWPOINT,
    )
    expect(kills).toEqual([{ key: 'k7', replayMs: 20_000, xuid: 'ennemi' }])
    expect(deaths).toEqual([{ key: 'k7-v', replayMs: 20_000, xuid: 'me' }])
  })

  it('un kill dont la victime est un COÉQUIPIER ne produit aucune mort', () => {
    const { deaths } = reduceFeed([killEntry({ xuid: 'ennemi', victimXuid: 'pote' })], VIEWPOINT)
    expect(deaths).toEqual([])
  })

  it('un kill dont la victime n’est PAS le point de vue ne produit aucune mort', () => {
    const { deaths } = reduceFeed([killEntry({ xuid: 'pote', victimXuid: 'ennemi' })], VIEWPOINT)
    expect(deaths).toEqual([])
  })

  it('TIR AMI : la même ligne peut être le frag de l’un et la mort de l’autre', () => {
    const { kills, deaths } = reduceFeed(
      [killEntry({ key: 'ff', xuid: 'pote', victimXuid: 'me' })],
      VIEWPOINT,
    )
    expect(kills.map((k) => k.xuid)).toEqual(['pote'])
    expect(deaths.map((d) => d.xuid)).toEqual(['me'])
  })
})

describe('reduceFeed — les FRAGS de la piste Dominance', () => {
  it('compte TOUTE la salle, pas seulement le joueur regardé : la dominance oppose deux camps', () => {
    const { frags } = reduceFeed(
      [
        killEntry({ key: 'a', replayMs: 10_000, xuid: 'me', teamID: 0 }),
        killEntry({ key: 'b', replayMs: 20_000, xuid: 'inconnu', teamID: 1 }),
      ],
      VIEWPOINT,
    )
    expect(frags).toEqual([
      { replayMs: 10_000, teamId: 0 },
      { replayMs: 20_000, teamId: 1 },
    ])
  })

  it('un tueur SANS camp résolu ne compte pour personne', () => {
    // L'attribuer par défaut couronnerait le seul camp qui a un identifiant — le meneur
    // affiché serait alors une conséquence du trou de données, pas du match.
    const { frags } = reduceFeed([killEntry({ xuid: 'me', teamID: null })], VIEWPOINT)
    expect(frags).toEqual([])
  })

  it('une MORT NEUTRE n’est le frag de personne', () => {
    expect(reduceFeed([deathEntry('me')], VIEWPOINT).frags).toEqual([])
  })
})

describe('reduceFeed — ce qui n’est ni un frag ni une mort', () => {
  it('une MÉDAILLE SEULE n’entre sur aucune piste', () => {
    const { kills, deaths } = reduceFeed([medalEntry()], VIEWPOINT)
    expect(kills).toEqual([])
    expect(deaths).toEqual([])
  })

  it('un fil vide rend trois listes vides, jamais undefined', () => {
    expect(reduceFeed([], VIEWPOINT)).toEqual({ kills: [], deaths: [], frags: [] })
  })

  it('l’ordre du fil est conservé — les pistes se lisent dans le sens du match', () => {
    const { kills } = reduceFeed(
      [
        killEntry({ key: 'a', replayMs: 10_000 }),
        killEntry({ key: 'b', replayMs: 20_000 }),
        killEntry({ key: 'c', replayMs: 30_000 }),
      ],
      VIEWPOINT,
    )
    expect(kills.map((k) => k.key)).toEqual(['a', 'b', 'c'])
  })
})
