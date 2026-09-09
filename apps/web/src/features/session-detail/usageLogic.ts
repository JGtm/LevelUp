/**
 * usageLogic.ts — LA PROJECTION DU BLOC « usages d'équipement, armes spéciales et objectifs »
 * d'une session (chantier session-usage S3, handoff §1 : la grammaire des formes).
 *
 * TOUT AXE EST NORMALISÉ (doctrine §1) : parts en %, cadences par dix minutes. Les
 * comptes bruts ne sortent d'ici QUE comme textes d'honnêteté (dénominateur écrit à
 * côté d'un taux), jamais comme valeur d'axe. La référence — l'équipe d'en face —
 * n'apparaît dans aucun modèle : elle n'existe que comme complément du dénominateur
 * et comme segment HACHURÉ anonyme de la piste du lobby.
 *
 * `undefined` N'EST PAS ZÉRO : le contrat (domain/session_usage.go) omet un champ
 * quand son dénominateur est nul ou son scope vide (0/0 n'est pas 0 %). Chaque
 * projection rend alors « non mesuré » (`null` → tiret), jamais un zéro inventé —
 * même règle que `valueGridModel`.
 *
 * Pur : aucun React, aucune couleur en dur, aucune lecture de store — les libellés
 * viennent de `usageI18n`, les encres arrivent par callbacks de l'appelant.
 */
import type { SemanticToken } from '@/lib/accessibility'
import type {
  SessionObjectiveRoleMetric,
  SessionUsageBlock,
  SessionUsageMatchPoint,
  SessionUsageMetric,
  SessionUsageOutcomes,
  SessionUsageSquadPlayer,
  SessionUsageSquadShare,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { deployedFamilyLabel, equipmentFamilyLabel, type UsageText } from './usageI18n'

// ─── Formatage (parts, cadences, comptes, durées) ────────────────────────────────

/** Nombre à `digits` décimales, virgule en FR — jamais de séparateur de milliers. */
export function formatUsageDecimal(v: number, locale: Locale, digits = 1): string {
  const s = v.toFixed(digits)
  return locale === 'fr' ? s.replace('.', ',') : s
}

/** Une part en pourcentage : « 45,6 % » / "45.6%". `null`/absent → tiret. */
export function formatUsagePct(v: number | null | undefined, locale: Locale): string {
  if (v == null) return '—'
  const s = formatUsageDecimal(v, locale)
  return locale === 'fr' ? `${s} %` : `${s}%`
}

/** Une cadence par dix minutes : une décimale. `null`/absent → tiret. */
export function formatUsageRate(v: number | null | undefined, locale: Locale): string {
  if (v == null) return '—'
  return formatUsageDecimal(v, locale)
}

/**
 * Un TOTAL brut (dénominateur d'honnêteté). Entier écrit tel quel ; une grandeur en
 * durée (rôle « tenir ») s'écrit m:ss — et un 0 MESURÉ s'écrit « 0:00 », jamais un
 * tiret (le tiret est réservé au non-mesuré ; cf. l'en-tête de
 * `lib/formatters/duration.ts`, dont le repli sur 0 ne convient pas ici).
 */
export function formatUsageCount(
  v: number | null | undefined,
  locale: Locale,
  isDuration = false,
): string {
  if (v == null) return '—'
  if (isDuration) {
    const total = Math.max(0, Math.round(v))
    const m = Math.floor(total / 60)
    const s = total % 60
    return `${m}:${s.toString().padStart(2, '0')}`
  }
  return Number.isInteger(v) ? String(v) : formatUsageDecimal(v, locale)
}

// ─── Grandeurs : ordre canonique, nature, libellés, encres ───────────────────────

/** La nature d'une grandeur du contrat — décide libellé, encre et bloc d'accueil. */
export type UsageMetricKind =
  | 'camo'
  | 'overshield'
  | 'wall'
  | 'deployed_other'
  | 'grapple'
  | 'dropped'
  | 'pads'
  | 'equipment'
  | 'other'

const DEPLOYED_PREFIX = 'deployed_'
/**
 * EQUIPMENT_PREFIX — LE BILAN D'ÉQUIPEMENT par famille (étape E4, jumeau web de
 * `sessionusage.MetricEquipmentPrefix`). Sa valeur est un compte d'OBJETS
 * (utilisé + gardé + lâché) et non de gestes ; c'est la seule famille de clés qui
 * porte `SessionUsageMetric.outcomes` — voir `equipmentMetrics` pour la règle de
 * substitution face à `deployed_<famille>` / `camo_episodes` / `overshield_episodes`.
 */
const EQUIPMENT_PREFIX = 'equipment_'

/** metricKind classe une clé du contrat (ensembles ouverts côté `deployed_*` et `equipment_*`). */
export function metricKind(key: string): UsageMetricKind {
  switch (key) {
    case 'camo_episodes':
      return 'camo'
    case 'overshield_episodes':
      return 'overshield'
    case 'deployed_wall':
      return 'wall'
    case 'grapple_pulls':
      return 'grapple'
    case 'dropped_objects':
      return 'dropped'
    case 'pad_pickups':
      return 'pads'
    default:
      if (key.startsWith(EQUIPMENT_PREFIX)) return 'equipment'
      return key.startsWith(DEPLOYED_PREFIX) ? 'deployed_other' : 'other'
  }
}

/** L'ordre d'affichage du bloc équipement (les inconnues ferment la marche). */
const METRIC_RANK: Partial<Record<UsageMetricKind, number>> = {
  camo: 0,
  overshield: 1,
  equipment: 2,
  wall: 3,
  deployed_other: 4,
  grapple: 5,
  other: 7,
}

/** metricLabel — le libellé bilingue d'une grandeur (vocabulaire du handoff). */
export function metricLabel(key: string, t: UsageText): string {
  switch (metricKind(key)) {
    case 'camo':
      return t.metricCamo
    case 'overshield':
      return t.metricOvershield
    case 'wall':
      return t.metricWall
    case 'grapple':
      return t.metricGrapple
    case 'dropped':
      return t.metricDropped
    case 'pads':
      return t.metricPads
    case 'equipment':
      return equipmentFamilyLabel(key.slice(EQUIPMENT_PREFIX.length), t)
    case 'deployed_other':
      return deployedFamilyLabel(key.slice(DEPLOYED_PREFIX.length), t)
    case 'other':
      return key
  }
}

/**
 * equipmentBilanFamilyOf — l'identité de famille (vocabulaire du bilan) que porte
 * une grandeur SUSCEPTIBLE D'ÊTRE SUPPLANTÉE par son équivalent `equipment_<famille>`
 * (voir `equipmentMetrics`). `null` pour toute autre clé (grapple, pads, inconnue…).
 */
function equipmentBilanFamilyOf(key: string): string | null {
  switch (key) {
    case 'camo_episodes':
      return 'powerup_camo'
    case 'overshield_episodes':
      return 'powerup_overshield'
    default:
      return key.startsWith(DEPLOYED_PREFIX) ? key.slice(DEPLOYED_PREFIX.length) : null
  }
}

/**
 * L'ENCRE DE CHAQUE FAMILLE DE GESTE — 2e copie ASSUMÉE de `USAGE_GROUP_TOKENS`
 * (match-replay/equipmentUsageChart.ts, import croisé interdit par le ratchet
 * lint-cross-feature-imports) : mêmes jetons pour que la session se lise avec la
 * même convention de couleur que la vue match. À la 3e copie : centraliser dans
 * `components/` et poser le garde-rail (règle CLAUDE.md n°6).
 */
export const USAGE_METRIC_TOKENS: Record<UsageMetricKind, SemanticToken> = {
  grapple: 'frag-sidearm', // vert — comme la famille grappin de la vue match
  camo: 'frag-heavy', // violet — états actifs
  overshield: 'frag-heavy', // violet — états actifs (même famille que camo)
  wall: 'frag-shoulder', // cyan — poses
  deployed_other: 'frag-shoulder', // cyan — poses
  // Même jeton que le groupe fusionné `equipment` de la vue match
  // (match-replay/equipmentUsageChart.ts, USAGE_GROUP_TOKENS, étape E2) : le bilan
  // d'équipement se lit avec la même convention de couleur sur les deux écrans.
  equipment: 'frag-shoulder',
  dropped: 'frag-melee', // rose — lâchés
  pads: 'frag-grenade', // ambre — libre ici (les grenades sont hors contrat)
  other: 'frag-unattributed', // gris — grandeur non cataloguée
}

/**
 * L'encre des trois rôles d'objectif (colonnes des grilles du bloc 3). Gamme
 * `chart-series-*` : la couleur IDENTIFIE un rôle, elle ne juge rien — une gamme
 * ordinale (perf-tier) ou de statut mentirait.
 */
export const ROLE_TOKENS: Record<string, SemanticToken> = {
  take: 'chart-series-1',
  defend: 'chart-series-2',
  hold: 'chart-series-3',
}

/** Le jeton d'un rôle, avec repli neutre pour un rôle non catalogué. */
export function roleToken(role: string): SemanticToken {
  return ROLE_TOKENS[role] ?? 'chart-series-4'
}

/**
 * Les grandeurs du bloc ÉQUIPEMENT, dans l'ordre canonique.
 *
 * TROIS EXCLUSIONS (étape E4) :
 *   - `pad_pickups` (armes spéciales, sa propre carte) — inchangé ;
 *   - `dropped_objects` (E4.5) — « une mort n'est pas un geste, elle est devenue un
 *     segment » : le total des lâchers vit désormais DANS la pile de chaque famille
 *     du bilan (`equipment_<famille>.outcomes.dropped`), une ligne à part le
 *     compterait deux fois ;
 *   - toute grandeur SUPPLANTÉE par son équivalent `equipment_<famille>` de LA MÊME
 *     SESSION : `deployed_<famille>`, `camo_episodes`, `overshield_episodes` comptent
 *     des GESTES (poses, épisodes), quand `equipment_<famille>` compte des OBJETS
 *     (les trois issues, décision P1) — deux lignes pour une même famille
 *     dupliqueraient la grandeur. Sans équivalent (grappin, propulseur — la table
 *     `equipmentOutcomeStems` côté Go ne les nomme pas, cf. §6 du plan), la grandeur
 *     GESTE reste seule, rendu STRICTEMENT INCHANGÉ (E1/E4.1 : absent de bilan =
 *     comme avant).
 */
export function equipmentMetrics(
  metrics: SessionUsageMetric[] | null | undefined,
): SessionUsageMetric[] {
  const all = metrics ?? []
  const bilanFamilies = new Set(
    all
      .filter((m) => metricKind(m.key) === 'equipment')
      .map((m) => m.key.slice(EQUIPMENT_PREFIX.length)),
  )
  return all
    .filter((m) => metricKind(m.key) !== 'pads')
    .filter((m) => metricKind(m.key) !== 'dropped')
    .filter((m) => {
      const family = equipmentBilanFamilyOf(m.key)
      return family == null || !bilanFamilies.has(family)
    })
    .sort((a, b) => {
      const ra = METRIC_RANK[metricKind(a.key)] ?? 9
      const rb = METRIC_RANK[metricKind(b.key)] ?? 9
      return ra !== rb ? ra - rb : a.key.localeCompare(b.key)
    })
}

/** Le TOTAL des ramassages d'armes spéciales (clé `pad_pickups` du contrat). */
export function padMetric(
  metrics: SessionUsageMetric[] | null | undefined,
): SessionUsageMetric | null {
  return (metrics ?? []).find((m) => metricKind(m.key) === 'pads') ?? null
}

// ─── Parités ─────────────────────────────────────────────────────────────────────

/**
 * La parité du dénominateur « mon camp / lobby » : la part attendue d'un camp moyen,
 * soit 100 × effectif d'équipe / effectif du lobby. Non publiée telle quelle par le
 * contrat (qui publie les deux parités de JOUEUR) — dérivée des deux effectifs moyens.
 */
export function teamOfLobbyParityPct(
  teamSizeAvg: number | null | undefined,
  lobbySizeAvg: number | null | undefined,
): number | null {
  if (teamSizeAvg == null || lobbySizeAvg == null || lobbySizeAvg <= 0) return null
  return (teamSizeAvg / lobbySizeAvg) * 100
}

// ─── Forme « écart à la parité » ─────────────────────────────────────────────────

/**
 * Une jauge 0..100 % : sa valeur, son trait de parité, ses deux textes.
 *
 * PLUS D'ÉTENDUE MIN/MAX (revue de lisibilité 2026-09-09, D3). Le rail portait un second
 * trait fin, sans légende, disant l'écart entre le meilleur et le pire match — la BANDE DE
 * RÉGULARITÉ de la même carte dit exactement cela, case par match, et se lit. Deux formes
 * pour une information, dont une muette : celle qui ne parle pas a été retirée du modèle,
 * pas seulement cachée du rendu (règle « 0 code mort »).
 *
 * `honestyText` (le compte brut) reste dans le MODÈLE mais ne sort plus en cellule : il
 * n'apparaît que dans `tooltip`. Un pourcentage doublé de sa fraction dans une colonne de
 * 3 lignes triple la charge de lecture sans rien ajouter (D2).
 */
export interface UsageGaugeModel {
  key: string
  valuePct: number | null
  parityPct: number | null
  valueText: string
  honestyText: string
  tooltip: string
  /**
   * LA PILE DES TROIS ISSUES (P1, P6, étape E4) — remplit la TRANCHE (0..`valuePct`),
   * jamais le reste du rail. Absent = rendu STRICTEMENT INCHANGÉ, un seul aplat
   * (même contrat que `ValueGridCell.segments`, E1) : toute grandeur sans
   * `SessionUsageOutcomes` (armes spéciales, objectifs, familles hors bilan) n'a
   * jamais porté cette clé et continue de rendre exactement comme avant.
   */
  segments?: UsageGaugeOutcomeSegment[]
  /**
   * LES DEUX REPÈRES DE TAUX DANS LA TRANCHE (P7, décision produit du §3.2) : des
   * MARQUES, jamais un chiffre affiché (E4.2) — le texte vit dans `tooltip`. Chacun
   * est un pourcentage RELATIF À LA TRANCHE (même dénominateur que `segments`,
   * PAS le rail entier) : `null` quand la référence n'a pas de scope à camp connu
   * (P7 exclut le joueur — session FFA, aucune référence d'équipe ne se calcule),
   * jamais un 0 % inventé.
   */
  teammatesRatePct: number | null
  opponentsRatePct: number | null
}

/** L'ordre canonique des trois issues (P1, ratifié par le test E4.6). */
const OUTCOME_ORDER = ['used', 'dropped', 'kept'] as const
export type UsageOutcomeKind = (typeof OUTCOME_ORDER)[number]

/** Le jeton d'une issue — table normative §3.1 du plan. */
const OUTCOME_TOKENS: Record<UsageOutcomeKind, SemanticToken> = {
  used: 'divergent-pos',
  dropped: 'divergent-neg',
  kept: 'divergent-neutral',
}

function outcomeKindLabel(kind: UsageOutcomeKind, t: UsageText): string {
  switch (kind) {
    case 'used':
      return t.outcomeUsed
    case 'dropped':
      return t.outcomeDropped
    case 'kept':
      return t.outcomeKept
  }
}

export interface UsageGaugeOutcomeSegment {
  key: UsageOutcomeKind
  /** Fraction 0..1 DE LA TRANCHE (used+kept+dropped = 100 %), jamais du rail entier. */
  fraction: number
  token: SemanticToken
  label: string
}

/**
 * buildOutcomeSegments — la pile utilisé → lâché → gardé (ordre P1/E4.6), UNIQUEMENT
 * les issues NON NULLES (« une famille sans troisième issue rend deux segments »,
 * E4.6) — un segment à fraction 0 ne se dessine pas. `outcomes` absent, ou dont la
 * somme des trois issues est nulle (aucun objet mesuré) : PAS de pile — le rendu
 * simple (un seul aplat) reste la vérité, jamais une pile à une seule couleur.
 */
function buildOutcomeSegments(
  outcomes: SessionUsageOutcomes | null | undefined,
  t: UsageText,
): UsageGaugeOutcomeSegment[] | undefined {
  if (outcomes == null) return undefined
  const total = outcomes.used + outcomes.kept + outcomes.dropped
  if (total <= 0) return undefined
  const segments = OUTCOME_ORDER.filter((kind) => outcomes[kind] > 0).map((kind) => ({
    key: kind,
    fraction: outcomes[kind] / total,
    token: OUTCOME_TOKENS[kind],
    label: outcomeKindLabel(kind, t),
  }))
  return segments
}

/** Une ligne de jauges : la grandeur, et ses trois dénominateurs (§7 du handoff). */
export interface UsageGaugeRowModel {
  key: string
  label: string
  gauges: UsageGaugeModel[]
  /**
   * Cette ligne est le TOTAL des lignes qui la précèdent (D6) — la grille la sépare d'un
   * filet et la pose en teinte secondaire. Le seul cas aujourd'hui : « toutes armes
   * spéciales », qui est exactement la somme de ses familles ; affichée comme leur égale,
   * elle se lisait comme une grandeur de plus.
   */
  isTotal?: boolean
}

/** Le sous-ensemble « parts » commun aux grandeurs, familles d'arme et rôles. */
export interface UsageSharesLike {
  player_total: number
  team_total?: number
  lobby_total: number
  team_share_of_lobby_pct?: number
  player_share_of_team_pct?: number
  player_share_of_lobby_pct?: number
}

export interface UsageGaugeRowInput {
  key: string
  label: string
  shares: UsageSharesLike
  isDuration?: boolean
  teamParityPct: number | null | undefined
  lobbyParityPct: number | null | undefined
  teamOfLobbyParityPct: number | null | undefined
  /** Cette grandeur est le total des lignes qui la précèdent (cf. `UsageGaugeRowModel`). */
  isTotal?: boolean
  /**
   * Les trois issues DU JOUEUR et les deux repères de taux (étape E4, contrat étendu
   * en E3). Absent pour toute grandeur hors bilan d'équipement (armes spéciales,
   * objectifs) — jamais posé à zéro.
   */
  outcomes?: SessionUsageOutcomes | null
  t: UsageText
  locale: Locale
}

/**
 * buildGaugeRow — les trois jauges d'une grandeur : mon équipe dans le lobby, ma part
 * dans mon équipe, ma part dans le lobby. Chaque jauge porte sa part (l'axe) et son
 * compte brut (l'honnêteté, en infobulle seulement — D2). Une part absente du contrat
 * rend une jauge VIDE au tiret — jamais un 0 %.
 *
 * L'ORDRE DES JAUGES EST UN CONTRAT DE RENDU : la 2e (`player-of-team`) est celle que la
 * grille montre seule quand le repli est fermé — `PRIMARY_GAUGE_INDEX`, SessionUsageForms.
 *
 * `outcomes` NE S'APPLIQUE QU'AUX DEUX JAUGES DE PART DU JOUEUR (`player-of-team`,
 * `player-of-lobby`) — jamais à `team-of-lobby` : les trois issues sont une grandeur
 * DU JOUEUR (domain.SessionUsageOutcomes ne porte que « mine »), quand la première
 * jauge mesure la part de MON ÉQUIPE dans le lobby, une population entière sans
 * porteur individuel.
 */
export function buildGaugeRow(input: UsageGaugeRowInput): UsageGaugeRowModel {
  const { shares, t, locale } = input
  const isDur = input.isDuration === true
  const count = (v: number | null | undefined) => formatUsageCount(v, locale, isDur)
  const segments = buildOutcomeSegments(input.outcomes, t)
  const teammatesRatePct = input.outcomes?.teammates_used_rate_pct ?? null
  const opponentsRatePct = input.outcomes?.opponents_used_rate_pct ?? null

  const gauge = (
    key: string,
    gaugeLabel: string,
    valuePct: number | null | undefined,
    parityPct: number | null | undefined,
    numerator: number | null | undefined,
    denominator: number | null | undefined,
    withOutcomes: boolean,
  ): UsageGaugeModel => {
    const valueText = formatUsagePct(valuePct, locale)
    const honestyText = t.honestyFmt(count(numerator), count(denominator))
    let tooltip = t.gaugeTipFmt(input.label, gaugeLabel, valueText, honestyText)
    const gaugeSegments = withOutcomes ? segments : undefined
    if (gaugeSegments != null && input.outcomes != null) {
      tooltip = t.gaugeOutcomeTipFmt(
        tooltip,
        count(input.outcomes.used),
        count(input.outcomes.kept),
        count(input.outcomes.dropped),
      )
    }
    const gaugeTeammatesRatePct = withOutcomes ? teammatesRatePct : null
    const gaugeOpponentsRatePct = withOutcomes ? opponentsRatePct : null
    if (gaugeTeammatesRatePct != null || gaugeOpponentsRatePct != null) {
      tooltip = t.gaugeReferenceTipFmt(
        tooltip,
        formatUsagePct(gaugeTeammatesRatePct, locale),
        formatUsagePct(gaugeOpponentsRatePct, locale),
      )
    }
    return {
      key,
      valuePct: valuePct ?? null,
      parityPct: parityPct ?? null,
      valueText,
      honestyText,
      tooltip,
      segments: gaugeSegments,
      teammatesRatePct: gaugeTeammatesRatePct,
      opponentsRatePct: gaugeOpponentsRatePct,
    }
  }

  return {
    key: input.key,
    label: input.label,
    isTotal: input.isTotal,
    gauges: [
      gauge(
        'team-of-lobby',
        t.gaugeTeamOfLobby,
        shares.team_share_of_lobby_pct,
        input.teamOfLobbyParityPct,
        shares.team_total,
        shares.lobby_total,
        false,
      ),
      gauge(
        'player-of-team',
        t.gaugePlayerOfTeam,
        shares.player_share_of_team_pct,
        input.teamParityPct,
        shares.player_total,
        shares.team_total,
        true,
      ),
      gauge(
        'player-of-lobby',
        t.gaugePlayerOfLobby,
        shares.player_share_of_lobby_pct,
        input.lobbyParityPct,
        shares.player_total,
        shares.lobby_total,
        true,
      ),
    ],
  }
}

// ─── Forme « piste du lobby » ────────────────────────────────────────────────────

/** Un segment de la piste : coloré = nous (découpé par joueur), hachuré = eux. */
export interface UsageTrackSegment {
  key: string
  label: string
  kind: 'me' | 'squad' | 'team-rest' | 'enemy'
  /** Index 0..2 dans SquadPlayers — décide le jeton squad-player-1..3. */
  squadIndex?: number
  count: number
  pctText: string
  tooltip: string
}

export interface UsageTrackInput {
  meLabel: string
  shares: UsageSharesLike
  squadPlayers: SessionUsageSquadPlayer[]
  squadShares: SessionUsageSquadShare[] | null | undefined
  t: UsageText
  locale: Locale
}

/**
 * buildLobbyTrack — la piste 100 % du lobby : moi, chaque coéquipier suivi, le reste
 * de mon équipe (colorés), et EUX en un seul segment hachuré, anonyme — ni nom, ni
 * couleur d'équipe (doctrine §1 : la référence n'est jamais affichée).
 *
 * `team_total` absent (aucun match du scope à camp connu) → PAS de piste : sans la
 * frontière nous/eux, le découpage mentirait. Les résidus (reste d'équipe, camp
 * adverse) sont bornés à zéro : les scopes joueur (tout le scope) et équipe (matchs
 * à camp connu) peuvent différer d'une poignée d'unités sur une session mixte.
 */
export function buildLobbyTrack(input: UsageTrackInput): UsageTrackSegment[] | null {
  const { shares, t, locale } = input
  if (shares.team_total == null || shares.lobby_total <= 0) return null

  const bySquadXuid = new Map((input.squadShares ?? []).map((s) => [s.xuid, s.total]))
  const squadCounts = input.squadPlayers.map((p) => bySquadXuid.get(p.xuid) ?? 0)
  const squadSum = squadCounts.reduce((a, b) => a + b, 0)
  const teamRest = Math.max(0, shares.team_total - shares.player_total - squadSum)
  const enemy = Math.max(0, shares.lobby_total - shares.team_total)

  const raw: Array<Omit<UsageTrackSegment, 'pctText' | 'tooltip'>> = [
    { key: 'me', label: input.meLabel, kind: 'me', count: shares.player_total },
    ...input.squadPlayers.map((p, i) => ({
      key: `squad-${p.xuid}`,
      label: p.gamertag,
      kind: 'squad' as const,
      squadIndex: i,
      count: squadCounts[i],
    })),
    { key: 'team-rest', label: t.segTeamRest, kind: 'team-rest', count: teamRest },
    { key: 'enemy', label: t.segEnemy, kind: 'enemy', count: enemy },
  ]
  const total = raw.reduce((a, s) => a + s.count, 0)
  if (total <= 0) return null

  return raw
    .filter((s) => s.count > 0)
    .map((s) => {
      const pctText = formatUsagePct((s.count / total) * 100, locale)
      return {
        ...s,
        pctText,
        tooltip: t.trackTipFmt(s.label, formatUsageCount(s.count, locale), pctText),
      }
    })
}

// ─── Forme « bande de régularité » ───────────────────────────────────────────────

export type UsageBandTone = 'above' | 'near' | 'below' | 'unmeasured'

export interface UsageBandCell {
  matchId: string
  tone: UsageBandTone
  tooltip: string
}

/** Sous ±ε points de la parité, une case est « à la parité » (ni bonus ni déficit). */
const BAND_EPSILON_PT = 1

/**
 * buildRegularityBand — une case par match MESURÉ, teintée par l'écart de la part
 * d'équipe du joueur à la PARITÉ DE SESSION (le contrat ne publie pas la parité de
 * chaque match ; les comptes « au-dessus de la parité », eux, sont calculés côté Go
 * contre la parité de CHAQUE match — la légende du bloc les écrit tels quels).
 * Part absente sur un match (camp inconnu, dénominateur nul) → case « non mesurée ».
 */
export function buildRegularityBand(
  perMatch: SessionUsageMatchPoint[] | null | undefined,
  teamParityPct: number | null | undefined,
  t: UsageText,
  locale: Locale,
): UsageBandCell[] {
  return (perMatch ?? []).map((p, i) => {
    const share = p.player_share_of_team_pct
    if (share == null || teamParityPct == null) {
      return { matchId: p.match_id, tone: 'unmeasured', tooltip: t.bandTipUnmeasured(i + 1) }
    }
    const delta = share - teamParityPct
    const tone: UsageBandTone =
      delta > BAND_EPSILON_PT ? 'above' : delta < -BAND_EPSILON_PT ? 'below' : 'near'
    return {
      matchId: p.match_id,
      tone,
      tooltip: t.bandTipFmt(
        i + 1,
        formatUsagePct(share, locale),
        formatUsagePct(teamParityPct, locale),
      ),
    }
  })
}

// ─── Objectifs : rôles, familles, escouade ───────────────────────────────────────
// Les GRILLES ALIGNÉES (cadences, familles d'objectif, escouade) vivent dans
// `usageGrids.ts` — même frontière que valueGridModel : ce fichier-ci porte les
// formes propres (jauges, piste, bande), l'autre les projections ValueGrid.

export const ROLE_ORDER = ['take', 'defend', 'hold'] as const
// Les libellés de rôle, de famille de mode et de bonus vivent avec le dictionnaire
// (`usageI18n.ts` : roleLabel, familyLabel, powerupLabel) — ce fichier ne garde que
// l'ordre canonique, dont le tri a besoin.

/** Tri des rôles dans l'ordre canonique prendre / défendre / tenir. */
export function sortRoles(roles: SessionObjectiveRoleMetric[] | null | undefined) {
  const rank = (r: string) => {
    const i = (ROLE_ORDER as readonly string[]).indexOf(r)
    return i === -1 ? ROLE_ORDER.length : i
  }
  return [...(roles ?? [])].sort((a, b) => rank(a.role) - rank(b.role))
}

// ─── Disponibilité du bloc ───────────────────────────────────────────────────────

export type UsageAvailability =
  | { kind: 'ok' }
  | { kind: 'hidden' }
  | { kind: 'empty'; message: string }

/**
 * usageAvailability — l'état du bloc, et la distinction entre les DEUX raisons de ne
 * rien avoir (règle des deux portes, 2026-09-05, registre L4) :
 *
 *   - absent du payload (vieux serveur) → RIEN, pas de bloc fantôme ;
 *   - `unsupported` → RIEN NON PLUS : le TITRE ne publie pas de résumé d'usage (pas de
 *     décodeur de film, donc pas d'artefact, donc rien à résumer — jamais). Une carte
 *     « Ce titre ne publie pas de résumé d'usage des films » était un bloc mort : elle
 *     occupait une place, ne disait rien d'actionnable, et ne disparaîtrait jamais ;
 *   - `load_failed` → état vide AVEC la raison : le titre sait le produire, c'est CETTE
 *     lecture-là qui a échoué. Transitoire, donc il faut le dire ;
 *   - aucun match mesuré → état vide « aucun film » (les objectifs, au scope indépendant
 *     des films, restent affichables par l'appelant).
 */
export function usageAvailability(
  usage: SessionUsageBlock | null | undefined,
  t: UsageText,
): UsageAvailability {
  if (usage == null) return { kind: 'hidden' }
  if (!usage.available) {
    if (usage.unavailable_reason === 'unsupported') return { kind: 'hidden' }
    return { kind: 'empty', message: t.unavailableLoadFailed }
  }
  if (usage.matches_measured <= 0) return { kind: 'empty', message: t.unavailableNoMeasured }
  return { kind: 'ok' }
}
