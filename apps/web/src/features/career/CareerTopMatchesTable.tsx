/**
 * CareerTopMatchesTable — tableau des meilleurs/pires matchs.
 * A2/A3 NATIVE_COMPONENTS — colonnes K/D/A, badge typé, clic → Match View.
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { CareerTopMatch } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { outcomeKey } from '@/lib/outcome-color'
import { formatDate, intlLocale as toIntlLocale } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  items: CareerTopMatch[]
  /** Si défini, n'affiche que ce variant */
  variant?: 'best' | 'worst'
  title?: string
  playerSlug?: string
}

import { getMatchNarrativeBadgeMeta } from '@/components/ui/match-card-presentation'

function MatchBadge({ type }: { type: string | null }) {
  const meta = getMatchNarrativeBadgeMeta(type)
  if (!meta) {
    return type ? <span className="text-xs text-muted-foreground">{type}</span> : null
  }
  return (
    <span
      className="rounded px-1.5 py-0.5 text-xs font-semibold"
      style={{ backgroundColor: meta.color, color: meta.textColor }}
    >
      {meta.label}
    </span>
  )
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

  const filtered = variant ? items.filter((m) => m.variant === variant) : items
  const defaultTitle =
    variant === 'worst'
      ? t('career.top_matches.default_title_worst')
      : variant === 'best'
        ? t('career.top_matches.default_title_best')
        : t('career.top_matches.default_title_neutral')

  function goToMatch(matchId: string) {
    navigateToMatch(matchId, {
      source: 'history',
      matchIds: filtered.map((m) => m.match_id),
      contextDescriptor: { kind: 'top_matches' },
    })
  }

  if (filtered.length === 0) {
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
                <th className="pb-2 text-left">{t('career.top_matches.col_date')}</th>
                <th className="pb-2 text-left">{t('career.top_matches.col_map_mode')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_kills_short')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_deaths_short')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_assists_short')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_kd')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_score')}</th>
                <th className="pb-2 text-right">{t('career.top_matches.col_outcome')}</th>
                <th className="pb-2 text-left pl-3">{t('career.top_matches.col_badge')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filtered.map((m, idx) => (
                <tr
                  key={m.match_id}
                  className="cursor-pointer transition-colors hover:bg-accent"
                  onClick={() => goToMatch(m.match_id)}
                >
                  <td className="py-1.5 text-muted-foreground font-mono text-xs">{idx + 1}</td>
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
                    {m.kills ?? '—'}
                  </td>
                  <td className="py-1.5 text-right font-mono" style={{ color: tokenCssVar('divergent-neg') }}>
                    {m.deaths ?? '—'}
                  </td>
                  <td className="py-1.5 text-right text-muted-foreground font-mono">
                    {m.assists ?? '—'}
                  </td>
                  <td className="py-1.5 text-right font-mono text-foreground">
                    {m.kd_ratio != null ? m.kd_ratio.toFixed(1) : '—'}
                  </td>
                  <td className="py-1.5 text-right text-muted-foreground">{m.score_label ?? '—'}</td>
                  <td className="py-1.5 text-right">
                    {m.outcome_label && (
                      <Badge variant={outcomeBadgeVariant(m.outcome_code)}>
                        {m.outcome_label}
                      </Badge>
                    )}
                  </td>
                  <td className="py-1.5 pl-3">
                    <MatchBadge type={m.badge_type ?? null} />
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
