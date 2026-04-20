/**
 * CareerTopMatchesTable — tableau des meilleurs/pires matchs.
 * A2/A3 NATIVE_COMPONENTS — colonnes K/D/A, badge typé, clic → Match View.
 */
import { useNavigate, useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { CareerTopMatch } from '@/lib/api/types'

interface Props {
  items: CareerTopMatch[]
  /** Si défini, n'affiche que ce variant */
  variant?: 'best' | 'worst'
  title?: string
  playerSlug?: string
}

const BADGE_STYLE: Record<string, { label: string; color: string }> = {
  dominant: { label: 'DOMINATION', color: '#00DC82' },
  humiliation: { label: 'HUMILIATION', color: '#8B5CF6' },
  remontada: { label: 'REMONTADA', color: '#0072B2' },
  debacle: { label: 'DÉBÂCLE', color: '#D55E00' },
  contre_remontada: { label: 'CONTRE-REMONTADA', color: '#33D6FF' },
}

function MatchBadge({ type }: { type: string | null }) {
  if (!type) return null
  const style = BADGE_STYLE[type]
  if (!style) return <span className="text-xs text-muted-foreground">{type}</span>
  return (
    <span
      className="rounded px-1.5 py-0.5 text-xs font-semibold text-primary-foreground"
      style={{ backgroundColor: style.color }}
    >
      {style.label}
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
      to: '/players/$playerSlug/explorer/matches/$matchId',
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
                  <td className="py-1.5 text-right text-[#33D6FF] font-mono">
                    {m.kills ?? '—'}
                  </td>
                  <td className="py-1.5 text-right text-[#FF4B4B] font-mono">
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
