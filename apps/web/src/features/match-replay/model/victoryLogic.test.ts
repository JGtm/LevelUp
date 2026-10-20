/**
 * Tests — readVictory (comment le match s'est terminé pour le joueur de la page).
 *
 * Ce que ces tests protègent : une lecture fausse ne casse rien à l'exécution, elle HABILLE
 * L'ÉCRAN DE FIN AVEC LA MAUVAISE ÉQUIPE, en plein cadre. Trois erreurs sont plausibles et
 * couvertes ici : confondre « mon équipe » et « l'équipe qui gagne » sur une défaite (elles
 * diffèrent, et c'est le cœur de l'amendement du 2026-08-26), inverser le camp adverse, et
 * afficher l'écran là où il n'a pas de sens (FFA, trois camps, abandon).
 *
 * LES CAMPS, L'ÉQUIPE DU JOUEUR DE LA PAGE ET CELLE DU SUJET VIENNENT DU FILM (2026-10-06) :
 * chaque joueur porte l'équipe que le roster lui écrit, et sa ligne de feuille ne fait que
 * NOMMER son camp (`teamSide`). Les cas les construisent ainsi (`film`).
 */
import { describe, expect, it } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { finalScoreFromHeader, readVictory, victoryIsFlipped } from './victoryLogic'

/** Un joueur du film : sa clé, l'équipe que le film lui écrit, et le côté de sa ligne de feuille. */
type Joueur = [xuid: string, team: number | undefined, side: string | null]

/** Les joueurs du film, joints à leur ligne de feuille (qui ne fait que nommer le camp). */
function film(joueurs: Joueur[]) {
  const players = joueurs.map(
    ([xuid, team, side]) => ({ xuid, team, lives: [], board: { xuid, team_side: side } }) as unknown as ReplayPlayer,
  )
  // La référence de l'allégeance n'est pas lue par la fin de match : page et sujet sont passés.
  return buildFilmAllegiance(players, null)
}

/** Le cas de référence : deux camps, le joueur de la page (`moi`) au camp 0. */
const DEUX_CAMPS = film([
  ['moi', 0, 't0'],
  ['pote', 0, 't0'],
  ['eux-1', 1, 't1'],
  ['eux-2', 1, 't1'],
])

describe('readVictory — l’issue vient du code d’en-tête, pas du score', () => {
  it('victoire (code 2) : mon équipe habille l’écran, et c’est elle qui gagne', () => {
    expect(readVictory(DEUX_CAMPS, 2, 'moi')).toEqual({
      outcome: 'win',
      mine: { teamID: 0, teamSide: 't0', ally: true },
      winner: { teamID: 0, teamSide: 't0', ally: true },
    })
  })

  it('défaite (code 3) : mon équipe habille TOUJOURS l’écran, le vainqueur est l’AUTRE', () => {
    expect(readVictory(DEUX_CAMPS, 3, 'moi')).toEqual({
      outcome: 'loss',
      mine: { teamID: 0, teamSide: 't0', ally: true },
      winner: { teamID: 1, teamSide: 't1', ally: false },
    })
  })

  it('le camp adverse est nommé par SON camp réel, pas par « l’autre numéro »', () => {
    // Camps 2 et 5 : un calcul par complément (1 − index) sur les IDENTIFIANTS donnerait
    // n'importe quoi. La lecture indexe les CAMPS, pas les numéros d'équipe.
    const exotique = film([
      ['moi', 5, 't5'],
      ['eux', 2, 't2'],
    ])
    expect(readVictory(exotique, 3, 'moi')).toEqual({
      outcome: 'loss',
      mine: { teamID: 5, teamSide: 't5', ally: true },
      winner: { teamID: 2, teamSide: 't2', ally: false },
    })
  })

  it('un camp que la feuille ne nomme pas garde son numéro : `teamSide` nul, jamais inventé', () => {
    const muette = film([
      ['moi', 0, 't0'],
      ['bot:Ritzy', 1, null],
    ])
    expect(readVictory(muette, 3, 'moi')?.winner).toEqual({ teamID: 1, teamSide: null, ally: false })
  })

  it('égalité (code 1) : aucune équipe rendue — la neutralité est dans la donnée', () => {
    expect(readVictory(DEUX_CAMPS, 1, 'moi')).toEqual({ outcome: 'tie', mine: null, winner: null })
  })
})

describe('readVictory — les situations où aucun écran ne doit s’afficher', () => {
  it('FFA (le film ne donne aucune équipe, `-1`) : null', () => {
    const ffa = film([
      ['moi', -1, null],
      ['a', -1, null],
      ['b', -1, null],
    ])
    expect(readVictory(ffa, 2, 'moi')).toBeNull()
    // L'égalité en FFA aussi : le panneau annonce la fin d'un duel de camps.
    expect(readVictory(ffa, 1, 'moi')).toBeNull()
  })

  it('trois camps : null — un écran à deux camps ne peut pas dire un match à trois', () => {
    const troisCamps = film([
      ['moi', 0, 't0'],
      ['eux', 1, 't1'],
      ['autres', 2, 't2'],
    ])
    expect(readVictory(troisCamps, 2, 'moi')).toBeNull()
  })

  it('un seul camp : null', () => {
    expect(readVictory(film([['moi', 0, 't0'], ['pote', 0, 't0']]), 2, 'moi')).toBeNull()
  })

  it('code d’issue absent : null (rien à annoncer, on n’invente pas de résultat)', () => {
    expect(readVictory(DEUX_CAMPS, undefined, 'moi')).toBeNull()
    expect(readVictory(DEUX_CAMPS, null, 'moi')).toBeNull()
  })

  it('abandon (code 4) : null — un match quitté ne se conclut pas', () => {
    expect(readVictory(DEUX_CAMPS, 4, 'moi')).toBeNull()
  })

  it('code hors contrat (0, 99) : null', () => {
    expect(readVictory(DEUX_CAMPS, 0, 'moi')).toBeNull()
    expect(readVictory(DEUX_CAMPS, 99, 'moi')).toBeNull()
  })

  it('joueur de la page inconnu (nul, ou absent du film) : null — le pont vers mon camp manque', () => {
    expect(readVictory(DEUX_CAMPS, 2, null)).toBeNull()
    expect(readVictory(DEUX_CAMPS, 2, 'hors-film')).toBeNull()
  })

  it('joueur de la page dont le film TAIT l’équipe : null — sa ligne de feuille ne la remplace pas', () => {
    const muet = film([
      ['moi', undefined, 't0'],
      ['pote', 0, 't0'],
      ['eux', 1, 't1'],
    ])
    expect(readVictory(muet, 2, 'moi')).toBeNull()
  })
})

// ─── finalScoreFromHeader : le résultat vient de l'API sur un mode à manches ─

describe('finalScoreFromHeader', () => {
  it("rend le compte de manches quand l'en-tête le dit", () => {
    // Témoin 293a763e : 2 manches à 1, alors que le calque du film rendrait « 100 - 43 ».
    expect(
      finalScoreFromHeader({ score_kind: 'rounds', score_mine: 2, score_theirs: 1 }),
    ).toEqual({ ally: 2, enemy: 1 })
  })

  // Le critère est la PRÉSENCE des deux nombres, jamais leur nature. Une variante à manches
  // dont les camps finissent à égalité (témoin adb93fb7 : 1 partout + 1 nulle) retombe côté
  // serveur sur les points et publie score_kind = "points" ; filtrer sur « rounds » renverrait
  // alors l'écran de fin vers les points de la DERNIÈRE MANCHE, en contradiction avec la vue
  // match qui affiche le total.
  it("rend aussi le score quand il est en points (repli d'une égalité de manches)", () => {
    expect(
      finalScoreFromHeader({ score_kind: 'points', score_mine: 277, score_theirs: 234 }),
    ).toEqual({ ally: 277, enemy: 234 })
  })

  it('rend null quand les nombres manquent (ligne antérieure au backfill)', () => {
    expect(finalScoreFromHeader({ score_kind: 'rounds' })).toBeNull()
    expect(finalScoreFromHeader({ score_kind: 'points', score_mine: 50 })).toBeNull()
    expect(finalScoreFromHeader(undefined)).toBeNull()
  })

  it('accepte un zéro : 2 manches à 0 est une mesure, pas une absence', () => {
    expect(
      finalScoreFromHeader({ score_kind: 'rounds', score_mine: 2, score_theirs: 0 }),
    ).toEqual({ ally: 2, enemy: 0 })
  })
})

/**
 * Le SUJET : par les yeux de qui cette fin se lit (2026-09-06, lot L2b).
 *
 * CE QUE CES CAS PROTÈGENT : `outcome_code` est le verdict DU JOUEUR DE LA PAGE, et il n'en
 * existe pas d'autre. Lu depuis un adversaire sans permutation, l'écran annoncerait « Victoire »
 * aux couleurs du camp qui a PERDU — un écran faux, en plein cadre, et parfaitement silencieux.
 */
describe('readVictory — vu par les yeux d’un point de vue (sujet)', () => {
  /** Le même lobby, nommé ; `nomad-9` est un joueur dont le film tait l'équipe. */
  const NOMME = film([
    ['me-1', 0, 't0'],
    ['ally-2', 0, 't0'],
    ['foe-1', 1, 't1'],
    ['nomad-9', undefined, null],
  ])

  it('sujet = le joueur de la page : rigoureusement la lecture d’origine', () => {
    expect(readVictory(NOMME, 2, 'me-1', 'me-1')).toEqual(readVictory(NOMME, 2, 'me-1'))
    expect(readVictory(NOMME, 3, 'me-1', 'me-1')).toEqual(readVictory(NOMME, 3, 'me-1'))
  })

  it('sujet = un coéquipier : identique aussi — même camp, même verdict', () => {
    expect(readVictory(NOMME, 3, 'me-1', 'ally-2')).toEqual(readVictory(NOMME, 3, 'me-1'))
  })

  it('sujet ADVERSE sur une victoire du joueur de la page : chez lui, c’est une DÉFAITE', () => {
    expect(readVictory(NOMME, 2, 'me-1', 'foe-1')).toEqual({
      outcome: 'loss',
      mine: { teamID: 1, teamSide: 't1', ally: true },
      winner: { teamID: 0, teamSide: 't0', ally: false },
    })
  })

  it('sujet ADVERSE sur une défaite du joueur de la page : c’est une VICTOIRE', () => {
    expect(readVictory(NOMME, 3, 'me-1', 'foe-1')).toEqual({
      outcome: 'win',
      mine: { teamID: 1, teamSide: 't1', ally: true },
      winner: { teamID: 1, teamSide: 't1', ally: true },
    })
  })

  it('égalité : elle l’est pour tout le monde, rien à permuter', () => {
    expect(readVictory(NOMME, 1, 'me-1', 'foe-1')).toEqual({ outcome: 'tie', mine: null, winner: null })
  })

  it('sujet dont le film tait l’équipe, ou absent du film : null — aucun écran plutôt qu’un écran faux', () => {
    expect(readVictory(NOMME, 2, 'me-1', 'nomad-9')).toBeNull()
    expect(readVictory(NOMME, 2, 'me-1', 'xuid-jamais-vu')).toBeNull()
  })

  it('sujet à null : le comportement d’origine, le joueur de la page', () => {
    expect(readVictory(NOMME, 2, 'me-1', null)).toEqual(readVictory(NOMME, 2, 'me-1'))
  })

  it('joueur de la page inconnu : un sujet situable ne suffit pas, le pont d’issue manque', () => {
    // `outcome_code` n'est interprétable que RELATIVEMENT au joueur de la page. Sans son camp,
    // on ne sait pas de quel camp part la permutation — donc pas d'écran, sujet ou pas.
    expect(readVictory(NOMME, 2, null, 'foe-1')).toBeNull()
  })

  it('LE FILM DÉCIDE, PAS LA FEUILLE : un sujet que la feuille range du côté de la page reste du camp du film', () => {
    const contradiction = film([
      ['me-1', 0, 't0'],
      ['foe-1', 1, 't0'], // la feuille le dit du côté t0 ; le film l'écrit au camp 1
      ['autre', 1, 't1'],
    ])
    expect(readVictory(contradiction, 2, 'me-1', 'foe-1')?.outcome).toBe('loss')
  })
})

/**
 * LE SCORE FINAL VU DEPUIS L'AUTRE CAMP (2026-09-07, lot L3).
 *
 * `score_mine` / `score_theirs` sont ancrés sur le JOUEUR DE LA PAGE, comme `outcome_code` —
 * l'API ne publie pas le score vu d'un adversaire. Or l'écran de fin donne à cette lecture la
 * PRIORITÉ sur celle du calque du film, qui, elle, suit le point de vue : vu depuis un
 * adversaire, l'écran annonçait donc l'issue permutée et le score dans l'ordre du joueur de la
 * page — « Défaite, 3 - 1 ».
 */
describe('finalScoreFromHeader — vu par les yeux d’un point de vue (sujet)', () => {
  const NOMME = film([
    ['me-1', 0, 't0'],
    ['ally-2', 0, 't0'],
    ['foe-1', 1, 't1'],
    ['nomad-9', undefined, null],
  ])
  const HEADER = { score_kind: 'rounds', score_mine: 3, score_theirs: 1 }

  it('sujet = le joueur de la page : rigoureusement l’ordre d’origine', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1', 'me-1')).toEqual({ ally: 3, enemy: 1 })
  })

  it('sujet = un coéquipier : identique aussi — même camp, même ordre', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1', 'ally-2')).toEqual({ ally: 3, enemy: 1 })
  })

  it('SUJET DE L’AUTRE CAMP : le score est PERMUTÉ, comme l’issue', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1', 'foe-1')).toEqual({ ally: 1, enemy: 3 })
  })

  /**
   * SUJET NON SITUABLE : l'ordre du joueur de la page, PAS `null`. Sans camp il n'y a rien à
   * permuter, et l'ordre de l'API est le seul sens que ces deux nombres aient. Aucun score faux
   * ne peut atteindre l'écran par là : les deux surfaces qui l'affichent ne se rendent qu'avec
   * une lecture de `readVictory`, qui rend `null` dans exactement ces cas-là.
   */
  it('sujet dont le film tait l’équipe, ou absent du film : l’ordre du joueur de la page', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1', 'nomad-9')).toEqual({ ally: 3, enemy: 1 })
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1', 'xuid-jamais-vu')).toEqual({ ally: 3, enemy: 1 })
  })

  it('joueur de la page inconnu, rien à permuter : l’ordre publié par l’API', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, null, 'foe-1')).toEqual({ ally: 3, enemy: 1 })
  })

  it('un match qui n’oppose pas exactement deux camps ne se permute pas non plus', () => {
    const troisCamps = film([
      ['me-1', 0, 't0'],
      ['foe-1', 1, 't1'],
      ['third-1', 2, 't2'],
    ])
    expect(finalScoreFromHeader(HEADER, troisCamps, 'me-1', 'foe-1')).toEqual({ ally: 3, enemy: 1 })
  })

  it('sujet à null, ou film absent : le comportement à un argument, à la ligne près', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1', null)).toEqual(finalScoreFromHeader(HEADER))
    expect(finalScoreFromHeader(HEADER, undefined, 'me-1', 'foe-1')).toEqual(finalScoreFromHeader(HEADER))
  })

  it('les nombres manquants restent null, sujet ou pas', () => {
    expect(finalScoreFromHeader({ score_kind: 'rounds' }, NOMME, 'me-1', 'foe-1')).toBeNull()
  })
})

/**
 * `victoryIsFlipped`, LE PRÉDICAT DU SCORE SERVI PAR L'EN-TÊTE (2026-09-07).
 *
 * Il ne dit pas l'issue, il dit si la lecture a été RETOURNÉE. `finalScoreFromHeader` s'en sert
 * pour échanger les deux nombres de l'en-tête, qui valent pour le joueur de la page.
 */
describe('victoryIsFlipped — le sujet est-il du camp opposé au joueur de la page ?', () => {
  const NOMME = film([
    ['me-1', 0, 't0'],
    ['ally-2', 0, 't0'],
    ['foe-1', 1, 't1'],
    ['nomad-9', undefined, null],
  ])

  it('sujet de l’autre camp : vrai', () => {
    expect(victoryIsFlipped(NOMME, 'me-1', 'foe-1')).toBe(true)
  })

  it('sujet = le joueur de la page, ou un coéquipier : faux', () => {
    expect(victoryIsFlipped(NOMME, 'me-1', 'me-1')).toBe(false)
    expect(victoryIsFlipped(NOMME, 'me-1', 'ally-2')).toBe(false)
  })

  it('sans sujet : faux — c’est le régime d’origine, celui du joueur de la page', () => {
    expect(victoryIsFlipped(NOMME, 'me-1', null)).toBe(false)
    expect(victoryIsFlipped(NOMME, 'me-1')).toBe(false)
  })

  it('sujet non situable : faux — rien à permuter, donc rien de permuté', () => {
    expect(victoryIsFlipped(NOMME, 'me-1', 'nomad-9')).toBe(false)
    expect(victoryIsFlipped(NOMME, 'me-1', 'xuid-jamais-vu')).toBe(false)
  })

  it('joueur de la page inconnu, ou hors d’un match à deux camps : faux', () => {
    expect(victoryIsFlipped(NOMME, null, 'foe-1')).toBe(false)
    expect(victoryIsFlipped(film([['me-1', 0, 't0'], ['foe-1', 1, 't1'], ['t3', 2, 't2']]), 'me-1', 'foe-1')).toBe(false)
    expect(victoryIsFlipped(film([]), 'me-1', 'foe-1')).toBe(false)
  })

  it('IL EST LE MÊME PRÉDICAT QUE LA PERMUTATION DU SCORE, et c’est ce qui les tient ensemble', () => {
    const header = { score_mine: 3, score_theirs: 1 }
    for (const sujet of ['me-1', 'ally-2', 'foe-1', 'nomad-9', 'xuid-jamais-vu', null]) {
      const permute = victoryIsFlipped(NOMME, 'me-1', sujet)
      const attendu = permute ? { ally: 1, enemy: 3 } : { ally: 3, enemy: 1 }
      expect(finalScoreFromHeader(header, NOMME, 'me-1', sujet)).toEqual(attendu)
    }
  })
})
