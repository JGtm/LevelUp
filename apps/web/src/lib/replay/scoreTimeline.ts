/**
 * scoreTimeline.ts — LE SCORE À L'INSTANT LU, et ce que le film a le droit de dire.
 *
 * POURQUOI DANS `lib/` ET NON DANS UNE FEATURE. Deux surfaces lisent ce calque : le rejeu 2D
 * (en-têtes d'équipe, fiches joueur, frise) et la vue match (la courbe « Score dans le
 * temps »). Tant qu'il vivait dans `features/match-replay/`, la vue match devait aller le
 * chercher chez sa voisine — un import cross-feature que le ratchet P8.5
 * (`tools/lint-cross-feature-imports.mjs`) refuse à juste titre : la règle du dépôt
 * (`frontend-patterns`) veut la logique PURE partagée dans `lib/`, jamais dans une feature
 * importée par une autre. Ce fichier est donc le foyer, et les deux features en sont
 * clientes à égalité.
 *
 * IL NE CONNAÎT AUCUNE FEATURE, et c'est ce qui le rend déplaçable : les types du document
 * viennent du contrat généré (`lib/api/types`), et tout ce qu'il attend d'un document ou
 * d'un index de joueurs est décrit STRUCTURELLEMENT ici même (`ReplayScoreDocument`,
 * `AllyIndex`). `ReplayDocumentReady` et `XuidMeta` s'y conforment sans que `lib/` ait à
 * les nommer.
 *
 * CE QUE PUBLIE L'ARTEFACT (schéma 12). Le décodeur du film ne transmet pas un score
 * échantillonné : il transmet les CHANGEMENTS. Une série est donc une suite de PALIERS —
 * `{t, v}` veut dire « à partir de la frame t, la valeur est v », et rien n'est émis tant
 * que rien ne bouge. Lire cette série au frame courant, c'est prendre la dernière valeur
 * transmise au plus tard à ce frame : exactement le patron de `heldReading`
 * (match-replay/replayLogic), transposé à une grandeur qui ne vieillit pas. Un score ne
 * PÂLIT pas — il n'y a pas de lecture « ancienne » d'un score, il n'y a que le score, qui
 * reste ce qu'il est jusqu'au point suivant. C'est pourquoi cette lecture rend un nombre et
 * non un `HeldReading`.
 *
 * DEUX GRANDEURS QUI NE SE CONFONDENT PAS. `total` est le cumul du match, `rounds[]` la
 * valeur DANS chaque manche — celle qui repart de zéro à la manche suivante d'un Oddball.
 * Le témoin `24dbb67d` le montre : manches 100/78 puis 100/43, total 200/121. Afficher
 * l'un pour l'autre est un contresens de 79 points.
 *
 * UNE ÉQUIPE SANS SÉRIE VAUT ZÉRO, ET C'EST UNE MESURE. Le témoin CTF `530820e5` (3-0) ne
 * publie qu'UNE série d'équipe : le camp qui n'a jamais marqué n'émet rien du tout. Son
 * score n'est pas inconnu — il est nul, et le film le dit en se taisant. C'est le seul
 * endroit de ce module où une absence se lit comme un zéro, et `teamSeriesFor` est ce qui
 * le concentre en un point (cf. son commentaire).
 *
 * UN JOUEUR SANS SÉRIE, LUI, N'EST PAS À ZÉRO — IL N'EST PAS PUBLIÉ. La distinction est
 * l'inverse de la précédente et elle est vitale : sur le témoin Slayer, 6 joueurs sur 8 ont
 * des compteurs, les 2 autres n'en ont aucun (mesure de la phase 0 : Oddball 0/32, mode
 * entier sans compteurs joueur). Afficher « 0 frag » pour quelqu'un que le film n'a pas
 * apparié serait une affirmation fausse. D'où `playerCountersAt` qui rend `null`, jamais
 * un objet de zéros.
 *
 * Tout ce fichier est PUR : ni React, ni canvas, ni DOM.
 */
import { parseTeamSideID } from '@/lib/halo/teamNames'
import type {
  MatchScoreboardRow,
  ReplayPlayerScore,
  ReplayScoreRound,
  ReplayScoreSeries,
  ReplayScoreTick,
  ReplayScoreTimeline,
  ReplayTeamScore,
} from '@/lib/api/types'

// ---------------------------------------------------------------------------------------
// Les types du calque, une fois comblés
// ---------------------------------------------------------------------------------------

/**
 * LES QUATRE ÉTAGES DU CALQUE DE SCORE (schéma 12), et pourquoi ils demandent quatre types.
 *
 * `scoreTimeline` est le premier champ du document qui empile des tableaux nullables sur
 * QUATRE niveaux : `teams` → `rounds` → `points`, et `players` → `score|kills|deaths|
 * assists` → `rounds` → `points`. La garde de tête du document
 * (`match-replay/replayContract.test.ts`) ne voit que la racine ; c'est son walker de
 * chemins qui exige désormais chacun de ces niveaux, et ce sont ces types-là qui prouvent
 * au compilateur qu'ils sont comblés.
 */
export type ReplayScoreRoundReady = Omit<ReplayScoreRound, 'points'> & {
  points: ReplayScoreTick[]
}
export type ReplayScoreSeriesReady = Omit<ReplayScoreSeries, 'rounds' | 'total'> & {
  rounds: ReplayScoreRoundReady[]
  total: ReplayScoreTick[]
}
export type ReplayTeamScoreReady = Omit<ReplayTeamScore, 'rounds' | 'total'> & {
  rounds: ReplayScoreRoundReady[]
  total: ReplayScoreTick[]
}
export type ReplayPlayerScoreReady = Omit<
  ReplayPlayerScore,
  'assists' | 'deaths' | 'kills' | 'score'
> & {
  assists: ReplayScoreSeriesReady
  deaths: ReplayScoreSeriesReady
  kills: ReplayScoreSeriesReady
  score: ReplayScoreSeriesReady
}
export type ReplayScoreTimelineReady = Omit<ReplayScoreTimeline, 'players' | 'teams'> & {
  players: ReplayPlayerScoreReady[]
  teams: ReplayTeamScoreReady[]
}

/**
 * ReplayScoreDocument — le strict minimum qu'un document de rejeu doit porter pour que ces
 * lectures aient un sens.
 *
 * STRUCTUREL, ET NON NOMINAL : `ReplayDocumentReady` (la frontière de `match-replay`) s'y
 * conforme sans que `lib/` ait à le connaître. C'est exactement ce qui permet à cette
 * logique de vivre hors des features — la dépendance va de la feature vers `lib/`, jamais
 * l'inverse.
 */
export interface ReplayScoreDocument {
  originMs?: number
  coverage?: { originResolved?: boolean }
  scoreTimeline?: ReplayScoreTimelineReady
}

/**
 * AllyIndex — de quel côté est un joueur, par xuid.
 *
 * Décrit structurellement pour la même raison : `XuidMeta` (match-view) porte aussi le
 * gamertag, dont ce module n'a que faire. Un index plus riche reste acceptable.
 */
type AllyIndex = ReadonlyMap<string, { ally: boolean }>

// ---------------------------------------------------------------------------------------
// La frontière : combler les cinq tableaux nullables du calque
// ---------------------------------------------------------------------------------------

/** Une manche dont les paliers sont comblés : un tableau vide dit « aucun point marqué ». */
function normalizeRound(r: ReplayScoreRound): ReplayScoreRoundReady {
  return { ...r, points: r.points ?? [] }
}

/**
 * Une série (score, frags, morts, assistances) d'un joueur, ses deux tableaux comblés.
 *
 * `series` peut manquer alors que le contrat la déclare obligatoire : c'est justement le
 * rôle d'une frontière de ne pas tomber sur un producteur qui déraille. Le résultat est
 * vide, jamais inventé — et ce qui distingue « ce joueur n'est pas publié » de « ce joueur
 * est à zéro » se joue un cran plus haut, sur sa PRÉSENCE dans `players`.
 */
function normalizeSeries(series: ReplayScoreSeries | undefined): ReplayScoreSeriesReady {
  return {
    ...series,
    rounds: (series?.rounds ?? []).map(normalizeRound),
    total: series?.total ?? [],
  }
}

/**
 * normalizeScoreTimeline comble les CINQ tableaux nullables du calque de score
 * (`teams`, `players`, `rounds`, `total`, `points`) et rend l'absence du calque telle
 * quelle : `undefined` entre, `undefined` sort.
 *
 * L'OBJET GARDE LE DROIT D'ÊTRE ABSENT : un artefact de schéma antérieur à 12 n'en porte
 * aucun, et un objet vide se lirait « le film a été lu, il n'y avait pas de score ».
 */
export function normalizeScoreTimeline(
  raw: ReplayScoreTimeline | undefined,
): ReplayScoreTimelineReady | undefined {
  if (!raw) return undefined
  return {
    ...raw,
    players: (raw.players ?? []).map((p) => ({
      ...p,
      assists: normalizeSeries(p.assists),
      deaths: normalizeSeries(p.deaths),
      kills: normalizeSeries(p.kills),
      score: normalizeSeries(p.score),
    })),
    teams: (raw.teams ?? []).map((t) => ({
      ...t,
      rounds: (t.rounds ?? []).map(normalizeRound),
      total: t.total ?? [],
    })),
  }
}

// ---------------------------------------------------------------------------------------
// La garde d'horloge
// ---------------------------------------------------------------------------------------

/**
 * filmClockTrusted — l'artefact sait-il OÙ commence son temps ?
 *
 * POURQUOI CETTE GARDE EXISTE (P2 de la revue adversariale du lot A phase 1, registre du
 * 2026-08-18). Les calques `objectives[]` et `scoreTimeline` sont datés par l'horloge du
 * FILM, puis recalés sur la grille du document en retranchant `originMs`. Quand l'origine
 * n'a pas pu être résolue ET qu'aucune valeur n'est publiée, ce recalage n'a pas eu lieu :
 * les frames sont décalées d'un écart INCONNU — mesuré de 3,6 s à 50,8 s selon le match.
 * Un score qui tique 30 s trop tôt est pire qu'un score absent : il se lit comme juste.
 *
 * `coverage.originResolved` est un booléen NON pointeur : un artefact de schéma 11 servi
 * tel quel dit donc `false` alors qu'il porte un `originMs` parfaitement valide. C'est
 * exactement pour cela que la règle exige LES DEUX conditions — le drapeau à `false` seul
 * ne prouve rien sur les documents antérieurs au champ.
 */
export function filmClockTrusted(doc: ReplayScoreDocument): boolean {
  return !(doc.coverage?.originResolved === false && doc.originMs == null)
}

/**
 * scoreTimelineOf — LE SEUL accès au calque de score, et il porte la garde d'horloge.
 *
 * Passer par un point unique plutôt que par `doc.scoreTimeline` évite qu'un consommateur
 * futur oublie la garde : l'oubli ne se verrait pas à l'écran, il se lirait comme une
 * mesure. `undefined` veut dire « rien à afficher » — artefact antérieur au schéma 12,
 * mode sans compteur, ou horloge non recalée.
 */
export function scoreTimelineOf(doc: ReplayScoreDocument): ReplayScoreTimelineReady | undefined {
  if (!filmClockTrusted(doc)) return undefined
  return doc.scoreTimeline
}

// ---------------------------------------------------------------------------------------
// Les lectures au frame courant
// ---------------------------------------------------------------------------------------

/**
 * scoreAtFrame rend la valeur du palier courant : la dernière transmise au plus tard à
 * `frame`, et 0 avant le premier point.
 *
 * ZÉRO AVANT LE PREMIER POINT N'EST PAS UNE INVENTION : la série ne commence qu'au premier
 * changement, et avant le premier changement le score EST nul — c'est le début du match.
 * Recherche par dichotomie : la frise se déplace au frame près et les séries d'équipe
 * comptent jusqu'à 800 paliers.
 */
export function scoreAtFrame(points: readonly ReplayScoreTick[], frame: number): number {
  if (points.length === 0 || frame < points[0].t) return 0
  let lo = 0
  let hi = points.length - 1
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (points[mid].t <= frame) lo = mid
    else hi = mid - 1
  }
  return points[lo].v
}

/**
 * teamSeriesFor rend la série d'une équipe, ou `null` quand le film n'en publie aucune.
 *
 * `null` ET ZÉRO DISENT LA MÊME CHOSE ICI, contrairement au reste du calque : une équipe
 * qui n'a jamais marqué n'émet aucun palier (témoin CTF 3-0, une seule série pour deux
 * camps). Les appelants qui veulent un nombre passent par `teamScoreAtFrame`, qui rend 0 —
 * la vérité du film. Ceux qui veulent savoir si le camp a une série (pour ne pas dessiner
 * une courbe plate là où il n'y a pas de donnée) lisent ce `null`.
 */
export function teamSeriesFor(
  timeline: ReplayScoreTimelineReady | undefined,
  teamId: number | null,
): ReplayTeamScoreReady | null {
  if (!timeline || teamId == null) return null
  return timeline.teams.find((t) => t.teamId === teamId) ?? null
}

/** teamScoreAtFrame — le TOTAL de l'équipe au frame courant ; 0 sans série (cf. ci-dessus). */
export function teamScoreAtFrame(
  timeline: ReplayScoreTimelineReady | undefined,
  teamId: number | null,
  frame: number,
): number {
  return scoreAtFrame(teamSeriesFor(timeline, teamId)?.total ?? [], frame)
}

/** teamIdOfSide traduit le camp du scoreboard (`t{N}`) en identifiant d'équipe du film. */
export function teamIdOfSide(side: string | null | undefined): number | null {
  return parseTeamSideID(side)
}

/** La manche courante d'une équipe et sa valeur DANS cette manche. */
export interface RoundReading {
  /** Numéro de manche tel que le film l'écrit (0 pour un mode à manche unique). */
  round: number
  /** Rang d'affichage, à partir de 1 : « manche 2 » se lit mieux que « manche 1 ». */
  index: number
  /** Valeur atteinte dans CETTE manche au frame courant (elle repart de zéro à la suivante). */
  value: number
  /** Nombre total de manches publiées pour cette équipe. */
  count: number
}

/**
 * roundAtFrame rend la manche en cours au frame donné, ou `null` si l'équipe n'a pas de
 * série. La manche courante est la DERNIÈRE dont le premier palier est déjà passé ; avant
 * le premier palier de toutes, c'est la première (le match a commencé, pas le compteur).
 */
export function roundAtFrame(team: ReplayTeamScoreReady | null, frame: number): RoundReading | null {
  if (!team || team.rounds.length === 0) return null
  let idx = 0
  for (let i = 0; i < team.rounds.length; i++) {
    const first = team.rounds[i].points[0]
    if (first && first.t <= frame) idx = i
  }
  const round = team.rounds[idx]
  return {
    round: round.round,
    index: idx + 1,
    value: scoreAtFrame(round.points, frame),
    count: team.rounds.length,
  }
}

/** Les quatre compteurs vivants d'un joueur, au frame courant. */
export interface PlayerCounters {
  score: number
  kills: number
  deaths: number
  assists: number
}

/** playerScoreFor rend la fiche de compteurs d'un joueur, ou `null` s'il n'est pas publié. */
export function playerScoreFor(
  timeline: ReplayScoreTimelineReady | undefined,
  xuid: string,
): ReplayPlayerScoreReady | null {
  if (!timeline) return null
  return timeline.players.find((p) => p.xuid === xuid) ?? null
}

/**
 * playerCountersAt rend les quatre compteurs d'un joueur au frame courant, ou `null`.
 *
 * `null` VEUT DIRE « CE JOUEUR N'EST PAS PUBLIÉ », jamais « il est à zéro ». La fiche
 * n'affiche alors PAS la ligne vivante et garde ses totaux de fin de match, qui viennent de
 * la base — deux grandeurs différentes, jamais mélangées silencieusement.
 */
export function playerCountersAt(
  timeline: ReplayScoreTimelineReady | undefined,
  xuid: string,
  frame: number,
): PlayerCounters | null {
  const p = playerScoreFor(timeline, xuid)
  if (!p) return null
  return {
    score: scoreAtFrame(p.score.total, frame),
    kills: scoreAtFrame(p.kills.total, frame),
    deaths: scoreAtFrame(p.deaths.total, frame),
    assists: scoreAtFrame(p.assists.total, frame),
  }
}

// ---------------------------------------------------------------------------------------
// Les retournements
// ---------------------------------------------------------------------------------------

/** Un RETOURNEMENT : l'instant où le match change de meneur. */
export interface LeadChange {
  /** Frame du document où le nouveau meneur passe devant. */
  frame: number
  /** Identifiant d'équipe du nouveau meneur. */
  teamId: number
}

/**
 * leadChanges rend les instants où le meneur CHANGE — les retournements du match.
 *
 * CE QU'UN RETOURNEMENT EST, ET N'EST PAS. Le meneur est l'équipe seule en tête ; une
 * égalité n'a pas de meneur et ne compte donc pas comme un retournement — elle le SUSPEND.
 * Reprendre la tête après une égalité, en revanche, EST un retournement si c'est l'autre
 * camp qui la reprend : c'est le comparatif au dernier meneur CONNU qui tranche, pas au
 * dernier instant. La première prise de tête du match n'en est pas un non plus : personne
 * n'était devant avant elle.
 *
 * Généralise à plus de deux camps (grande bataille) sans rien changer : le meneur est
 * l'argmax UNIQUE ; deux camps à égalité en tête ne désignent personne.
 *
 * Mesure sur les trois témoins (2026-08-18) : Slayer `000d5950` 0 retournement (le camp
 * gagnant mène de bout en bout, deux égalités à 1 et à 2), CTF `530820e5` 0 (un seul camp
 * marque), Oddball `24dbb67d` 3 — c'est le témoin de cette marque.
 */
export function leadChanges(timeline: ReplayScoreTimelineReady | undefined): LeadChange[] {
  const out: LeadChange[] = []
  let previous: number | null = null
  for (const state of leaderStates(timeline)) {
    // UNE ÉGALITÉ NE FERME PAS LE MENEUR PRÉCÉDENT ici : elle le suspend (cf. l'en-tête). Le
    // comparatif se fait au dernier meneur CONNU, pas au dernier état.
    if (state.teamId == null) continue
    if (previous != null && state.teamId !== previous) out.push({ frame: state.frame, teamId: state.teamId })
    previous = state.teamId
  }
  return out
}

/**
 * Un ÉTAT de la course au score : à partir de cette frame, voilà qui mène — ou personne.
 * `teamId: null` EST une mesure : les camps sont à égalité, pas « inconnus ».
 */
export interface LeadState {
  frame: number
  teamId: number | null
}

/**
 * leaderStates rend la suite des états de la course au score : un état par CHANGEMENT, égalités
 * comprises. C'est la lecture complète dont `leadChanges` n'est qu'une projection (les seuls
 * retournements, l'égalité effacée) — les deux ne peuvent donc pas diverger, et il n'y a qu'une
 * implémentation du « qui mène » dans le dépôt.
 *
 * POURQUOI LES ÉGALITÉS SORTENT ICI. La piste SCORE de la frise du rejeu les PEINT (encre
 * d'égalité du dépôt) : un mode où les deux camps se tiennent longtemps a une piste bleue, et
 * c'est un fait du match. `leadChanges`, lui, date des retournements — une égalité n'en est
 * pas un.
 *
 * Le premier état est celui du PREMIER PALIER publié, quel qu'il soit : avant lui, le calque ne
 * dit rien (0-0 est l'affaire de l'appelant, qui seul sait où commence sa frise).
 */
export function leaderStates(timeline: ReplayScoreTimelineReady | undefined): LeadState[] {
  // Une équipe SANS identifiant ne peut être ni meneuse ni menée : la comparer reviendrait
  // à couronner la seule qui en a un (`coverage.score.teamIdentity = "unresolved"`).
  const teams = (timeline?.teams ?? []).filter((t) => t.teamId != null)
  if (teams.length < 2) return []
  const frames = [...new Set(teams.flatMap((t) => t.total.map((p) => p.t)))].sort((a, b) => a - b)
  const out: LeadState[] = []
  for (const frame of frames) {
    const leader = leaderAt(teams, frame)
    if (out.length > 0 && leader === out[out.length - 1].teamId) continue
    out.push({ frame, teamId: leader })
  }
  return out
}

/**
 * allyOfTeamId dit si une équipe DU FILM est du côté du joueur de la page.
 *
 * Le film numérote ses équipes (`teamId`) ; la page raisonne en « allié / adverse », une
 * notion RELATIVE au joueur consulté. Le pont entre les deux passe par le scoreboard, seul
 * endroit où le camp (`team_side` au format `t{N}`) et le xuid coexistent. `null` = camp
 * introuvable ou aucun joueur reconnu : la marque prend une encre neutre, jamais l'une des
 * deux couleurs par défaut.
 */
export function allyOfTeamId(
  scoreboard: ReadonlyArray<Pick<MatchScoreboardRow, 'xuid' | 'team_side'>>,
  allies: AllyIndex | undefined,
  teamId: number,
): boolean | null {
  if (!allies) return null
  for (const row of scoreboard) {
    if (parseTeamSideID(row.team_side) !== teamId) continue
    const meta = allies.get(row.xuid)
    if (meta) return meta.ally
  }
  return null
}

/** leaderAt rend l'équipe SEULE en tête au frame donné, ou `null` (égalité, aucun point). */
function leaderAt(teams: readonly ReplayTeamScoreReady[], frame: number): number | null {
  let best = -1
  let bestTeam: number | null = null
  let tied = false
  for (const t of teams) {
    if (t.teamId == null) continue
    const v = scoreAtFrame(t.total, frame)
    if (v > best) {
      best = v
      bestTeam = t.teamId
      tied = false
    } else if (v === best) {
      tied = true
    }
  }
  return tied ? null : bestTeam
}
