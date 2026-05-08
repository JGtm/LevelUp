/**
 * ExplorerEncounterBriefing — affiche les stats de rencontre d'un couple
 * (player_courant, target) en mode briefing (carte + grille KPI).
 *
 * Reproduit exactement les **calculs et représentations** des 6 KPI colonnes
 * du tableau `MatchEncountersTable` de la page match-view (la colonne "Rôle"
 * est omise car ne s'applique pas hors d'un match courant) :
 *
 *   1. Joueur (gamertag + badges narratifs)
 *   2. Rencontres : count_together + breakdown "(A:X | E:Y)"
 *   3. WR allié : winrate_as_ally — ≥50% vert, sinon orange
 *   4. WR ennemi : winrate_vs_enemy — ≥50% vert, sinon orange
 *   5. K/D croisé : kills_dealt / deaths_suffered
 *   6. Vu pour la dernière fois : last_seen_at (relative time)
 *
 * Conteneur style SessionBriefing : carte avec header + grille de KPIs.
 */
import type { ExplorerEncounterStats } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

interface Props {
  stats: ExplorerEncounterStats
  /** Locale UI ('fr' | 'en') — défaut 'fr' si autre valeur. */
  locale: string
}

// ─── Helpers de formatage (copiés à l'identique de MatchEncountersTable.tsx) ─

function formatPercent(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return `${Math.round(v * 100)}%`
}

function percentClass(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return ''
  return v >= 0.5 ? 'text-success font-bold' : 'text-warning font-bold'
}

function formatKDCross(
  kills: number | null | undefined,
  deaths: number | null | undefined,
): string {
  if (kills == null && deaths == null) return '—'
  return `${kills ?? 0}/${deaths ?? 0}`
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

// ─── KpiCard (style SessionBriefing) ─────────────────────────────────────────

interface KpiCardProps {
  label: string
  value: React.ReactNode
  detail?: React.ReactNode
}

function KpiCard({ label, value, detail }: KpiCardProps) {
  return (
    <div className="rounded-md border border-border bg-card p-3">
      <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <p className="mt-1 text-xl font-bold tabular-nums leading-tight text-foreground">
        {value}
      </p>
      {detail && <p className="mt-0.5 text-xs text-muted-foreground">{detail}</p>}
    </div>
  )
}

// ─── Composant principal ─────────────────────────────────────────────────────

export function ExplorerEncounterBriefing({ stats, locale }: Props) {
  const manifestLocale: 'fr' | 'en' = locale === 'en' ? 'en' : 'fr'
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, manifestLocale, values)
  const formatRelative = manifestLocale === 'en' ? formatRelativeEN : formatRelativeFR

  // Encounters : count_together + breakdown "(A:X | E:Y)"
  const ally = stats.ally_count ?? null
  const enemy = stats.enemy_count ?? null
  const breakdown =
    ally != null && enemy != null ? `(A:${ally} | E:${enemy})` : ''

  return (
    // Wrapper carte rounded — sans barre titre (cf. demande user 2026-05-08).
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      <div className="p-3">
        {/* Grille KPI — 5 colonnes (Rencontres, WR allié, WR ennemi, K/D croisé, Vu) */}
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          {/* Rencontres */}
          <KpiCard
            label={t('explorer.encounter.encounters')}
            value={stats.count_together}
            detail={breakdown || undefined}
          />
          {/* WR allié */}
          <KpiCard
            label={t('explorer.encounter.wr_ally')}
            value={
              <span className={percentClass(stats.winrate_as_ally)}>
                {formatPercent(stats.winrate_as_ally)}
              </span>
            }
            detail={
              ally != null
                ? `${ally} ${manifestLocale === 'en' ? 'matches' : 'matchs'}`
                : undefined
            }
          />
          {/* WR ennemi */}
          <KpiCard
            label={t('explorer.encounter.wr_enemy')}
            value={
              <span className={percentClass(stats.winrate_vs_enemy)}>
                {formatPercent(stats.winrate_vs_enemy)}
              </span>
            }
            detail={
              enemy != null
                ? `${enemy} ${manifestLocale === 'en' ? 'matches' : 'matchs'}`
                : undefined
            }
          />
          {/* K/D croisé */}
          <KpiCard
            label={t('explorer.encounter.kd_cross')}
            value={
              <span className="font-mono">
                {formatKDCross(stats.kills_dealt, stats.deaths_suffered)}
              </span>
            }
          />
          {/* Vu pour la dernière fois */}
          <KpiCard
            label={t('explorer.encounter.last_seen')}
            value={
              <span className="text-base text-muted-foreground">
                {stats.last_seen_at ? formatRelative(stats.last_seen_at) : '—'}
              </span>
            }
          />
        </div>
      </div>
    </div>
  )
}
