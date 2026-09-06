/**
 * xuidMeta — CARACTÉRISATION DU 2026-09-06, écrite AVANT le point de vue du rejeu.
 *
 * POURQUOI CE FICHIER, ET POURQUOI MAINTENANT. `resolveXuidMeta` décide QUI EST ALLIÉ sur six
 * surfaces : cinq charts de la page match, plus le rejeu 2D. Le chantier « point de vue de la
 * frise » (plan `.ai/PLAN_FRISE_POINT_DE_VUE_2026-09-06.md`) va lui ajouter un troisième
 * paramètre, et sa décision 15 exige que RIEN NE CHANGE HORS DE LA PAGE DE REJEU. Or jusqu'ici
 * RIEN ne fixait les SORTIES de cette fonction : `xuidMeta.guard.test.ts` interdit bien que la
 * cascade d'appartenance soit recopiée ailleurs, mais un grep ne dit pas ce que la fonction
 * rend. Une inversion d'allié y passerait sans un test rouge, et se verrait à l'écran, tard,
 * sous la forme d'un kill peint de la couleur adverse.
 *
 * CE QUE CES TESTS SONT — ET NE SONT PAS. Ils décrivent le comportement À DEUX ARGUMENTS tel
 * qu'il est AUJOURD'HUI, y compris son repli le plus discutable (le court-circuit `is_me`,
 * plus bas). Ce n'est pas le comportement souhaité, c'est le comportement CONSTATÉ : une
 * photographie. Un test qui rougirait ici est un changement de comportement, pas un test
 * périmé.
 *
 * CONTRAT POUR LE LOT SUIVANT (le point de vue) : il n'a le droit que d'AJOUTER des cas à ce
 * fichier — aucune ligne supprimée, aucune modifiée. Un appel à deux arguments doit rendre
 * après le chantier exactement ce qu'il rend ici.
 *
 * Ce qui est fixé : la Map ENTIÈRE (gamertag ET ally pour chaque xuid), jamais un champ isolé
 * — une entrée qui disparaîtrait, ou un allié qui s'ajouterait, doivent faire rougir.
 */
import { describe, expect, it } from 'vitest'

import { meXUIDOf, resolveXuidMeta, type XuidMeta } from './xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

/**
 * Le lobby témoin : deux camps, une ligne « moi » dans `t0`, un bot (clé `bid(N.0)` et suffixe
 * de données ` [bot]` posé par killsource), et une ligne SANS camp transmis — les trois formes
 * que le rejeu croise réellement. Champs réduits à ce que la fonction regarde, patron des
 * fixtures du dossier (`MatchPadControlSection.test.tsx`).
 */
const BOARD = [
  { xuid: 'me-1', gamertag: 'Alpha', team_side: 't0', is_me: true },
  { xuid: 'ally-2', gamertag: 'Bravo', team_side: 't0', is_me: false },
  { xuid: 'foe-1', gamertag: 'Charlie', team_side: 't1', is_me: false },
  { xuid: 'foe-2', gamertag: 'Delta', team_side: 't1', is_me: false },
  { xuid: 'bid(1.0)', gamertag: '343 Guilty Spark [bot]', team_side: 't1', is_me: false, is_bot: true },
  { xuid: 'nomad-9', gamertag: 'Echo', team_side: null, is_me: false },
] as unknown as MatchScoreboardRow[]

/** Le même lobby SANS aucune ligne « moi » : le pont vers le camp de référence manque. */
const BOARD_SANS_MOI = BOARD.map((r) => ({ ...r, is_me: false }))

/**
 * Les noms d'affichage attendus, une fois pour toutes : ils ne dépendent pas du point de vue.
 * Le bot y perd son suffixe ` [bot]` — c'est `displayPlayerName` qui le retire, et cette
 * caractérisation le fixe aussi.
 */
const NOMS: Record<string, string> = {
  'me-1': 'Alpha',
  'ally-2': 'Bravo',
  'foe-1': 'Charlie',
  'foe-2': 'Delta',
  'bid(1.0)': '343 Guilty Spark',
  'nomad-9': 'Echo',
}

/** La Map rendue, sous une forme comparable par égalité profonde sans dépendre de son ordre. */
function carte(meta: XuidMeta): Record<string, { gamertag: string; ally: boolean }> {
  return Object.fromEntries(meta)
}

/** La Map ATTENDUE : tout le lobby, alliés = ceux qu'on nomme, tous les autres non. */
function attendu(...allies: readonly string[]): Record<string, { gamertag: string; ally: boolean }> {
  return Object.fromEntries(
    Object.entries(NOMS).map(([xuid, gamertag]) => [xuid, { gamertag, ally: allies.includes(xuid) }]),
  )
}

describe('resolveXuidMeta — la table complète, telle qu’elle est rendue le 2026-09-06', () => {
  it('(a) vu depuis la ligne « moi » : son camp est le camp allié, le reste ne l’est pas', () => {
    const meta = resolveXuidMeta(BOARD, 'me-1')
    expect(carte(meta)).toEqual(attendu('me-1', 'ally-2'))
    expect(meta.size).toBe(6)
  })

  it('(b) meXUID = null avec une ligne « moi » présente : SEULE cette ligne reste alliée', () => {
    // C'EST LE CAS DU COURT-CIRCUIT `r.is_me ||`, et c'est lui que cette caractérisation
    // protège. Sans meXUID il n'y a AUCUN camp de référence (`allyTeam` vaut null, donc la
    // comparaison de camp est morte) : le repli `is_me` sauve la seule ligne « moi », et son
    // coéquipier `ally-2` — pourtant du même `t0` — ressort NON allié. Comportement constaté,
    // pas comportement voulu : c'est exactement ce que le lot suivant doit laisser intact,
    // puisque cinq charts de la page match passent par ce chemin quand `meRow` est absent.
    const meta = resolveXuidMeta(BOARD, null)
    expect(carte(meta)).toEqual(attendu('me-1'))
    expect(meta.size).toBe(6)
  })

  it('(c) vu depuis un adversaire : le camp adverse devient allié, ET la ligne « moi » LE RESTE', () => {
    // Deuxième visage du même court-circuit : demandée depuis `foe-1` (camp `t1`), la table
    // rend `t1` alliée — mais la ligne marquée `is_me`, dans `t0`, ressort alliée elle aussi.
    // Deux camps alliés en même temps, donc, et c'est le défaut que le point de vue du rejeu
    // devra corriger CHEZ LUI sans changer ce que rend l'appel à deux arguments.
    const meta = resolveXuidMeta(BOARD, 'foe-1')
    expect(carte(meta)).toEqual(attendu('me-1', 'foe-1', 'foe-2', 'bid(1.0)'))
    expect(meta.size).toBe(6)
  })

  it('(d) aucune ligne « moi » : le camp du xuid demandé fait seul l’alliance', () => {
    const meta = resolveXuidMeta(BOARD_SANS_MOI, 'ally-2')
    expect(carte(meta)).toEqual(attendu('me-1', 'ally-2'))
    expect(meta.size).toBe(6)
  })

  it('(e) meXUID absent du tableau de score : personne n’est allié, sauf la ligne « moi »', () => {
    const meta = resolveXuidMeta(BOARD, 'xuid-jamais-vu')
    expect(carte(meta)).toEqual(attendu('me-1'))
    expect(meta.size).toBe(6)
  })

  it('(e bis) meXUID absent ET aucune ligne « moi » : plus personne n’est allié', () => {
    expect(carte(resolveXuidMeta(BOARD_SANS_MOI, 'xuid-jamais-vu'))).toEqual(attendu())
  })

  it('scoreboard absent : une table vide, jamais une exception', () => {
    expect(carte(resolveXuidMeta(null, null))).toEqual({})
    expect(resolveXuidMeta(null, null).size).toBe(0)
    expect(carte(resolveXuidMeta(undefined, 'me-1'))).toEqual({})
    expect(resolveXuidMeta(undefined, 'me-1').size).toBe(0)
  })
})

describe('meXUIDOf — le joueur de la page vient de la ligne marquée par le backend', () => {
  it('(a) rend le xuid de la ligne « moi »', () => {
    expect(meXUIDOf(BOARD)).toBe('me-1')
  })

  it('(d) rend null quand aucune ligne n’est marquée — aucun joueur deviné', () => {
    expect(meXUIDOf(BOARD_SANS_MOI)).toBeNull()
  })

  it('rend null sur un scoreboard absent', () => {
    expect(meXUIDOf(null)).toBeNull()
    expect(meXUIDOf(undefined)).toBeNull()
  })
})

/**
 * AJOUT DU 2026-09-06 (lot L2b) — le TROISIÈME argument, celui du rejeu 2D.
 *
 * Ces cas ne touchent à aucun de ceux du dessus : ils décrivent un régime qui n'existait pas.
 * Le contrat de la caractérisation tient donc dans les deux sens — à deux arguments, rien n'a
 * bougé ; à trois, le court-circuit `is_me` n'existe plus.
 */
describe('resolveXuidMeta — vu par les yeux d’un point de vue (3e argument)', () => {
  it('vu depuis un adversaire : UN SEUL camp allié, et la ligne « moi » N’EN EST PAS', () => {
    // Le contraire exact du cas (c) à deux arguments, qui rendait `me-1` allié EN PLUS du camp
    // adverse. C'est ce défaut-là que le point de vue corrige, et seulement chez lui.
    const meta = resolveXuidMeta(BOARD, 'me-1', 'foe-1')
    expect(carte(meta)).toEqual(attendu('foe-1', 'foe-2', 'bid(1.0)'))
    expect(meta.get('me-1')?.ally).toBe(false)
    expect(meta.size).toBe(6)
  })

  it('vu depuis le joueur de la page : identique à l’appel à deux arguments', () => {
    expect(carte(resolveXuidMeta(BOARD, 'me-1', 'me-1'))).toEqual(carte(resolveXuidMeta(BOARD, 'me-1')))
  })

  it('vu depuis un coéquipier : le même camp allié, la ligne « moi » comprise', () => {
    expect(carte(resolveXuidMeta(BOARD, 'me-1', 'ally-2'))).toEqual(attendu('me-1', 'ally-2'))
  })

  it('vu depuis un BOT rangé dans un camp : ce camp est l’allié (décision 7 du plan)', () => {
    expect(carte(resolveXuidMeta(BOARD, 'me-1', 'bid(1.0)'))).toEqual(
      attendu('foe-1', 'foe-2', 'bid(1.0)'),
    )
  })

  it('vu depuis une ligne SANS camp transmis : personne n’est allié — aucun camp deviné', () => {
    expect(carte(resolveXuidMeta(BOARD, 'me-1', 'nomad-9'))).toEqual(attendu())
  })

  it('point de vue ABSENT du tableau de score : personne n’est allié, « moi » compris', () => {
    // Le court-circuit est mort dès qu'un point de vue est passé : même introuvable, il ne
    // laisse pas la ligne « moi » se rattraper toute seule.
    expect(carte(resolveXuidMeta(BOARD, 'me-1', 'xuid-jamais-vu'))).toEqual(attendu())
  })

  it('le point de vue PRIME sur meXUID quand les deux sont donnés', () => {
    expect(carte(resolveXuidMeta(BOARD, 'ally-2', 'foe-2'))).toEqual(
      attendu('foe-1', 'foe-2', 'bid(1.0)'),
    )
  })

  it('point de vue à `null` : la fonction retombe sur son régime à deux arguments', () => {
    // `null` veut dire « pas de point de vue distinct », pas « point de vue introuvable » :
    // c'est ce que passe le modèle quand le tableau de score ne nomme personne.
    expect(carte(resolveXuidMeta(BOARD, 'me-1', null))).toEqual(carte(resolveXuidMeta(BOARD, 'me-1')))
  })

  it('les gamertags ne dépendent JAMAIS du point de vue', () => {
    const noms = (m: XuidMeta) => Object.fromEntries([...m].map(([k, v]) => [k, v.gamertag]))
    expect(noms(resolveXuidMeta(BOARD, 'me-1', 'foe-1'))).toEqual(noms(resolveXuidMeta(BOARD, 'me-1')))
  })
})
