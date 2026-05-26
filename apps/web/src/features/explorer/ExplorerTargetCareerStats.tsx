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
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { NormalizedPlayerStats } from '@/lib/api/types'

interface ExplorerTargetCareerStatsProps {
  careerStats: NormalizedPlayerStats
  /** Compteurs wins/losses/draws — déduits de matches × winRate quand possible. */
  wins?: number
  losses?: number
  draws?: number
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

export function ExplorerTargetCareerStats({
  careerStats,
  wins,
  losses,
  draws,
}: ExplorerTargetCareerStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const numberLocale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, appLocale, values)

  // Si wins/losses/draws ne sont pas fournis, on les dérive du win_rate (best-effort).
  const totalMatches = careerStats.matches ?? 0
  const derivedDraws = draws ?? 0
  let derivedWins = wins
  let derivedLosses = losses
  if (derivedWins == null || derivedLosses == null) {
    // win_rate est typiquement en 0..1.
    const wr = careerStats.win_rate > 1.5 ? careerStats.win_rate / 100 : careerStats.win_rate
    derivedWins = Math.round(totalMatches * wr)
    derivedLosses = totalMatches - derivedWins - derivedDraws
    if (derivedLosses < 0) derivedLosses = 0
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-baseline gap-2 text-base">
          {t('explorer.target_profile.section_career_title')}
          <span className="text-xs font-normal text-muted-foreground">
            {t('explorer.target_profile.section_career_subtitle')}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <KpiTile label={t('explorer.target_profile.label_matches')} value={fmtNumber(totalMatches, numberLocale, 0)} />
          <KpiTile label={t('explorer.target_profile.label_kda')} value={fmtNumber(careerStats.kda, numberLocale)} />
          <KpiTile label={t('explorer.target_profile.label_kdr')} value={fmtNumber(careerStats.kdr, numberLocale)} />
          <KpiTile label={t('explorer.target_profile.label_win_rate')} value={fmtPct(careerStats.win_rate, numberLocale)} />
          <KpiTile label={t('explorer.target_profile.label_accuracy')} value={fmtPct(careerStats.accuracy, numberLocale)} />
          <KpiTile label={t('explorer.target_profile.label_dmg_per_game')} value={fmtNumber(careerStats.damage_per_game, numberLocale, 0)} />
        </div>
        {totalMatches > 0 && (
          <OutcomeBar wins={derivedWins ?? 0} draws={derivedDraws} losses={derivedLosses ?? 0} />
        )}
      </CardContent>
    </Card>
  )
}

interface KpiTileProps {
  label: string
  value: string
}

function KpiTile({ label, value }: KpiTileProps) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border border-border bg-card px-3 py-2">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
      <span className="text-xl font-semibold text-foreground">{value}</span>
    </div>
  )
}
