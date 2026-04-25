/**
 * CareerTopMatchesTable — tableau des meilleurs/pires matchs.
 * A2/A3 NATIVE_COMPONENTS — colonnes K/D/A, badge typé, clic → Match View.
 */
import { useNavigate, useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { CareerTopMatch } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'

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

export function CareerTopMatchesTable({ items, variant, title, playerSlug: slugProp }: Props) {
  const navigate = useNavigate()
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = slugProp ?? params.playerSlug ?? ''

  const filtered = variant ? items.filter((m) => m.variant === variant) : items
  const defaultTitle =
    variant === 'worst' ? 'Pires matchs' : variant === 'best' ? 'Meilleurs matchs' : 'Top matchs'

  function goToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/matches/$matchId',
      params: { playerSlug, matchId },
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
                <th className="pb-2 text-left">#</th>
                <th className="pb-2 text-left">Date</th>
                <th className="pb-2 text-left">Carte / Mode</th>
                <th className="pb-2 text-right">K</th>
                <th className="pb-2 text-right">D</th>
                <th className="pb-2 text-right">A</th>
                <th className="pb-2 text-right">K/D</th>
                <th className="pb-2 text-right">Score</th>
                <th className="pb-2 text-right">Résultat</th>
                <th className="pb-2 text-left pl-3">Badge</th>
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
                    {m.start_time
                      ? new Date(m.start_time).toLocaleDateString('fr-FR')
                      : '—'}
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
                      <Badge
                        variant={
                          m.outcome_label.toLowerCase().includes('victoire')
                            ? 'success'
                            : m.outcome_label.toLowerCase().includes('défaite')
                            ? 'destructive'
                            : 'secondary'
                        }
                      >
                        {m.outcome_label}
                      </Badge>
                    )}
                  </td>
                  <td className="py-1.5 pl-3">
                    <MatchBadge type={m.badge_type} />
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
