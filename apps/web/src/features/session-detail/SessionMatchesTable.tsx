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
 * "Voir tout") ; les colonnes restent celles de l'Explorer (scroll horizontal).
 */
import { useMemo } from 'react'

import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import type { ExplorerMatchRow, SessionDetailMatchRow } from '@/lib/api/types'

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
export function toExplorerRow(m: SessionDetailMatchRow, withFriends: boolean): ExplorerMatchRow {
  const ratingType = m.skill_rating_type ? m.skill_rating_type.toUpperCase() : null
  // L'Explorer affiche le rang (skill_tier_label) + le type (rating_type) ; on place
  // la VALEUR de rating de la session dans la colonne Rang (CSR entier / LUSR 2 déc.).
  const ratingValueLabel =
    m.skill_rating_value != null
      ? ratingType === 'LUSR'
        ? m.skill_rating_value.toFixed(2)
        : String(Math.round(m.skill_rating_value))
      : null
  return {
    match_id: m.match_id,
    start_time: m.start_time,
    start_time_label: '',
    map_ui: m.map_name ?? '',
    mode_ui: m.mode_ui || m.pair_name || '',
    playlist_label: m.playlist_name ?? '',
    outcome_label: '',
    outcome_code: m.outcome ?? 4,
    score_label: m.personal_score != null ? m.personal_score.toLocaleString('fr-FR') : '',
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
    skill_tier_label: ratingValueLabel,
    placement_done: null,
    placement_total: null,
    team_mmr: m.team_mmr ?? null,
    enemy_mmr: m.enemy_mmr ?? null,
    delta_mmr: m.delta_mmr ?? null,
    duration_seconds: m.duration_seconds ?? null,
    dominance_flag: 0,
  }
}

export function SessionMatchesTable({ matches, playerSlug, variant = 'full', withFriends = false }: Props) {
  const rows = useMemo(() => matches.map((m) => toExplorerRow(m, withFriends)), [matches, withFriends])

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
    />
  )
}
