/**
 * ExplorerBriefingTiles — tuiles KPI composées du socle du briefing Explorer.
 *
 * Tuiles réutilisant BriefingTile : Taux de victoire (hero — OutcomeBar + V-D-N +
 * tooltip des 4 issues, DEC-TILES), Durée totale (DEC-DURATION), Pic rang / Pic MMR
 * (DEC-PEAK) et Séries marquantes (bicolore). Expose aussi MinMaxTriptych (DP-1) :
 * l'affichage « min · moyenne · max » des tuiles FDA & Perf (rendues inline dans le
 * Strip). Le Pic FDA autonome a été fusionné dans le triptyque FDA (max = peak_kda).
 * Le Classement a quitté le socle pour la rangée « Par… » (ExplorerRankedBlock).
 * Chaque tuile porte un accent (DP-6 ; neutre = outcome-draw). Tokens sémantiques
 * uniquement (aucune couleur hex) ; les sentiments colorés via les helpers d'accès.
 */
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { formatDurationHM, formatPercentInt } from '@/lib/formatters'
import { useOutcomeLabel } from '@/lib/i18n/fieldMappings'
import type {
  ExplorerBriefingBaseline,
  ExplorerBriefingPeakRank,
  ExplorerBriefingScope,
  ExplorerBriefingStreaks,
} from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingTile } from './BriefingTile'
import { formatSignedPoints } from '@/lib/baseline'
import { deltaToken } from './ExplorerBriefing.logic'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

// MinMaxTriptych (DP-1/DEC-TRIPTYCH) : affichage compact « min · moyenne · max ». La
// moyenne au centre en grand (hérite text-xl du conteneur BriefingTile), colorée via
// midColor ; les bornes min/max lisibles (contraste plein via text-foreground, poids
// normal, text-xs) — hiérarchie : moyenne colorée dominante >> extrêmes neutres. Chaque
// borne nulle est OMISE (pas de « — » parasite) ; moyenne nulle → « — ». Deux
// consommateurs (tuiles FDA & Perf du Strip) → un seul composant (CLAUDE.md §6).
export function MinMaxTriptych({
  min,
  mid,
  max,
  midColor,
  format,
}: {
  min: number | null | undefined
  mid: number | null | undefined
  max: number | null | undefined
  midColor?: string
  format: (v: number) => string
}) {
  return (
    <span className="inline-flex items-baseline justify-center gap-1.5">
      {min != null && (
        <span className="text-xs font-normal tabular-nums text-foreground">{format(min)}</span>
      )}
      <span style={midColor != null ? { color: midColor } : undefined}>
        {mid != null ? format(mid) : '—'}
      </span>
      {max != null && (
        <span className="text-xs font-normal tabular-nums text-foreground">{format(max)}</span>
      )}
    </span>
  )
}

// Tuile Séries marquantes (DP-9) : valeur bicolore « {best} V / {worst} D » (tokens
// outcome-win/outcome-loss). Segment à zéro omis ; la tuile elle-même est omise
// par l'appelant (Strip) quand les deux segments sont à zéro. Accent neutre (DP-6).
export function StreaksTile({ streaks, t }: { streaks: ExplorerBriefingStreaks; t: T }) {
  const best = streaks.best_win_streak ?? 0
  const worst = streaks.worst_loss_streak ?? 0
  return (
    <BriefingTile
      label={t('explorer.briefing.streaks_title')}
      accent="outcome-draw"
      info={<InfoTooltip content={t('explorer.briefing.tip_streaks')} iconClass="w-3.5 h-3.5" />}
      value={
        <span className="inline-flex items-baseline gap-1">
          {best > 0 && (
            <span style={{ color: tokenCssVar('outcome-win') }}>
              {t('explorer.briefing.streak_wins', { n: best })}
            </span>
          )}
          {best > 0 && worst > 0 && <span className="text-muted-foreground">/</span>}
          {worst > 0 && (
            <span style={{ color: tokenCssVar('outcome-loss') }}>
              {t('explorer.briefing.streak_losses', { n: worst })}
            </span>
          )}
        </span>
      }
    />
  )
}

// ─── Tuile Taux de victoire (hero) — DEC-TILES ────────────────────────────────
// Contenu repris de HomeHeroKPIGrid : valeur NEUTRE (le sentiment est porté par
// l'accent 3px, bande neutre 0.45-0.55), ruban OutcomeBar flanqué des victoires
// (gauche) et défaites (droite), détail des 4 issues au survol du ruban. Sous-texte :
// delta « vs habituel » (masqué en plein historique). Remplace la sparkline (DEC-SPARK).
export function WinRateTile({
  scope,
  baseline,
  fullHistory,
  t,
}: {
  scope: ExplorerBriefingScope
  baseline: ExplorerBriefingBaseline | null | undefined
  fullHistory: boolean
  t: T
}) {
  const wr = scope.win_rate ?? null
  const { wins, losses, ties: draws, dnf: dnfs } = scope
  // Libellés canoniques des issues (field-mappings) — détail affiché au survol.
  const winLabel = useOutcomeLabel('win')
  const drawLabel = useOutcomeLabel('tie')
  const dnfLabel = useOutcomeLabel('dnf')
  const lossLabel = useOutcomeLabel('loss')
  const accent: SemanticToken =
    wr != null && wr > 0.55
      ? 'outcome-win'
      : wr != null && wr < 0.45
        ? 'outcome-loss'
        : 'outcome-draw'
  const outcomeRows: { token: SemanticToken; label: string; value: number }[] = [
    { token: 'outcome-win', label: winLabel, value: wins },
    { token: 'outcome-draw', label: drawLabel, value: draws },
    { token: 'outcome-dnf', label: dnfLabel, value: dnfs },
    { token: 'outcome-loss', label: lossLabel, value: losses },
  ]
  const outcomeTooltip = (
    <div className="space-y-1 text-left">
      {outcomeRows
        .filter((o) => o.value > 0)
        .map((o) => (
          <div key={o.label} className="flex items-center gap-2">
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ backgroundColor: tokenCssVar(o.token) }}
            />
            <span className="flex-1 whitespace-nowrap">{o.label}</span>
            <span className="font-semibold tabular-nums">{o.value}</span>
          </div>
        ))}
    </div>
  )
  return (
    <BriefingTile
      label={t('explorer.briefing.win_rate_label')}
      info={<InfoTooltip content={t('explorer.briefing.tip_win_rate')} iconClass="w-3.5 h-3.5" />}
      accent={accent}
      value={
        <div>
          <span>{formatPercentInt(wr)}</span>
          <div className="mt-1 flex items-center gap-2">
            <span
              className="shrink-0 text-xs font-semibold tabular-nums"
              style={{ color: tokenCssVar('outcome-win') }}
            >
              {wins}
            </span>
            <Tooltip className="min-w-0 flex-1" content={outcomeTooltip}>
              <OutcomeBar wins={wins} draws={draws} losses={losses} dnfs={dnfs} />
            </Tooltip>
            <span
              className="shrink-0 text-xs font-semibold tabular-nums"
              style={{ color: tokenCssVar('outcome-loss') }}
            >
              {losses}
            </span>
          </div>
        </div>
      }
      sub={
        baseline && !fullHistory ? (
          <>
            <span
              className="font-semibold"
              style={{ color: tokenCssVar(deltaToken(baseline.delta_win_rate)) }}
            >
              {formatSignedPoints(baseline.delta_win_rate)}
            </span>{' '}
            {t('explorer.briefing.vs_baseline')}
          </>
        ) : undefined
      }
    />
  )
}

// ─── Tuile Durée totale — DEC-DURATION ────────────────────────────────────────
// Somme serveur des durées du scope, format « h min » (jamais MM:SS). Descriptive,
// accent neutre (DP-6).
export function DurationTile({ seconds, t }: { seconds: number | null | undefined; t: T }) {
  return (
    <BriefingTile
      label={t('explorer.briefing.duration_total_label')}
      accent="outcome-draw"
      info={<InfoTooltip content={t('explorer.briefing.tip_duration')} iconClass="w-3.5 h-3.5" />}
      value={formatDurationHM(seconds)}
    />
  )
}

// ─── Tuile Pic rang — DEC-PEAKRANK ────────────────────────────────────────────
// Meilleur palier ATTEINT par système de rating sur le scope (jusqu'à 2 lignes
// LUSR/CSR). Valeur = 1er système « {TYPE} {palier} », sous-ligne = 2e système.
// Type de rating en capitales (classe uppercase, comme le Classement).
export function PeakRankTile({ ranks, t }: { ranks: ExplorerBriefingPeakRank[]; t: T }) {
  if (ranks.length === 0) return null
  const [first, second] = ranks
  return (
    <BriefingTile
      label={t('explorer.briefing.peak_rank_label')}
      accent="outcome-draw"
      info={<InfoTooltip content={t('explorer.briefing.tip_peak_rank')} iconClass="w-3.5 h-3.5" />}
      value={
        <span className="text-sm">
          <span className="uppercase text-muted-foreground">{first.rating_type}</span>{' '}
          {first.tier_label}
        </span>
      }
      sub={
        second != null ? (
          <span>
            <span className="uppercase">{second.rating_type}</span> {second.tier_label}
          </span>
        ) : undefined
      }
    />
  )
}

// ─── Tuile Pic MMR — DEC-PEAKRANK (priorité la plus basse) ─────────────────────
// Meilleur team_mmr du scope, valeur brute arrondie (non colorée, masquage MMR).
export function PeakMmrTile({ value, t }: { value: number; t: T }) {
  return (
    <BriefingTile
      label={t('explorer.briefing.peak_mmr_label')}
      accent="outcome-draw"
      info={<InfoTooltip content={t('explorer.briefing.tip_peak_mmr')} iconClass="w-3.5 h-3.5" />}
      value={String(Math.round(value))}
    />
  )
}
