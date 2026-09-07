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
import { reduceFeed, useReplayTimeline } from './useReplayTimeline'
import { renderHook } from '@testing-library/react'
import { testReplayDoc } from '../test/testDoc'
import type { ReplayWindowBounds } from '../model/replayWindow'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

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
    kill: { xuid: 'me', victimXuid: '', medals: [], ...kill } as ReplayKill,
    medal: null,
    death: null,
  }
}

/** Une ligne de mort NEUTRE (suicide, chute, sortie) : personne n'est crédité. */
function deathEntry(xuid: string, key = 'd1', replayMs = 25_000): ReplayFeedEntry {
  return { key, replayMs, kill: null, medal: null, death: { xuid } as ReplayDeath }
}

/** Une médaille seule : ni tueur ni défunt — une médaille d'OBJECTIF, presque toujours. */
function medalEntry(label = 'Capture'): ReplayFeedEntry {
  const medal = { xuid: 'me', label } as never
  return { key: 'm1', replayMs: 21_000, kill: null, medal, death: null }
}

describe('reduceFeed — une élimination appartient à son TUEUR', () => {
  it('range le kill sous le xuid du tueur, pas sous celui de la victime', () => {
    const { kills, deaths } = reduceFeed(
      [killEntry({ xuid: 'pote', victimXuid: 'ennemi' })],
      VIEWPOINT,
    )
    expect(kills).toEqual([{ key: 'k1', replayMs: 20_000, xuid: 'pote', medals: [] }])
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
    expect(kills).toEqual([{ key: 'k7', replayMs: 20_000, xuid: 'ennemi', medals: [] }])
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

/**
 * LES MÉDAILLES (2026-09-07, lot L4). Deux chemins qui ne se confondent pas : rattachée à un
 * kill, la médaille voyage AVEC lui en libellés (la marque existe déjà, elle recevra un anneau) ;
 * orpheline, elle prend une entrée à elle. Sans libellé, rien — sur les matchs d'avant le
 * backfill, une décoration muette ne dirait pas ce qui a été décroché.
 */
describe('reduceFeed — les médailles', () => {
  it('les médailles d’un kill le suivent en LIBELLÉS, pas en repère de plus', () => {
    const decore = [{ label: 'Doublé' }, { label: 'Vengeance' }] as never
    const { kills, medals } = reduceFeed([killEntry({ medals: decore })], VIEWPOINT)
    expect(kills[0].medals).toEqual(['Doublé', 'Vengeance'])
    expect(medals).toEqual([])
  })

  it('un libellé VIDE ne décore rien : la marque reste nue plutôt que muette', () => {
    const sansNom = [{ label: '' }, { label: 'Doublé' }] as never
    const { kills } = reduceFeed([killEntry({ medals: sansNom })], VIEWPOINT)
    expect(kills[0].medals).toEqual(['Doublé'])
  })

  it('une MÉDAILLE SEULE n’est ni un kill ni une mort : elle sort par sa propre liste', () => {
    const { kills, deaths, medals } = reduceFeed([medalEntry()], VIEWPOINT)
    expect(kills).toEqual([])
    expect(deaths).toEqual([])
    expect(medals).toEqual([{ key: 'm1', replayMs: 21_000, xuid: 'me', label: 'Capture' }])
  })

  it('une médaille seule SANS LIBELLÉ ne sort pas : rien à nommer, rien à dessiner', () => {
    expect(reduceFeed([medalEntry('')], VIEWPOINT).medals).toEqual([])
  })

  it('une médaille seule n’est le frag de personne', () => {
    expect(reduceFeed([medalEntry()], VIEWPOINT).frags).toEqual([])
  })
})

describe('reduceFeed — ce qui n’est ni un frag ni une mort', () => {
  it('un fil vide rend quatre listes vides, jamais undefined', () => {
    expect(reduceFeed([], VIEWPOINT)).toEqual({ kills: [], deaths: [], medals: [], frags: [] })
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

/**
 * AJOUT DU 2026-09-07 (revue F5) — LE BRANCHEMENT DU HOOK, pas seulement sa réduction.
 *
 * POURQUOI CES CAS. `reduceFeed` était testé ; `useReplayTimeline` lui-même ne l'était par
 * RIEN — aucun test ne le montait. Or c'est lui qui décide À QUOI le point de vue est branché :
 * quel joueur ombre sa piste (`presenceShades`), quel effectif sert de dénominateur à la piste
 * Coéquipiers (`teammatesAbsence`), et ce que le menu propose. Un relais coupé — `null` au lieu
 * du point de vue, le sujet compté parmi ses propres coéquipiers — laissait la suite verte et
 * l'écran faux : une piste sans ombre, ou une piste d'équipe assombrie par l'arrivée de celui
 * qu'on regarde.
 */
describe('useReplayTimeline — ce à quoi le point de vue est branché', () => {
  /** Document 10 Hz, 200 images : la frise couvre les images 0 à 199, un ratio r vaut 199·r. */
  const DOC = testReplayDoc({ frameCount: 200, frameIntervalMs: 100, originMs: 0 })
  const WINDOW: ReplayWindowBounds = {
    startFrame: 0,
    leadInFrame: 0,
    endFrame: 200,
    startMs: 0,
    endMs: 20_000,
  }

  /** Une ligne d'entrée en partie, du fil déjà fusionné (`mergeFeedWithPresence`). */
  function arrivee(xuid: string, replayMs: number): ReplayFeedEntry {
    return {
      key: `p-joined-${xuid}-${replayMs}`,
      replayMs,
      kill: null,
      medal: null,
      death: null,
      presence: { kind: 'joined', xuid, name: xuid, bot: false, source: 'api' },
    }
  }

  /** Le roster joint : deux camps, plus un joueur que le tableau de score ne connaît pas. */
  const PLAYERS = [
    { xuid: 'moi', filmName: 'Moi', lives: [], board: { xuid: 'moi', team_side: 't0' } },
    { xuid: 'pote', filmName: 'Pote', lives: [], board: { xuid: 'pote', team_side: 't0' } },
    { xuid: 'eux', filmName: 'Eux', lives: [], board: { xuid: 'eux', team_side: 't1' } },
    { xuid: 'bot:Oscar', filmName: 'Oscar [bot]', bot: true, lives: [] },
  ] as unknown as ReplayPlayer[]

  /** L'identité RELATIVE au point de vue, telle que le modèle la rend (`resolveXuidMeta`). */
  function identiteVueDe(sujet: string): ReadonlyMap<string, { ally: boolean }> {
    const camp: Record<string, string> = { moi: 't0', pote: 't0', eux: 't1' }
    const mien = camp[sujet]
    return new Map(
      Object.entries(camp).map(([xuid, side]) => [xuid, { ally: xuid === sujet || side === mien }]),
    )
  }

  const PLAYBACK = {
    sliderRef: { current: null },
    startFrame: 0,
    endFrame: 200,
    onScrub: () => {},
    playing: false,
    togglePlay: () => {},
    restart: () => {},
    seekBy: () => {},
    stepFrames: () => {},
    seekToFrame: () => {},
  }

  function monter(viewpoint: string, feedEntries: readonly ReplayFeedEntry[]) {
    return renderHook(() =>
      useReplayTimeline({
        doc: DOC,
        playWindow: WINDOW,
        feedEntries,
        marks: new Map(),
        viewpoint,
        identity: identiteVueDe(viewpoint),
        players: PLAYERS,
        onSelectViewpoint: () => {},
        lead: { allyOf: () => null, labelOf: (id: number) => `Équipe ${id}` },
        playback: PLAYBACK,
        toggleSound: () => {},
        renderWidth: 480,
        locale: 'fr',
      }),
    )
  }

  it('(i) les ombres rendues sont celles du POINT DE VUE, pas celles de personne', () => {
    const feed = [arrivee('moi', 5_000), arrivee('eux', 8_000)]
    expect(monter('moi', feed).result.current.shades.map((s) => s.xuid)).toEqual(['moi'])
    // Le relais suit la bascule : c'est CE branchement-là qu'aucun test ne tenait.
    expect(monter('eux', feed).result.current.shades.map((s) => s.xuid)).toEqual(['eux'])
  })

  it('(i bis) sans ligne de présence pour lui, aucune ombre — le cas nominal', () => {
    expect(monter('pote', [arrivee('moi', 5_000)]).result.current.shades).toEqual([])
  })

  it('(ii) les coéquipiers EXCLUENT le point de vue : il n’assombrit pas sa propre piste', () => {
    // `moi` et `pote` sont du même camp : vu de `moi`, l'effectif de référence est de UN.
    const { result } = monter('moi', [arrivee('moi', 8_000), arrivee('pote', 4_000)])
    expect(result.current.absence.map((a) => [a.absent, a.total])).toEqual([[1, 1]])
    // Son arrivée à 8 s n'ouvre aucun palier : le seul palier s'arrête à celle de `pote`, à 4 s.
    expect(result.current.absence[0].to).toBeCloseTo(0.2, 6)
  })

  it('(ii bis) vu d’un camp d’un seul joueur, la piste Coéquipiers n’a aucun palier', () => {
    const { result } = monter('eux', [arrivee('eux', 4_000), arrivee('moi', 4_000)])
    expect(result.current.absence).toEqual([])
  })

  it('(iii) le menu porte les libellés i18n attendus, groupe « Sans équipe » compris', () => {
    const groupes = monter('moi', []).result.current.viewpointGroups
    expect(groupes.map((g) => g.label)).toEqual(['Équipe 0', 'Équipe 1', 'Sans équipe'])
    expect(groupes[0].options.map((o) => o.label)).toEqual(['Moi', 'Pote'])
    // Un joueur sans ligne de tableau de score est listé mais INERTE (décision 7 bis du plan),
    // et son infobulle donne la raison — jamais une option qui fait semblant.
    const orphelin = groupes[2].options[0]
    expect(orphelin).toMatchObject({
      label: 'Oscar',
      disabled: true,
      title: 'Aucune donnée de match pour ce joueur',
    })
  })

  it('(iii bis) le point de vue et le geste de sélection ressortent tels quels', () => {
    const { result } = monter('eux', [])
    expect(result.current.viewpoint).toBe('eux')
    expect(typeof result.current.onSelectViewpoint).toBe('function')
  })
})
