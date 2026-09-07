/**
 * Tests — readVictory (comment le match s'est terminé pour le joueur de la page).
 *
 * Ce que ces tests protègent : une lecture fausse ne casse rien à l'exécution, elle HABILLE
 * L'ÉCRAN DE FIN AVEC LA MAUVAISE ÉQUIPE, en plein cadre. Trois erreurs sont plausibles et
 * couvertes ici : confondre « mon équipe » et « l'équipe qui gagne » sur une défaite (elles
 * diffèrent, et c'est le cœur de l'amendement du 2026-08-26), inverser le camp adverse, et
 * afficher l'écran là où il n'a pas de sens (FFA, trois camps, abandon).
 */
import { describe, expect, it } from 'vitest'

import { finalScoreFromHeader, readVictory } from './victoryLogic'
// AJOUT DU 2026-09-07 (revue F2) — IMPORT SÉPARÉ, À DESSEIN. La caractérisation L2a de ce
// fichier n'accepte que des lignes AJOUTÉES (gate du plan : le diff n'a aucune ligne `-`).
// Compléter l'import du dessus l'aurait MODIFIÉ. Aucune règle du dépôt n'interdit deux imports
// du même module (pas de `import/no-duplicates` dans `eslint.config.js`) ; les deux lignes se
// fondront en une le jour où la caractérisation cessera d'être gelée.
import { victoryIsFlipped } from './victoryLogic'

/** Une ligne de scoreboard réduite à ce que la lecture regarde. */
function row(side: string | null, isMe = false) {
  return { team_side: side, is_me: isMe }
}

/** Le cas de référence : deux camps, le joueur de la page dans `t0`. */
const DEUX_CAMPS = [row('t0', true), row('t0'), row('t1'), row('t1')]

describe('readVictory — l’issue vient du code d’en-tête, pas du score', () => {
  it('victoire (code 2) : mon équipe habille l’écran, et c’est elle qui gagne', () => {
    expect(readVictory(DEUX_CAMPS, 2)).toEqual({
      outcome: 'win',
      mine: { teamID: 0, teamSide: 't0', ally: true },
      winner: { teamID: 0, teamSide: 't0', ally: true },
    })
  })

  it('défaite (code 3) : mon équipe habille TOUJOURS l’écran, le vainqueur est l’AUTRE', () => {
    expect(readVictory(DEUX_CAMPS, 3)).toEqual({
      outcome: 'loss',
      mine: { teamID: 0, teamSide: 't0', ally: true },
      winner: { teamID: 1, teamSide: 't1', ally: false },
    })
  })

  it('le camp adverse est nommé par son team_side réel, pas par « l’autre numéro »', () => {
    // Camps t2 et t5 : un calcul par complément (1 − index) sur les IDENTIFIANTS donnerait
    // n'importe quoi. La lecture indexe les CAMPS, pas les numéros d'équipe.
    const exotique = [row('t5', true), row('t2')]
    expect(readVictory(exotique, 3)).toEqual({
      outcome: 'loss',
      mine: { teamID: 5, teamSide: 't5', ally: true },
      winner: { teamID: 2, teamSide: 't2', ally: false },
    })
  })

  it('égalité (code 1) : aucune équipe rendue — la neutralité est dans la donnée', () => {
    expect(readVictory(DEUX_CAMPS, 1)).toEqual({ outcome: 'tie', mine: null, winner: null })
  })
})

describe('readVictory — les situations où aucun écran ne doit s’afficher', () => {
  it('FFA (aucun camp transmis) : null', () => {
    expect(readVictory([row(null, true), row(null), row(null)], 2)).toBeNull()
  })

  it('trois camps : null — un écran à deux camps ne peut pas dire un match à trois', () => {
    const troisCamps = [row('t0', true), row('t1'), row('t2')]
    expect(readVictory(troisCamps, 2)).toBeNull()
  })

  it('un seul camp identifié : null', () => {
    expect(readVictory([row('t0', true), row('t0')], 2)).toBeNull()
  })

  it('code d’issue absent : null (rien à annoncer, on n’invente pas de résultat)', () => {
    expect(readVictory(DEUX_CAMPS, undefined)).toBeNull()
    expect(readVictory(DEUX_CAMPS, null)).toBeNull()
  })

  it('abandon (code 4) : null — un match quitté ne se conclut pas', () => {
    expect(readVictory(DEUX_CAMPS, 4)).toBeNull()
  })

  it('code hors contrat (0, 99) : null', () => {
    expect(readVictory(DEUX_CAMPS, 0)).toBeNull()
    expect(readVictory(DEUX_CAMPS, 99)).toBeNull()
  })

  it('scoreboard sans ligne `is_me` : null — le pont vers mon camp manque', () => {
    expect(readVictory([row('t0'), row('t1')], 2)).toBeNull()
  })

  it('ligne `is_me` sans camp transmis : null', () => {
    const sansCamp = [row(null, true), row('t0'), row('t1')]
    expect(readVictory(sansCamp, 2)).toBeNull()
  })

  it('égalité en FFA : null aussi — le panneau annonce la fin d’un duel de camps', () => {
    expect(readVictory([row(null, true), row(null)], 1)).toBeNull()
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
 * AJOUT DU 2026-09-06 (lot L2b) — le SUJET : par les yeux de qui cette fin se lit.
 *
 * Ajouts seulement : les cas ci-dessus fixent le comportement à deux arguments, qui est celui
 * de la fin de partie SONORE (décision 3 — elle reste ancrée sur le joueur de la page).
 *
 * CE QUE CES CAS PROTÈGENT : `outcome_code` est le verdict DU JOUEUR DE LA PAGE, et il n'en
 * existe pas d'autre. Lu depuis un adversaire sans permutation, l'écran annoncerait « Victoire »
 * aux couleurs du camp qui a PERDU — un écran faux, en plein cadre, et parfaitement silencieux.
 */
describe('readVictory — vu par les yeux d’un point de vue (3e argument)', () => {
  /** Le même lobby que `DEUX_CAMPS`, mais nommé : le sujet se cherche par xuid. */
  const NOMME = [
    { xuid: 'me-1', team_side: 't0', is_me: true },
    { xuid: 'ally-2', team_side: 't0', is_me: false },
    { xuid: 'foe-1', team_side: 't1', is_me: false },
    { xuid: 'nomad-9', team_side: null, is_me: false },
  ]

  it('sujet = le joueur de la page : rigoureusement la lecture d’origine', () => {
    expect(readVictory(NOMME, 2, 'me-1')).toEqual(readVictory(NOMME, 2))
    expect(readVictory(NOMME, 3, 'me-1')).toEqual(readVictory(NOMME, 3))
  })

  it('sujet = un coéquipier : identique aussi — même camp, même verdict', () => {
    expect(readVictory(NOMME, 3, 'ally-2')).toEqual(readVictory(NOMME, 3))
  })

  it('sujet ADVERSE sur une victoire du joueur de la page : chez lui, c’est une DÉFAITE', () => {
    expect(readVictory(NOMME, 2, 'foe-1')).toEqual({
      outcome: 'loss',
      mine: { teamID: 1, teamSide: 't1', ally: true },
      winner: { teamID: 0, teamSide: 't0', ally: false },
    })
  })

  it('sujet ADVERSE sur une défaite du joueur de la page : c’est une VICTOIRE', () => {
    expect(readVictory(NOMME, 3, 'foe-1')).toEqual({
      outcome: 'win',
      mine: { teamID: 1, teamSide: 't1', ally: true },
      winner: { teamID: 1, teamSide: 't1', ally: true },
    })
  })

  it('égalité : elle l’est pour tout le monde, rien à permuter', () => {
    expect(readVictory(NOMME, 1, 'foe-1')).toEqual({ outcome: 'tie', mine: null, winner: null })
  })

  it('sujet sans camp transmis : null — aucun écran plutôt qu’un écran faux', () => {
    expect(readVictory(NOMME, 2, 'nomad-9')).toBeNull()
  })

  it('sujet absent du tableau de score : null', () => {
    expect(readVictory(NOMME, 2, 'xuid-jamais-vu')).toBeNull()
  })

  it('sujet à null : le comportement d’origine, le joueur de la page', () => {
    expect(readVictory(NOMME, 2, null)).toEqual(readVictory(NOMME, 2))
  })

  it('sans ligne « moi », un sujet situable ne suffit pas : le pont d’issue manque', () => {
    // `outcome_code` n'est interprétable que RELATIVEMENT au joueur de la page. Sans sa ligne,
    // on ne sait pas de quel camp part la permutation — donc pas d'écran, sujet ou pas.
    const sansMoi = NOMME.map((r) => ({ ...r, is_me: false }))
    expect(readVictory(sansMoi, 2, 'foe-1')).toBeNull()
  })
})

/**
 * AJOUT DU 2026-09-07 (lot L3) — LE SCORE FINAL VU DEPUIS L'AUTRE CAMP.
 *
 * Ajouts seulement : les cas à un argument ci-dessus fixent le comportement d'origine, celui de
 * toute surface qui ne connaît pas de point de vue.
 *
 * CE QU'ILS PROTÈGENT. `score_mine` / `score_theirs` sont ancrés sur le JOUEUR DE LA PAGE, comme
 * `outcome_code` — l'API ne publie pas le score vu d'un adversaire. Or l'écran de fin donne à
 * cette lecture la PRIORITÉ sur celle du calque du film, qui, elle, suit le point de vue : vu
 * depuis un adversaire, l'écran annonçait donc l'issue permutée et le score dans l'ordre du
 * joueur de la page — « Défaite, 3 - 1 ». Le défaut est arrivé avec L2b et ne se voyait qu'à la
 * fin de la lecture, après avoir changé de joueur.
 */
describe('finalScoreFromHeader — vu par les yeux d’un point de vue (sujet)', () => {
  const NOMME = [
    { xuid: 'me-1', team_side: 't0', is_me: true },
    { xuid: 'ally-2', team_side: 't0', is_me: false },
    { xuid: 'foe-1', team_side: 't1', is_me: false },
    { xuid: 'nomad-9', team_side: null, is_me: false },
  ]
  const HEADER = { score_kind: 'rounds', score_mine: 3, score_theirs: 1 }

  it('sujet = le joueur de la page : rigoureusement l’ordre d’origine', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'me-1')).toEqual({ ally: 3, enemy: 1 })
  })

  it('sujet = un coéquipier : identique aussi — même camp, même ordre', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'ally-2')).toEqual({ ally: 3, enemy: 1 })
  })

  it('SUJET DE L’AUTRE CAMP : le score est PERMUTÉ, comme l’issue', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'foe-1')).toEqual({ ally: 1, enemy: 3 })
  })

  /**
   * SUJET NON SITUABLE : l'ordre du joueur de la page, PAS `null`. Sans camp il n'y a rien à
   * permuter, et l'ordre de l'API est le seul sens que ces deux nombres aient. Aucun score faux
   * ne peut atteindre l'écran par là : les deux surfaces qui l'affichent ne se rendent qu'avec
   * une lecture de `readVictory`, qui rend `null` dans exactement ces cas-là.
   */
  it('sujet sans camp transmis, ou absent du tableau : l’ordre du joueur de la page', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, 'nomad-9')).toEqual({ ally: 3, enemy: 1 })
    expect(finalScoreFromHeader(HEADER, NOMME, 'xuid-jamais-vu')).toEqual({ ally: 3, enemy: 1 })
  })

  it('sans ligne « moi », rien à permuter : l’ordre publié par l’API', () => {
    const sansMoi = NOMME.map((r) => ({ ...r, is_me: false }))
    expect(finalScoreFromHeader(HEADER, sansMoi, 'foe-1')).toEqual({ ally: 3, enemy: 1 })
  })

  it('un match qui n’oppose pas exactement deux camps ne se permute pas non plus', () => {
    const troisCamps = [...NOMME, { xuid: 'third-1', team_side: 't2', is_me: false }]
    expect(finalScoreFromHeader(HEADER, troisCamps, 'foe-1')).toEqual({ ally: 3, enemy: 1 })
  })

  it('sujet à null, ou tableau absent : le comportement à un argument, à la ligne près', () => {
    expect(finalScoreFromHeader(HEADER, NOMME, null)).toEqual(finalScoreFromHeader(HEADER))
    expect(finalScoreFromHeader(HEADER, undefined, 'foe-1')).toEqual(finalScoreFromHeader(HEADER))
  })

  it('les nombres manquants restent null, sujet ou pas', () => {
    expect(finalScoreFromHeader({ score_kind: 'rounds' }, NOMME, 'foe-1')).toBeNull()
  })
})

/**
 * AJOUT DU 2026-09-07 (revue F2) — `victoryIsFlipped`, LE PRÉDICAT DU MOT.
 *
 * Il ne dit pas l'issue, il dit si la lecture a été RETOURNÉE. L'écran de fin et le panneau de
 * l'export s'en servent pour choisir leur TITRE : `header.outcome_label` quand rien n'a été
 * permuté (le mot du backend, celui de la Match View), le libellé canonique de l'issue permutée
 * sinon. Sans lui, un match gagné regardé depuis un adversaire affichait « Victoire » au-dessus
 * de l'équipe perdante — le seul mot du panneau, et il était faux.
 *
 * Ces cas n'en touchent aucun autre : la fonction n'existait pas.
 */
describe('victoryIsFlipped — le sujet est-il du camp opposé au joueur de la page ?', () => {
  const NOMME = [
    { xuid: 'me-1', team_side: 't0', is_me: true },
    { xuid: 'ally-2', team_side: 't0', is_me: false },
    { xuid: 'foe-1', team_side: 't1', is_me: false },
    { xuid: 'nomad-9', team_side: null, is_me: false },
  ]

  it('sujet de l’autre camp : vrai', () => {
    expect(victoryIsFlipped(NOMME, 'foe-1')).toBe(true)
  })

  it('sujet = le joueur de la page, ou un coéquipier : faux', () => {
    expect(victoryIsFlipped(NOMME, 'me-1')).toBe(false)
    expect(victoryIsFlipped(NOMME, 'ally-2')).toBe(false)
  })

  it('sans sujet : faux — c’est le régime d’origine, celui du joueur de la page', () => {
    expect(victoryIsFlipped(NOMME, null)).toBe(false)
    expect(victoryIsFlipped(NOMME)).toBe(false)
  })

  it('sujet non situable : faux — rien à permuter, donc rien de permuté', () => {
    expect(victoryIsFlipped(NOMME, 'nomad-9')).toBe(false)
    expect(victoryIsFlipped(NOMME, 'xuid-jamais-vu')).toBe(false)
  })

  it('sans ligne « moi », ou hors d’un match à deux camps : faux', () => {
    expect(victoryIsFlipped(NOMME.map((r) => ({ ...r, is_me: false })), 'foe-1')).toBe(false)
    expect(victoryIsFlipped([...NOMME, { xuid: 't3', team_side: 't2', is_me: false }], 'foe-1')).toBe(false)
    expect(victoryIsFlipped([], 'foe-1')).toBe(false)
  })

  it('IL EST LE MÊME PRÉDICAT QUE LA PERMUTATION DU SCORE, et c’est ce qui les tient ensemble', () => {
    const header = { score_mine: 3, score_theirs: 1 }
    for (const sujet of ['me-1', 'ally-2', 'foe-1', 'nomad-9', 'xuid-jamais-vu', null]) {
      const permute = victoryIsFlipped(NOMME, sujet)
      const attendu = permute ? { ally: 1, enemy: 3 } : { ally: 3, enemy: 1 }
      expect(finalScoreFromHeader(header, NOMME, sujet)).toEqual(attendu)
    }
  })
})
