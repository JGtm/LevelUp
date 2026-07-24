/**
 * CareerTopMatchesTable — tableau des meilleurs/pires matchs.
 * A2/A3 NATIVE_COMPONENTS — colonnes K/D/A, badge typé, clic → Match View.
 * V8b (2026-07-07) — consomme TopMatchDTO (shape réelle de l'endpoint
 * /pages/career/top-matches : best_matches / worst_matches). Les listes best/worst
 * sont déjà séparées côté backend — le composant reçoit un tableau prêt à rendre.
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { TopMatchDTO } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { outcomeKey } from '@/lib/outcome-color'
import { formatDate, intlLocale as toIntlLocale } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { buildWaypointMatchUrl, waypointLogoSrc } from '@/lib/match-nav/waypointUrl'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { useCapability } from '@/lib/capabilities/capabilities'

interface Props {
  items: TopMatchDTO[]
  /** 'best' | 'worst' | undefined — pilote le titre par défaut (les listes sont
   *  déjà filtrées côté backend, ce composant ne re-filtre pas). */
  variant?: 'best' | 'worst'
  title?: string
  playerSlug?: string
}

/**
 * Variante de badge (pill) selon le CODE d'outcome — jamais sur le label
 * localisé (CR C2 : `includes('victoire')` cassait tous les badges en EN).
 */
function outcomeBadgeVariant(code: number | null | undefined): 'success' | 'destructive' | 'secondary' {
  switch (outcomeKey(code ?? 0)) {
    case 'win':
      return 'success'
    case 'loss':
      return 'destructive'
    default:
      return 'secondary'
  }
}

export function CareerTopMatchesTable({ items, variant, title, playerSlug: slugProp }: Props) {
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = slugProp ?? params.playerSlug ?? ''
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const locale = useAppShellStore((s) => s.locale)
  const intlLocale = toIntlLocale(locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)
  // Colonne « Ouvrir sur Halo Waypoint » (I19) : gating par capability (absente
  // pour Halo 5) ET par préférence LOCALE (Apparence → « Colonne Halo Waypoint
  // sur les listes de matchs », défaut ON).
  const waypointCapability = useCapability('waypoint_match_url')
  const showWaypointColumnPref = useSettingsDraftStore((s) => s.localUiPrefs.showWaypointColumn)
  const showWaypoint = waypointCapability && showWaypointColumnPref
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)

  const defaultTitle =
    variant === 'worst'
      ? t('career.top_matches.default_title_worst')
      : variant === 'best'
        ? t('career.top_matches.default_title_best')
        : t('career.top_matches.default_title_neutral')

  function goToMatch(matchId: string) {
    navigateToMatch(matchId, {
      source: 'history',
      matchIds: items.map((m) => m.match_id),
      contextDescriptor: { kind: 'top_matches' },
    })
  }

  if (items.length === 0) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title ?? defaultTitle}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-xs font-medium text-muted-foreground">
                <th className="pb-2 text-left">{t('career.top_matches.col_index')}</th>
                {showWaypoint && <th className="pb-2 text-left" />}
                <th className="pb-2 text-left">{t('career.top_matches.col_date')}</th>
                <th className="pb-2 text-left">{t('career.top_matches.col_map_mode')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_kills_short')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_deaths_short')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_kd')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_score')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_outcome')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {items.map((m, idx) => (
                <tr
                  key={m.match_id}
                  className="cursor-pointer transition-colors hover:bg-accent"
                  onClick={() => goToMatch(m.match_id)}
                >
                  <td className="py-1.5 text-muted-foreground font-mono text-xs">{idx + 1}</td>
                  {showWaypoint && (
                    <td className="py-1.5">
                      <a
                        href={buildWaypointMatchUrl(playerSlug, m.match_id)}
                        target="_blank"
                        rel="noopener noreferrer"
                        onClick={(e) => e.stopPropagation()}
                        aria-label={t('career.top_matches.col_waypoint_aria')}
                        title={t('career.top_matches.col_waypoint_aria')}
                        className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                      >
                        <img
                          src={waypointLogoSrc(theme)}
                          alt=""
                          aria-hidden
                          className="h-4 w-4 opacity-60 group-hover:opacity-100 transition-opacity"
                        />
                      </a>
                    </td>
                  )}
                  <td className="py-1.5 text-muted-foreground whitespace-nowrap">
                    {formatDate(m.start_time, intlLocale, { dateStyle: 'short' }, '—')}
                  </td>
                  <td className="py-1.5">
                    <span className="font-medium text-foreground">{m.map_ui ?? '—'}</span>
                    {m.mode_ui && (
                      <span className="ml-1 text-xs text-muted-foreground">· {m.mode_ui}</span>
                    )}
                  </td>
                  <td className="py-1.5 text-right font-mono" style={{ color: tokenCssVar('perf-tier-2') }}>
                    {m.kills}
                  </td>
                  <td className="py-1.5 text-right font-mono" style={{ color: tokenCssVar('divergent-neg') }}>
                    {m.deaths}
                  </td>
                  <td className="py-1.5 text-right font-mono text-foreground">
                    {m.kda != null ? m.kda.toFixed(1) : '—'}
                  </td>
                  <td className="py-1.5 text-right text-muted-foreground">
                    {m.performance_score != null ? m.performance_score.toFixed(1) : '—'}
                  </td>
                  <td className="py-1.5 text-right">
                    {m.outcome_label && (
                      <Badge variant={outcomeBadgeVariant(m.outcome_code)}>
                        {m.outcome_label}
                      </Badge>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}
