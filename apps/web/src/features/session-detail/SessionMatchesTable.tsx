/**
 * SessionMatchesTable — RÉUTILISE le tableau de résultats de l'Explorer
 * (ExplorerMatchesTable) avec les matchs de la session, via un adapter de lignes
 * `SessionDetailMatchRow → ExplorerMatchRow`. Colonnes / en-têtes / rendu / pastille
 * Solo-Escouade strictement identiques à l'Explorer (pas de duplication de colonnes).
 *
 * Navigation : on conserve le contexte "session" (prev/next dans la session) via
 * `contextDescriptor` + `filterSpecOverride` exposés par ExplorerMatchesTable.
 *
 * variant="compact" (drawer compare, 50%) → pagination compacte (10 lignes +
 * "Voir tout") ET set de colonnes RÉDUIT (COMPACT_HIDDEN_COLUMNS) pour une
 * comparaison côte-à-côte lisible dans les panneaux étroits : on ne garde que
 * Issue · Mode · K · D · A · KDA · Perf · Rang(=rating) · Δ rang · ΔMMR. Même
 * composant et même style — colonnes masquées via `columnVisibility`.
 *
 * Colonne « Δ rang » : INJECTÉE par cette vue (extraColumns) car ExplorerMatchesTable
 * n'en a pas — elle ne doit PAS apparaître sur la page Explorer.
 */
import { useMemo } from 'react'
import { type ColumnDef } from '@tanstack/react-table'

import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import type { ExplorerMatchRow, SessionDetailMatchRow } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'

import { formatRankDelta, rankDeltaToken, useSessionT } from './_shared'

/**
 * Colonnes masquées en mode comparaison (drawer 50%). Clés = ids de colonnes
 * ExplorerMatchesTable (accessorKey, ou 'open'/'waypoint'). On retire le
 * contextuel/large (ouvrir, lien Waypoint, date, carte, playlist, solo/escouade,
 * dominance, score, durée, ΔPerf, type de rating, MMR équipe/adverse) pour ne
 * garder que l'essentiel comparable.
 * Δ rang (ratingDelta) n'existe pas comme colonne Explorer → non affichable sans
 * modifier le composant ; le rang/rating courant reste visible (skill_tier_label).
 */
const COMPACT_HIDDEN_COLUMNS: Record<string, boolean> = {
  open: false,
  waypoint: false,
  start_time: false,
  map_ui: false,
  playlist_label: false,
  is_with_friends: false,
  dominance_flag: false,
  score_label: false,
  duration_seconds: false,
  delta_perf: false,
  rating_type: false,
  team_mmr: false,
  enemy_mmr: false,
}

interface Props {
  matches: SessionDetailMatchRow[]
  playerSlug: string
  variant?: 'full' | 'compact'
  /** Session solo ou escouade (uniforme sur tous ses matchs) → colonne Solo/Escouade. */
  withFriends?: boolean
}

/**
 * Adapte une ligne de session vers le format ExplorerMatchRow consommé par
 * ExplorerMatchesTable. Exporté pour test. Les champs absents côté session
 * (delta_perf, dominance) restent nuls → l'Explorer affiche "-" gracieusement.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function toExplorerRow(m: SessionDetailMatchRow, withFriends: boolean, locale: ManifestLocale): ExplorerMatchRow {
  const ratingType = m.skill_rating_type ? m.skill_rating_type.toUpperCase() : null
  return {
    match_id: m.match_id,
    start_time: m.start_time,
    start_time_label: '',
    map_ui: m.map_name ?? '',
    mode_ui: m.mode_ui || m.pair_name || '',
    playlist_label: m.playlist_name ?? '',
    outcome_label: '',
    outcome_code: m.outcome ?? 4,
    score_label: m.personal_score != null ? m.personal_score.toLocaleString(intlLocale(locale)) : '',
    is_with_friends: withFriends,
    experience_type_label: '',
    kills: m.kills,
    deaths: m.deaths,
    assists: m.assists,
    kda: m.kda ?? null,
    perf_score: m.performance_score ?? null,
    perf_tier: m.perf_tier,
    delta_perf: null,
    rating_type: ratingType,
    // Colonne "Rang" = libellé du palier ("Or III"…) comme l'Explorer, fourni par le
    // backend (analysis.BuildSkillTierLabel). Plus la valeur brute.
    skill_tier_label: m.skill_tier_label ?? null,
    skill_rating_delta: m.skill_rating_delta ?? null,
    expected_win_prob: m.expected_win_prob ?? null,
    // Placement X/Y (phase de placement) : l'Explorer l'affiche dans la colonne Rang
    // à la place du palier quand les deux sont présents.
    placement_done: m.placement_done ?? null,
    placement_total: m.placement_total ?? null,
    team_mmr: m.team_mmr ?? null,
    enemy_mmr: m.enemy_mmr ?? null,
    delta_mmr: m.delta_mmr ?? null,
    duration_seconds: m.duration_seconds ?? null,
    dominance_flag: 0,
  }
}

export function SessionMatchesTable({ matches, playerSlug, variant = 'full', withFriends = false }: Props) {
  const t = useSessionT()
  const locale = useAppShellStore((s) => s.locale)
  // Tri chronologique ASC (oldest first) — cohérent avec les charts de session
  // au-dessus. start_time ISO UTC → comparaison lexicale = ordre chrono.
  const rows = useMemo(
    () =>
      [...matches]
        .sort((a, b) => a.start_time.localeCompare(b.start_time))
        .map((m) => toExplorerRow(m, withFriends, locale)),
    [matches, withFriends, locale],
  )

  // Colonne « Δ rang » (gain/perte de rating du match) INJECTÉE après la colonne
  // Rang — spécifique à la vue session, jamais exposée sur la page Explorer.
  // CSR : entier signé ; LUSR : 2 décimales ; couleur divergente (cf. _shared).
  const deltaRatingLabel = t('session.detail.col_rating_delta')
  const extraColumns = useMemo<ColumnDef<ExplorerMatchRow>[]>(
    () => [
      {
        id: 'delta_rating',
        header: deltaRatingLabel,
        cell: (ctx) => {
          const d = ctx.row.original.skill_rating_delta
          if (d == null) return '-'
          return (
            <span className="font-mono tabular-nums" style={{ color: tokenCssVar(rankDeltaToken(d)) }}>
              {formatRankDelta(d, ctx.row.original.rating_type ?? '')}
            </span>
          )
        },
      },
    ],
    [deltaRatingLabel],
  )

  // Début de session = match le plus ancien (start_time ISO UTC → tri lexical = chrono).
  const sessionStartUtc = useMemo(
    () => matches.reduce((min, m) => (min === '' || m.start_time < min ? m.start_time : min), ''),
    [matches],
  )
  const sessionId = useMemo(
    () => matches.find((m) => m.session_label)?.session_label ?? undefined,
    [matches],
  )

  return (
    <ExplorerMatchesTable
      rows={rows}
      playerSlug={playerSlug}
      contextDescriptor={{ kind: 'session', startTimeUtc: sessionStartUtc }}
      filterSpecOverride={sessionId ? { session_id: sessionId } : undefined}
      defaultPageSize={variant === 'compact' ? 10 : undefined}
      columnVisibility={variant === 'compact' ? COMPACT_HIDDEN_COLUMNS : undefined}
      extraColumns={extraColumns}
      extraColumnsAfterId="skill_tier_label"
    />
  )
}
