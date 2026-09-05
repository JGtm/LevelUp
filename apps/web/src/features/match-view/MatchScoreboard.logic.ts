/**
 * MatchScoreboard.logic — logique pure du scoreboard, séparée du rendu.
 *
 * Extraite de MatchScoreboard.tsx pour respecter la limite de 500 lignes par
 * module (CLAUDE.md §14) et faciliter le test unitaire (cf. CLAUDE.md §17 :
 * SRP — un fichier = une responsabilité).
 *
 * Contient uniquement des fonctions stateless :
 *   - getExtremes        : min/max d'une colonne sur le lobby
 *   - cellState          : best/worst/neutral d'une cellule (gère `inverted`)
 *   - cellStyle          : style CSS d'une cellule selon son état
 *   - getMvpLvp          : sélection MVP/LVP au LOBBY (≥2 best / ≥2 worst)
 *   - formatRank/Score   : formatters communs au scoreboard
 *
 * Tests : MatchScoreboard.test.ts (tests unitaires sans rendu).
 */
import type { MatchScoreboardObjective, MatchScoreboardRow } from '@/lib/api/types'

export interface ColDef {
  key: keyof MatchScoreboardRow
  label: string
  inverted: boolean
  fmt?: (v: number) => string
  /** Aide d'en-tête facultative (V72-04) : texte du tooltip ⓘ de la colonne. */
  tooltip?: string
}

export type Extremes = { min: number | null; max: number | null }

export function getExtremes(rows: MatchScoreboardRow[], key: keyof MatchScoreboardRow): Extremes {
  return getExtremesFromValues(rows.map((r) => r[key] as number | null))
}

/**
 * min/max d'une série de valeurs (nulls ignorés). {null,null} si moins de 2
 * valeurs comparables. Sert aussi bien aux colonnes brutes (getExtremes) qu'à la
 * colonne de frags AJUSTÉE pour le départage MVP/LVP (voir mvpKills / getMvpLvp).
 */
export function getExtremesFromValues(values: (number | null)[]): Extremes {
  const vals = values.filter((v): v is number => v != null)
  if (vals.length < 2) return { min: null, max: null }
  return { min: Math.min(...vals), max: Math.max(...vals) }
}

/** Clé de la colonne « Frags » — la seule dont la valeur de départage MVP/LVP est ajustée. */
const KILLS_KEY: keyof MatchScoreboardRow = 'kills'

/**
 * Frags retenus pour DÉPARTAGER le MVP/LVP : frags bruts moins les mécaniques de
 * kill exclues (assassinat, charge spartane / shoulder_bash, coup au sol /
 * ground_pound), qui ne reflètent pas la performance de tir. Retourne null si les
 * frags bruts sont absents. Borné à 0.
 *
 * N'affecte QUE la sélection MVP/LVP : la colonne Frags affichée et son highlight
 * best/worst gardent la valeur brute. Title-agnostic : ces champs sont nil hors
 * Halo 5 → aucun effet sur Infinite. Miroir de analysis.mvpKills côté back.
 */
export function mvpKills(r: MatchScoreboardRow): number | null {
  if (r.kills == null) return null
  const excluded =
    (r.assassination_kills ?? 0) + (r.shoulder_bash_kills ?? 0) + (r.ground_pound_kills ?? 0)
  return Math.max(0, r.kills - excluded)
}

export type CellState = 'best' | 'worst' | 'neutral'

export function cellState(value: number | null, ex: Extremes, inverted: boolean): CellState {
  if (value == null || ex.min == null || ex.max == null || ex.min === ex.max) return 'neutral'
  const isBest = inverted ? value === ex.min : value === ex.max
  if (isBest) return 'best'
  const isWorst = inverted ? value === ex.max : value === ex.min
  if (isWorst) return 'worst'
  return 'neutral'
}

/**
 * Style inline pour une cellule selon son état best/worst.
 *
 * Tokens `outcome-win` / `outcome-loss` accessibility-aware, overridables via
 * les palettes utilisateur (cf. lib/accessibility/palettes/). Cohérent avec
 * la border MVP/LVP sur la cellule gamertag (même rôle visuel : meilleur
 * lobby / pire lobby — voir thought_log 2026-05-07).
 *
 * Background tinté via `color-mix(in oklab, ...)` pour rester lisible avec le
 * texte pleine couleur du même token. `intensityPct` (défaut 28) pilote la
 * densité de la teinte : les callers scoreboard/leaderboard gardent 28 %,
 * l'Explorer passe une valeur plus douce pour ses bandes de décile
 * (ExplorerMatchesTable.highlight, DECILE_TINT_PCT). Pas de hex hardcodé.
 */
export function cellStyle(state: CellState, intensityPct: number = 28): React.CSSProperties {
  if (state === 'neutral') return {}
  const tokenVar = state === 'best' ? 'var(--ac-outcome-win)' : 'var(--ac-outcome-loss)'
  return {
    backgroundColor: `color-mix(in oklab, ${tokenVar} ${intensityPct}%, transparent)`,
    color: tokenVar,
    fontWeight: state === 'best' ? 600 : 500,
  }
}

/**
 * MVP/LVP basé sur le nombre de cellules best/worst par joueur sur les
 * colonnes comparables (mock 4b). Seuil ≥2 best / ≥2 worst pour éviter les
 * faux positifs sur de petits scoreboards. Tie → premier du tri stable.
 *
 * Calculé sur l'ensemble du LOBBY (toutes équipes confondues, hors bots) —
 * le caller filtre les bots avant de passer `rows`. Un seul MVP + un seul
 * LVP par lobby (pas par équipe).
 */
export function getMvpLvp(
  rows: MatchScoreboardRow[],
  cols: ColDef[],
  extremesByKey: Record<string, Extremes>,
): { mvp: string | null; lvp: string | null } {
  if (rows.length < 2) return { mvp: null, lvp: null }
  // Colonne Frags AJUSTÉE (hors assassinat / charge spartane / coup au sol) :
  // recalcule ses propres extremes pour ne pas départager le MVP/LVP sur des
  // kills « gratuits ». Les autres colonnes gardent leurs extremes bruts. Aucune
  // incidence sur la valeur/highlight affichés (uniquement le comptage MVP/LVP).
  const adjustedKills = rows.map(mvpKills)
  const adjustedKillsExtremes = getExtremesFromValues(adjustedKills)
  let mvpXuid: string | null = null
  let mvpBest = 1
  let lvpXuid: string | null = null
  let lvpWorst = 1
  for (let i = 0; i < rows.length; i += 1) {
    const r = rows[i]
    let best = 0
    let worst = 0
    for (const c of cols) {
      const isKills = c.key === KILLS_KEY
      const ex = isKills ? adjustedKillsExtremes : extremesByKey[String(c.key)]
      if (!ex) continue
      const value = isKills ? adjustedKills[i] : (r[c.key] as number | null)
      const state = cellState(value, ex, c.inverted)
      if (state === 'best') best += 1
      else if (state === 'worst') worst += 1
    }
    if (best >= 2 && best > mvpBest) {
      mvpBest = best
      mvpXuid = r.xuid
    }
    if (worst >= 2 && worst > lvpWorst) {
      lvpWorst = worst
      lvpXuid = r.xuid
    }
  }
  return { mvp: mvpXuid, lvp: lvpXuid }
}

export function formatRank(rank: number | null): string {
  if (rank == null) return '—'
  return String(rank)
}

// ---------------------------------------------------------------------------
// Section « Objectifs » (V72-03 ; Stockpile + Extraction : V721-02) — logique pure :
// détection du mode à objectif,
// colonnes pertinentes par mode, agrégat équipe. Le rendu vit dans
// MatchObjectivesSection.tsx ; les libellés viennent de l'i18n match-view.
// ---------------------------------------------------------------------------

/** Mode à objectif détecté sur le scoreboard (blocs mutuellement exclusifs). */
export type ObjectiveMode =
  | 'ctf'
  | 'zones'
  | 'oddball'
  | 'stockpile'
  | 'extraction'
  | 'vip'
  | 'bomb'

/**
 * Colonne objectif : clé du bloc, agrégat équipe (`sum` = cumul ; `max` pour les
 * « meilleurs temps » où une somme n'a pas de sens), et flag durée (secondes →
 * mm:ss). Les libellés/tooltips sont résolus côté rendu via l'i18n (key → label).
 */
export interface ObjectiveColSpec {
  key: keyof MatchScoreboardObjective
  agg: 'sum' | 'max'
  duration?: boolean
}

const CTF_COLS: ObjectiveColSpec[] = [
  { key: 'flag_captures', agg: 'sum' },
  { key: 'flag_returns', agg: 'sum' },
  { key: 'flag_steals', agg: 'sum' },
  { key: 'time_as_flag_carrier_seconds', agg: 'sum', duration: true },
]
const ZONES_COLS: ObjectiveColSpec[] = [
  { key: 'zone_captures', agg: 'sum' },
  { key: 'zone_secures', agg: 'sum' },
  { key: 'time_in_zones_seconds', agg: 'sum', duration: true },
]
const ODDBALL_COLS: ObjectiveColSpec[] = [
  { key: 'skull_grabs', agg: 'sum' },
  { key: 'time_as_skull_carrier_seconds', agg: 'sum', duration: true },
  { key: 'longest_time_as_skull_carrier_seconds', agg: 'max', duration: true },
]
// Stockpile (V721-02) : 6 champs natifs exposés, 4 affichés — dépôts et vols de
// graines d'énergie (le cœur du mode), porteurs adverses éliminés (défense), temps
// de portage. `time_as_power_seed_driver_seconds` et `kills_as_power_seed_carrier`
// restent dans le DTO sans colonne dédiée (parité avec CTF : 11 exposés, 4 affichés).
const STOCKPILE_COLS: ObjectiveColSpec[] = [
  { key: 'power_seeds_deposited', agg: 'sum' },
  { key: 'power_seeds_stolen', agg: 'sum' },
  { key: 'power_seed_carriers_killed', agg: 'sum' },
  { key: 'time_as_power_seed_carrier_seconds', agg: 'sum', duration: true },
]
// Extraction (V721-02) : 5 champs natifs, AUCUNE durée. 4 affichés — l'amorçage
// refusé (`extraction_initiations_denied`) reste exposé sans colonne dédiée.
const EXTRACTION_COLS: ObjectiveColSpec[] = [
  { key: 'successful_extractions', agg: 'sum' },
  { key: 'extraction_initiations_completed', agg: 'sum' },
  { key: 'extraction_conversions_completed', agg: 'sum' },
  { key: 'extraction_conversions_denied', agg: 'sum' },
]
// VIP (V721-02) : 7 champs natifs, 5 affichés — VIP adverses abattus (le score du
// mode), fois désigné VIP, frags réalisés en VIP, temps cumulé et meilleure survie
// en VIP (`max` : une somme de « meilleurs temps » n'a pas de sens, cf. Oddball).
// `vip_assists` et `max_killing_spree_as_vip` restent exposés sans colonne dédiée.
const VIP_COLS: ObjectiveColSpec[] = [
  { key: 'vip_kills', agg: 'sum' },
  { key: 'times_selected_as_vip', agg: 'sum' },
  { key: 'kills_as_vip', agg: 'sum' },
  { key: 'time_as_vip_seconds', agg: 'sum', duration: true },
  { key: 'longest_time_as_vip_seconds', agg: 'max', duration: true },
]
// Assaut (2026-09-05) : LES SEULES STATISTIQUES D'OBJECTIF QUE L'API 343 NE PUBLIE PAS. Elles
// sont reconstruites du film Theater (statborg pour les explosions, canal des armes tenues pour
// le portage, anneau `ti=12` + jointure pour l'armement) et servies par une table dédiée, gatée
// par la capability `film.bomb_stats`. 4 colonnes sur 5 exposées : `bomb_carriers_killed` reste
// dans le DTO SANS colonne dédiée — il est `null` partout à ce jour (la paire tueur/victime
// qu'il demande n'existe pas dans la chaîne de cuisson), et une colonne qui n'afficherait que
// des « — » sur tous les joueurs de tous les matchs ne dit rien.
const BOMB_COLS: ObjectiveColSpec[] = [
  { key: 'bomb_detonations', agg: 'sum' },
  { key: 'bomb_arms', agg: 'sum' },
  { key: 'bomb_grabs', agg: 'sum' },
  { key: 'time_as_bomb_carrier_seconds', agg: 'sum', duration: true },
]

/**
 * detectObjectiveMode — mode à objectif du match d'après le premier bloc non-nil
 * rencontré. null si aucune ligne ne porte de stats objectif (Slayer / titre non
 * supporté → section masquée). Discriminants : deux compteurs toujours présents
 * dans le bloc du mode (une valeur 0 reste non-nil, seul l'absent est nul).
 */
export function detectObjectiveMode(rows: MatchScoreboardRow[]): ObjectiveMode | null {
  for (const r of rows) {
    const o = r.objective
    if (!o) continue
    if (o.flag_grabs != null || o.flag_captures != null) return 'ctf'
    if (o.zone_captures != null || o.zone_scoring_ticks != null) return 'zones'
    if (o.skull_grabs != null || o.skull_scoring_ticks != null) return 'oddball'
    if (o.power_seeds_deposited != null || o.power_seed_carriers_killed != null) return 'stockpile'
    if (o.successful_extractions != null || o.extraction_initiations_completed != null) {
      return 'extraction'
    }
    if (o.times_selected_as_vip != null || o.kills_as_vip != null) return 'vip'
    // Assaut : dernier de la liste, et sans risque de collision — aucun autre mode ne porte de
    // clé `bomb_*`. Les deux compteurs pris ici sont ceux que la chaîne publie ensemble dès
    // qu'une source est lue.
    if (o.bomb_detonations != null || o.bomb_arms != null) return 'bomb'
  }
  return null
}

/** objectiveColsFor — colonnes affichées pour un mode donné. */
export function objectiveColsFor(mode: ObjectiveMode): ObjectiveColSpec[] {
  switch (mode) {
    case 'ctf':
      return CTF_COLS
    case 'zones':
      return ZONES_COLS
    case 'oddball':
      return ODDBALL_COLS
    case 'stockpile':
      return STOCKPILE_COLS
    case 'extraction':
      return EXTRACTION_COLS
    case 'vip':
      return VIP_COLS
    case 'bomb':
      return BOMB_COLS
  }
}

/**
 * objectiveTeamTotal — agrégat équipe d'une colonne (SUM cumulée, ou MAX pour un
 * « meilleur temps »). null si aucune valeur dans le groupe (cellule « — »).
 */
export function objectiveTeamTotal(
  rows: MatchScoreboardRow[],
  col: ObjectiveColSpec,
): number | null {
  const vals = rows
    .map((r) => (r.objective ? (r.objective[col.key] as number | null | undefined) : null))
    .filter((v): v is number => v != null)
  if (vals.length === 0) return null
  return col.agg === 'max' ? Math.max(...vals) : vals.reduce((a, b) => a + b, 0)
}
