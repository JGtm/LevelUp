/**
 * ExplorerEncounterBriefing — affiche les stats de rencontre d'un couple
 * (player_courant, target) en mode briefing (carte + grille KPI).
 *
 * Reproduit exactement les **calculs et représentations** des colonnes
 * du tableau `MatchEncountersTable` de la page match-view (la colonne "Rôle"
 * est omise car ne s'applique pas hors d'un match courant) :
 *
 *   1. Rencontres : AllyEnemySplitBar (A/E) ou count_together
 *   2. WR allié : winrate_as_ally — ≥50% vert, sinon orange
 *   3. WR ennemi : winrate_vs_enemy — ≥50% vert, sinon orange
 *   4. F/D croisé : KDSplitBar ou kills_dealt/deaths_suffered
 *   5. Ratio : kills_dealt ÷ deaths_suffered, coloré outcome-win/loss/draw
 *   6. Vu pour la dernière fois : last_seen_at (relative time)
 *
 * Conteneur style SessionBriefing : carte avec header + grille de KPIs.
 */
import type { ExplorerEncounterStats } from '@/lib/api/types'
import { formatPercentInt } from '@/lib/formatters'
import { kdRatioColor, winRateClass } from '@/lib/colors/outcomePalette'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { Tooltip } from '@/components/ui/tooltip'

// ─── Accent KPI selon le sentiment du contenu (barre colorée en haut) ────────
// Vert (positif) / rouge (négatif) / bleu = outcome-draw (ok, égal ou neutre).

/** cmpAccent : compare a vs b (frags vs morts, etc.). > = vert, < = rouge, = / nil = bleu. */
function cmpAccent(a: number | null | undefined, b: number | null | undefined): SemanticToken {
  if (a == null || b == null || a === b) return 'outcome-draw'
  return a > b ? 'outcome-win' : 'outcome-loss'
}

/** wrAccent : taux de victoire vs seuil 0.5. > = vert, < = rouge, = / nil = bleu. */
function wrAccent(wr: number | null | undefined): SemanticToken {
  if (wr == null) return 'outcome-draw'
  if (wr > 0.5) return 'outcome-win'
  if (wr < 0.5) return 'outcome-loss'
  return 'outcome-draw'
}

interface Props {
  stats: ExplorerEncounterStats
  /** Locale UI ('fr' | 'en') — défaut 'fr' si autre valeur. */
  locale: string
}

// ─── Helpers de formatage (copiés à l'identique de MatchEncountersTable.tsx) ─

function formatKDCross(
  kills: number | null | undefined,
  deaths: number | null | undefined,
): string {
  if (kills == null && deaths == null) return '—'
  return `${kills ?? 0}/${deaths ?? 0}`
}

function formatKDRatio(kills: number | null | undefined, deaths: number | null | undefined): string {
  if (kills == null || deaths == null) return '—'
  if (deaths === 0) return kills > 0 ? '∞' : '—'
  return (kills / deaths).toFixed(2)
}

function formatRelativeFR(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return "à l'instant"
  if (minutes < 60) return `il y a ${minutes} min`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return hours <= 1 ? 'il y a 1 h' : `il y a ${hours} h`
  const days = Math.round(hours / 24)
  if (days === 1) return 'hier'
  if (days < 7) return `il y a ${days} j`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return weeks <= 1 ? 'il y a 1 sem.' : `il y a ${weeks} sem.`
  const months = Math.round(days / 30)
  if (months < 12) return months <= 1 ? 'il y a 1 mois' : `il y a ${months} mois`
  const years = Math.round(days / 365)
  return years <= 1 ? 'il y a 1 an' : `il y a ${years} ans`
}

function formatRelativeEN(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours} h ago`
  const days = Math.round(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days} d ago`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return `${weeks} w ago`
  const months = Math.round(days / 30)
  if (months < 12) return `${months} mo ago`
  const years = Math.round(days / 365)
  return years <= 1 ? '1 y ago' : `${years} y ago`
}

// ─── SplitBar (copié depuis MatchEncountersTable.tsx) ─────────────────────────

function SplitBar({
  leftCount,
  rightCount,
  leftColor,
  rightColor,
  leftTooltip,
  rightTooltip,
}: {
  leftCount: number
  rightCount: number
  leftColor: string
  rightColor: string
  leftTooltip: string
  rightTooltip: string
}) {
  const total = leftCount + rightCount
  if (total === 0) return <span className="font-mono">—</span>
  const leftPct = Math.round((leftCount / total) * 100)
  return (
    <span className="inline-flex items-center gap-1 font-mono tabular-nums">
      <Tooltip content={leftTooltip}>
        <span style={{ color: leftColor }}>{leftCount}</span>
      </Tooltip>
      <span className="inline-flex h-2 w-12 border border-border overflow-hidden">
        <span style={{ width: `${leftPct}%`, backgroundColor: leftColor }} />
        <span style={{ flex: 1, backgroundColor: rightColor }} />
      </span>
      <Tooltip content={rightTooltip}>
        <span style={{ color: rightColor }}>{rightCount}</span>
      </Tooltip>
    </span>
  )
}

function AllyEnemySplitBar({
  allyCount,
  enemyCount,
  locale,
}: {
  allyCount: number
  enemyCount: number
  locale: 'fr' | 'en'
}) {
  const ttAlly = locale === 'en' ? `${allyCount} matches as ally` : `${allyCount} matchs en allié`
  const ttEnemy = locale === 'en' ? `${enemyCount} matches as enemy` : `${enemyCount} matchs en ennemi`
  return (
    <SplitBar
      leftCount={allyCount}
      rightCount={enemyCount}
      leftColor={tokenCssVar('team-ally')}
      rightColor={tokenCssVar('team-enemy')}
      leftTooltip={ttAlly}
      rightTooltip={ttEnemy}
    />
  )
}

function KDSplitBar({
  kills,
  deaths,
  locale,
}: {
  kills: number
  deaths: number
  locale: 'fr' | 'en'
}) {
  const ttKills = locale === 'en' ? `${kills} kills dealt` : `${kills} frags infligés`
  const ttDeaths = locale === 'en' ? `${deaths} deaths suffered` : `${deaths} morts subies`
  return (
    <SplitBar
      leftCount={kills}
      rightCount={deaths}
      leftColor={tokenCssVar('outcome-win')}
      rightColor={tokenCssVar('outcome-loss')}
      leftTooltip={ttKills}
      rightTooltip={ttDeaths}
    />
  )
}

// ─── KpiCard (style SessionBriefing) ─────────────────────────────────────────

interface KpiCardProps {
  label: React.ReactNode
  value: React.ReactNode
  detail?: React.ReactNode
  /** Barre d'accent (3px) en haut, couleur déterminée par le contenu. */
  accent?: SemanticToken
}

function KpiCard({ label, value, detail, accent }: KpiCardProps) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card">
      {accent && <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />}
      <div className="p-3">
        <p className="text-3xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        <p className="mt-1 text-xl font-bold tabular-nums leading-tight text-foreground">
          {value}
        </p>
        {detail && <p className="mt-0.5 text-xs text-muted-foreground">{detail}</p>}
      </div>
    </div>
  )
}

// ─── Composant principal ─────────────────────────────────────────────────────

export function ExplorerEncounterBriefing({ stats, locale }: Props) {
  const manifestLocale: 'fr' | 'en' = locale === 'en' ? 'en' : 'fr'
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, manifestLocale, values)
  const formatRelative = manifestLocale === 'en' ? formatRelativeEN : formatRelativeFR

  const ally = stats.ally_count ?? null
  const enemy = stats.enemy_count ?? null

  const ratioColor = kdRatioColor(stats.kills_dealt, stats.deaths_suffered)

  return (
    // Sortie du bloc (cf. demande user) : pas de carte englobante — la rangée de
    // KPI cards (chacune bordée) flotte directement, comme les autres rangées KPI.
    // Contenu des cards inchangé.
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
          {/* Rencontres */}
          <KpiCard
            label={t('explorer.encounter.encounters')}
            value={
              ally != null && enemy != null
                ? <AllyEnemySplitBar allyCount={ally} enemyCount={enemy} locale={manifestLocale} />
                : stats.count_together
            }
            accent="outcome-draw"
          />
          {/* WR allié */}
          <KpiCard
            label={t('explorer.encounter.wr_ally')}
            value={
              <span className="inline-flex items-baseline gap-1.5">
                <span className={winRateClass(stats.winrate_as_ally)}>
                  {formatPercentInt(stats.winrate_as_ally)}
                </span>
                {ally != null && (
                  <span className="text-xs font-normal text-muted-foreground">
                    {ally} {manifestLocale === 'en' ? 'matches' : 'matchs'}
                  </span>
                )}
              </span>
            }
            accent={wrAccent(stats.winrate_as_ally)}
          />
          {/* WR ennemi */}
          <KpiCard
            label={t('explorer.encounter.wr_enemy')}
            value={
              <span className="inline-flex items-baseline gap-1.5">
                <span className={winRateClass(stats.winrate_vs_enemy)}>
                  {formatPercentInt(stats.winrate_vs_enemy)}
                </span>
                {enemy != null && (
                  <span className="text-xs font-normal text-muted-foreground">
                    {enemy} {manifestLocale === 'en' ? 'matches' : 'matchs'}
                  </span>
                )}
              </span>
            }
            accent={wrAccent(stats.winrate_vs_enemy)}
          />
          {/* F/D croisé */}
          <KpiCard
            label={t('explorer.encounter.kd_cross')}
            value={
              stats.kills_dealt != null && stats.deaths_suffered != null
                ? <KDSplitBar kills={stats.kills_dealt} deaths={stats.deaths_suffered} locale={manifestLocale} />
                : <span className="font-mono">{formatKDCross(stats.kills_dealt, stats.deaths_suffered)}</span>
            }
            accent={cmpAccent(stats.kills_dealt, stats.deaths_suffered)}
          />
          {/* Ratio */}
          <KpiCard
            label={
              <Tooltip content={t('explorer.encounter.ratio_tooltip')}>
                <span className="cursor-help border-b border-dashed border-current">
                  {t('explorer.encounter.ratio')}
                </span>
              </Tooltip>
            }
            value={
              <span className="font-mono font-bold" style={ratioColor ? { color: ratioColor } : undefined}>
                {formatKDRatio(stats.kills_dealt, stats.deaths_suffered)}
              </span>
            }
            accent={cmpAccent(stats.kills_dealt, stats.deaths_suffered)}
          />
          {/* Vu pour la dernière fois */}
          <KpiCard
            label={t('explorer.encounter.last_seen')}
            value={
              <span className="text-base text-muted-foreground">
                {stats.last_seen_at ? formatRelative(stats.last_seen_at) : '—'}
              </span>
            }
            accent="outcome-draw"
          />
        </div>
  )
}
