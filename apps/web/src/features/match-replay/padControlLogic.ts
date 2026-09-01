/**
 * padControlLogic.ts — QUI A PRIS LES ARMES DE SOCLE, joueur par joueur et camp par camp.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE. Un socle d'arme de puissance est un point de bascule d'une
 * partie : le fusil de précision, l'épée, le lance-roquettes. Le rejeu 2D montre chaque socle se
 * vider à l'image où il se vide, et le bilan d'équipement voisin compte ces vidages SANS
 * RAMASSEUR — « le socle s'est vidé N fois », jamais « untel l'a pris ». Ce module-ci nomme le
 * ramasseur, et c'est tout ce qu'il ajoute : aucune donnée nouvelle, aucun appel de plus.
 *
 * LA SOURCE EST `padPickups[].xuid`, ET IL N'EST PAS DEVINÉ. Le contrat d'origine de ce champ
 * portait la mesure qui l'avait refusé (88,1 % en suivant le slot de vie, 79,7 % en suivant le
 * joueur, contre >= 90 % exigé) ; l'événement natif de ramassage l'a levé (schémas 30-31,
 * `pad_pickup_dating.go`) : quand un ramassage natif de la MÊME FAMILLE d'arme tombe dans la
 * fenêtre `[tLow, tHigh]` d'une occupation, le service publie l'instant exact ET son ramasseur —
 * le slot est exact sur 32/32 paires de vérité terrain. Quand PLUSIEURS y tombent, il s'abstient.
 * Ce module ne rattrape ni ne complète cette abstention : une occupation sans `xuid` n'est
 * comptée pour PERSONNE.
 *
 * CE QUI N'EST PAS ATTRIBUÉ NE DISPARAÎT PAS. Les occupations hors tableau sont comptées et
 * ventilées (`padControlGaps`) : ambiguës, non couvertes, datées sans ramasseur nommé, socles de
 * bonus structurellement hors jointure, et ramasseur nommé mais absent du film. La somme des
 * lignes plus ces manques redonne le nombre d'occupations du document — sans quoi le tableau
 * laisserait croire que le match n'a connu que les prises qu'il affiche.
 *
 * LES SOCLES DE BONUS N'ONT PAS DE RAMASSEUR ICI, ET CE N'EST PAS UN OUBLI : l'identité d'un tel
 * socle est un NOM canonique (`powerup_overshield`), pas un identifiant de famille d'arme, donc
 * aucun ramassage natif ne peut s'y apparier — jamais. Ils sont comptés à part
 * (`powerupOccupations`) plutôt que noyés dans les non couvertes, qui feraient lire « on a
 * cherché et on n'a pas trouvé » là où il n'y avait rien à chercher.
 *
 * LE PONT XUID -> JOUEUR -> ÉQUIPE EST CELUI DU REJEU, réutilisé tel quel (`buildPlayers`,
 * `groupByTeam` de rosterLogic) : le film ne porte AUCUNE équipe (`Track.Team` vaut -1 partout),
 * elle vient du scoreboard. Un joueur du film sans ligne de scoreboard garde sa ligne, sans
 * camp : le trou se montre, il ne se comble pas.
 *
 * Tout est PUR : aucun React, aucune couleur, AUCUNE LANGUE — ce fichier compte, il ne nomme
 * rien. Les libellés d'arme (`padNameFor`) et le rendu vivent dans `MatchPadControlSection.tsx`.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'

import type { ReplayDocumentReady } from './replayNormalize'
import { buildPlayers, groupByTeam, playerName, type ReplayPlayer } from './rosterLogic'

/** Les prises comptées, sans identité — la ligne d'un joueur comme le total d'un camp. */
export interface PadControlTally {
  /** Prises de socle attribuées, tous socles confondus. */
  total: number
  /** Détail par identifiant d'arme du socle (`weaponPads[].weapon`), jamais un libellé. */
  byWeapon: Record<string, number>
}

/** La ligne d'un joueur : son identité, son camp, ses prises. */
export interface PadControlRow extends PadControlTally {
  xuid: string
  /** Nom d'affichage (`displayPlayerName`), jamais un xuid brut. */
  name: string
  /** `team_side` du scoreboard ; `null` = joueur du film absent du scoreboard. */
  side: string | null
}

/** Un camp, ses joueurs (triés par total décroissant) et son total. */
export interface PadControlTeam {
  side: string | null
  players: PadControlRow[]
  total: PadControlTally
}

/**
 * CE QUE LA DATATION A PU FAIRE, repris de `coverage.padDating` — jamais recalculé ici.
 *
 * `hasStats` est faux pour un artefact antérieur au schéma 31 : les seuls nombres sûrs sont
 * alors le nombre d'occupations et celui des prises attribuées, et l'écran ne ventile rien.
 */
export interface PadControlCoverage {
  /** Occupations achevées examinées par la datation. */
  occupations: number
  /** Occupations dont l'instant exact a été publié. */
  dated: number
  /** Occupations dont le ramasseur a été nommé (sous-ensemble de `dated`). */
  named: number
  /** Fenêtres où PLUSIEURS ramassages natifs tombaient : on s'abstient plutôt que de nommer. */
  ambiguous: number
  /** Fenêtres qu'aucun ramassage natif ne couvre. */
  uncovered: number
  /** Occupations de socle de BONUS, structurellement hors jointure (nom canonique). */
  powerupOccupations: number
  /** Faux = artefact sans bloc `padDating` : aucune ventilation n'est affichable. */
  hasStats: boolean
}

/** Le résultat complet : les camps, les colonnes d'arme, et ce qui reste hors tableau. */
export interface PadControl {
  byTeam: PadControlTeam[]
  /** Identifiants d'arme à mettre en colonne, ordre écrit (cf. `weaponsOf`). */
  weapons: string[]
  coverage: PadControlCoverage
  /** Somme des lignes : les prises que le tableau montre réellement. */
  attributed: number
  /**
   * Prises NOMMÉES par le service mais rattachables à aucun joueur du film (xuid inconnu du
   * roster, ou index de socle hors bornes). Distinct des abstentions de la datation.
   */
  unjoined: number
  /** Faux = aucune prise attribuée : l'écran ne doit rien rendre (double porte). */
  hasData: boolean
}

/** Un compteur vide. Chaque appel rend un NOUVEL objet : les tables ne se partagent pas. */
function emptyTally(): PadControlTally {
  return { total: 0, byWeapon: {} }
}

/** Ajoute une prise à un compteur. */
function addPick(tally: PadControlTally, weapon: string): void {
  tally.total += 1
  tally.byWeapon[weapon] = (tally.byWeapon[weapon] ?? 0) + 1
}

/**
 * buildPadControl — l'agrégation complète, en une passe sur `padPickups`.
 *
 * `scoreboard` peut manquer (chargement, titre sans tableau des scores) : les joueurs existent
 * alors tous sans camp, et le tableau les range sous « sans équipe ». Aucun camp n'est deviné.
 */
export function buildPadControl(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[] | undefined,
): PadControl {
  // SEULS LES JOUEURS QUE LE FILM A VUS VIVRE ont une ligne, même règle que le bilan
  // d'équipement : une entrée de roster sans aucune vie n'a pu prendre aucun socle, et une
  // ligne de zéros la ferait passer pour quelqu'un qui n'en a pris aucun.
  const players = buildPlayers(doc, scoreboard ?? []).filter((p) => p.lives.length > 0)
  const known = new Set(players.map((p) => p.xuid))
  const tallies = new Map<string, PadControlTally>()
  const matchTotal: Record<string, number> = {}
  let unjoined = 0

  for (const pick of doc.padPickups) {
    // Pas de ramasseur nommé : la datation s'est abstenue, et on ne rattrape rien.
    if (!pick.xuid) continue
    // `pad` est un INDEX dans `weaponPads` (ordre stable garanti côté build) : un index hors
    // bornes ne compte pour aucun socle voisin — il rejoint les prises non rattachées.
    const pad = doc.weaponPads[pick.pad]
    if (!pad || !known.has(pick.xuid)) {
      unjoined += 1
      continue
    }
    let tally = tallies.get(pick.xuid)
    if (!tally) {
      tally = emptyTally()
      tallies.set(pick.xuid, tally)
    }
    addPick(tally, pad.weapon)
    matchTotal[pad.weapon] = (matchTotal[pad.weapon] ?? 0) + 1
  }

  const byTeam = teamsOf(players, tallies)
  const attributed = byTeam.reduce((sum, team) => sum + team.total.total, 0)
  return {
    byTeam,
    weapons: weaponsOf(matchTotal),
    coverage: coverageOf(doc),
    attributed,
    unjoined,
    hasData: attributed > 0,
  }
}

/**
 * teamsOf range les joueurs par camp, somme chaque camp, et TRIE PAR TOTAL DÉCROISSANT — à
 * l'intérieur d'un camp comme entre les camps.
 *
 * LE TRI EST LE SUJET DU TABLEAU : « qui a contrôlé les socles » se lit de haut en bas, et un
 * ordre de roster obligerait à comparer des nombres dispersés. À égalité, le nom (puis le camp)
 * départage : deux relectures du même match donnent le même tableau.
 */
function teamsOf(
  players: readonly ReplayPlayer[],
  tallies: ReadonlyMap<string, PadControlTally>,
): PadControlTeam[] {
  const teams = groupByTeam([...players]).map((group) => {
    const total = emptyTally()
    const rows: PadControlRow[] = group.players.map((p) => {
      const tally = tallies.get(p.xuid) ?? emptyTally()
      total.total += tally.total
      for (const [weapon, n] of Object.entries(tally.byWeapon)) {
        total.byWeapon[weapon] = (total.byWeapon[weapon] ?? 0) + n
      }
      return {
        ...tally,
        xuid: p.xuid,
        name: displayPlayerName(playerName(p), p.xuid),
        side: group.side,
      }
    })
    rows.sort((a, b) => b.total - a.total || a.name.localeCompare(b.name))
    return { side: group.side, players: rows, total }
  })
  // Le sentinelle de tri des camps sans nom est celui de `groupByTeam` : ils passent en dernier.
  teams.sort(
    (a, b) => b.total.total - a.total.total || (a.side ?? '￿').localeCompare(b.side ?? '￿'),
  )
  return teams
}

/**
 * weaponsOf retient les socles qu'au moins une prise attribuée justifie, du plus disputé au
 * moins disputé.
 *
 * UNE COLONNE DE ZÉROS N'EST PAS UNE MESURE : un socle que personne n'a pris n'a pas de colonne,
 * il reste dans le compte des occupations non attribuées. À égalité, l'identifiant départage —
 * l'ordre ne dépend jamais de l'ordre de rencontre dans le film.
 */
function weaponsOf(matchTotal: Record<string, number>): string[] {
  return Object.keys(matchTotal).sort((a, b) => matchTotal[b] - matchTotal[a] || a.localeCompare(b))
}

/** coverageOf recopie les compteurs de la datation — aucun n'est recalculé ni deviné. */
function coverageOf(doc: ReplayDocumentReady): PadControlCoverage {
  const stats = doc.coverage?.padDating
  if (!stats) {
    // Artefact sans bloc de datation : le nombre d'occupations reste vrai (c'est la longueur du
    // calque), tout le reste serait inventé. `hasStats` faux le dit à l'écran.
    return {
      occupations: doc.padPickups.length,
      dated: 0,
      named: 0,
      ambiguous: 0,
      uncovered: 0,
      powerupOccupations: 0,
      hasStats: false,
    }
  }
  return {
    occupations: stats.occupations,
    dated: stats.dated,
    named: stats.named,
    ambiguous: stats.ambiguous,
    uncovered: stats.uncovered,
    powerupOccupations: stats.powerupOccupations,
    hasStats: true,
  }
}

/** Les cinq raisons pour lesquelles une occupation n'est pas dans le tableau. */
export type PadControlGapKey = 'ambiguous' | 'uncovered' | 'unnamed' | 'powerup' | 'unjoined'

/** Une raison et son compte. */
export interface PadControlGap {
  key: PadControlGapKey
  count: number
}

/**
 * padControlGaps ventile les occupations que le tableau ne montre pas.
 *
 * LA SOMME DOIT RETOMBER SUR SES PIEDS : prises affichées + manques = occupations du document.
 * `unnamed` (datée sans ramasseur nommé) est le reste que les quatre autres n'expliquent pas —
 * le calculer par soustraction plutôt que le lire évite d'afficher une ventilation qui ne boucle
 * pas quand un compteur du service évolue.
 *
 * Sans bloc de datation (`hasStats` faux), aucune ventilation n'est rendue : le seul manque
 * connu est alors le nombre d'occupations moins les prises affichées, et l'écran l'écrit tel
 * quel sans prétendre en connaître la cause.
 */
export function padControlGaps(control: PadControl): PadControlGap[] {
  const cov = control.coverage
  if (!cov.hasStats) return []
  const explained = cov.ambiguous + cov.uncovered + cov.powerupOccupations + control.unjoined
  const unnamed = Math.max(0, cov.occupations - control.attributed - explained)
  return [
    { key: 'ambiguous' as const, count: cov.ambiguous },
    { key: 'uncovered' as const, count: cov.uncovered },
    { key: 'unnamed' as const, count: unnamed },
    { key: 'powerup' as const, count: cov.powerupOccupations },
    { key: 'unjoined' as const, count: control.unjoined },
  ].filter((g) => g.count > 0)
}

/** Le nombre total d'occupations hors tableau, ventilables ou non. */
export function padControlMissing(control: PadControl): number {
  return Math.max(0, control.coverage.occupations - control.attributed)
}
