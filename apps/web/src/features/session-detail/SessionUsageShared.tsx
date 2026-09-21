/**
 * SessionUsageShared — LE CHROME ET LES PROJECTIONS PARTAGÉS des blocs « usages
 * d'équipement, armes spéciales et objectifs » de la page Sessions.
 *
 * Extrait de `SessionUsageSection.tsx` le 2026-09-13 (scission de taille — CLAUDE.md
 * n°5) au moment où la carte « Usages d'équipement » s'est ouverte en TROIS cartes
 * (cadences / parts / régularité, demande utilisateur) : les deux fichiers appelants
 * ont besoin du même bandeau de titre, des mêmes encres de grille et des mêmes
 * projections de jauges — les recopier aurait fait diverger le chrome (CLAUDE.md n°6).
 *
 * Aucun calcul propre ici : tout vient de `@/features/_shared/usage/`.
 */
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility'
import type { SessionUsageBlock, SessionUsageMetric } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { buildGaugeRow, type UsageGaugeRowModel } from '@/features/_shared/usage/usageGaugeModel'
import { usagePlayerInk, type UsageGridInks } from '@/features/_shared/usage/usageGrids'
import type { UsageText } from '@/features/_shared/usage/usageI18n'
import { USAGE_METRIC_TOKENS, metricKind, metricLabel } from '@/features/_shared/usage/usageMetricKinds'
import { teamOfLobbyParityPct } from '@/features/_shared/usage/usageParity'

/** Les encres partagées des grilles : colonnes par famille/rôle, lignes par joueur. */
export function useGridInks(): UsageGridInks {
  return useMemo(
    () => ({
      columnColor: (key: string) => tokenCssVar(USAGE_METRIC_TOKENS[metricKind(key)]),
      rowAccent: (kind, squadIndex) =>
        kind === 'aggregate' ? undefined : usagePlayerInk(kind, squadIndex),
    }),
    [],
  )
}

export interface CardProps {
  usage: SessionUsageBlock
  meLabel: string
  t: UsageText
  locale: Locale
  /**
   * Colonne divisée (drawer de comparaison ouvert) : les DEUX colonnes passent en
   * compact, jamais une seule — deux rendus différents côte à côte ne se comparent pas.
   *
   * DEPUIS LE 2026-09-13, LE COMPACT NE RETIRE PLUS RIEN (demande utilisateur : « il
   * manque des éléments du drawer »). Il RESSERRE : mêmes cartes, mêmes vues, mêmes
   * colonnes de jauges, avec des rails, des cases et des libellés plus petits. Une
   * forme trop large pour une demi-colonne défile dans son propre conteneur — elle ne
   * disparaît pas.
   */
  compact: boolean
}

/** Les jauges d'une liste de grandeurs, contre les trois parités du bloc. */
export function metricGaugeRows(
  metrics: SessionUsageMetric[],
  usage: SessionUsageBlock,
  t: UsageText,
  locale: Locale,
): UsageGaugeRowModel[] {
  const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
  return metrics.map((m) =>
    buildGaugeRow({
      key: m.key,
      label: metricLabel(m.key, t),
      shares: m,
      // E4.3 : les trois issues et les deux repères de taux, UNIQUEMENT portés par
      // les grandeurs "equipment_<famille>" (contrat étendu en E3) — `undefined`
      // partout ailleurs, jamais posé à zéro (buildGaugeRow ignore un `outcomes` nul).
      outcomes: m.outcomes,
      teamParityPct: usage.team_parity_pct,
      lobbyParityPct: usage.lobby_parity_pct,
      teamOfLobbyParityPct: teamOfLobby,
      t,
      locale,
    }),
  )
}

/**
 * Le comptage d'une bande : les matchs au-dessus de la parité D'ÉQUIPE, NU.
 *
 * SEULEMENT L'ÉQUIPE, et c'est la même parité que celle qui TEINTE les cases
 * (`buildRegularityBand` compare à `team_parity_pct`). Le compte « lobby » qui suivait
 * était vrai mais orphelin : aucune case de la bande ne le représentait (D5).
 *
 * LA PHRASE A DISPARU le 2026-09-21 (retour utilisateur) : « 3/8 au-dessus de la parité »
 * s'écrivait à droite de CHAQUE ligne. Ce que la phrase disait vit maintenant dans
 * `UsageBandLegend`, posée UNE fois sous les bandes ; il ne reste ici que « 3/8 ».
 */
export function bandAboveCaption(m: SessionUsageMetric, measured: number, t: UsageText): string {
  if (m.matches_above_team_parity == null) return t.notMeasured
  return `${m.matches_above_team_parity}/${measured}`
}
