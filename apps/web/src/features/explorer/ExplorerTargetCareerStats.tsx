/**
 * ExplorerTargetCareerStats — section "Carrière complète" de l'encart Explorer.
 *
 * Stats agrégées carrière entière du joueur cible (KDA / KDR / win rate /
 * accuracy / damage per game / matches) provenant de l'endpoint Halo
 * career-stats. 6 cards KPI sur une ligne + OutcomeBar W/D/L dessous.
 *
 * Affichée seulement quand `careerStats != null`. Le parent gère le masquage
 * et le rendu d'un hint "Connexion Halo requise" en mode no-tokens.
 */
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { NormalizedPlayerStats } from '@/lib/api/types'

interface ExplorerTargetCareerStatsProps {
  careerStats: NormalizedPlayerStats
}

function fmtPct(value: number, locale: string): string {
  // career-stats expose un win_rate en 0..1 ou 0..100 selon les versions.
  // On normalise : si > 1.5, on suppose %, sinon ratio.
  const pct = value > 1.5 ? value : value * 100
  return `${pct.toLocaleString(locale, { maximumFractionDigits: 1 })}%`
}

function fmtNumber(value: number, locale: string, fractionDigits = 2): string {
  return value.toLocaleString(locale, { maximumFractionDigits: fractionDigits })
}

/** formatPlaytime convertit des secondes en "Xj Yh" / "Yh Zm" / "Zm" (FR/EN). */
function formatPlaytime(seconds: number, locale: string): string {
  const totalMin = Math.floor(seconds / 60)
  const days = Math.floor(totalMin / 1440)
  const hours = Math.floor((totalMin % 1440) / 60)
  const mins = totalMin % 60
  const dUnit = locale === 'en' ? 'd' : 'j'
  if (days > 0) return `${days}${dUnit} ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

export function ExplorerTargetCareerStats({ careerStats }: ExplorerTargetCareerStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const numberLocale = intlLocale(appLocale)
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, appLocale, values)

  const totalMatches = careerStats.matches ?? 0

  // Titre « Carrière complète » rendu hors bloc (en-tête de section, cf.
  // ExplorerTargetProfileCard) ; ici uniquement la grille de tuiles KPI (chacune
  // bordée), comme les cards de « Profil de combat ».
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
      <KpiTile label={t('explorer.target_profile.label_matches')} value={fmtNumber(totalMatches, numberLocale, 0)} accent="chart-series-1" />
      <KpiTile label={t('explorer.target_profile.label_kda')} value={fmtNumber(careerStats.kda, numberLocale)} accent="perf-tier-2" />
      <KpiTile label={t('explorer.target_profile.label_win_rate')} value={fmtPct(careerStats.win_rate, numberLocale)} accent="outcome-win" />
      <KpiTile label={t('explorer.target_profile.label_accuracy')} value={fmtPct(careerStats.accuracy, numberLocale)} accent="info" />
      <KpiTile label={t('explorer.target_profile.label_dmg_per_game')} value={fmtNumber(careerStats.damage_per_game, numberLocale, 0)} accent="chart-series-4" />
      <KpiTile
        label={t('explorer.target_profile.time_played_label')}
        value={formatPlaytime(careerStats.time_played_seconds ?? 0, appLocale)}
        accent="chart-series-2"
        testId="explorer-target-time-played"
      />
    </div>
  )
}

interface KpiTileProps {
  label: string
  value: string
  /** Couleur de la barre d'accent (3px) en haut de la card — parité Synthesis AccentCard. */
  accent: SemanticToken
  testId?: string
}

function KpiTile({ label, value, accent, testId }: KpiTileProps) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-border bg-card"
      data-testid={testId}
    >
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />
      <div className="flex flex-col gap-1 px-3 py-2">
        <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
        <span className="text-xl font-semibold text-foreground">{value}</span>
      </div>
    </div>
  )
}
