/**
 * SynthesisHighlightsSection — Sprint 55 D5.
 * Affiche les meilleurs et pires matchs du scope courant.
 */
import { Link } from '@tanstack/react-router'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SynthesisHighlightsPreview, SynthesisMatchHighlight } from '@/lib/api/types'

interface HighlightRowProps {
  item: SynthesisMatchHighlight
  playerSlug: string
}

function HighlightRow({ item, playerSlug }: HighlightRowProps) {
  const isWin = item.outcome === 2
  const kda = item.kda != null ? item.kda.toFixed(2) : '—'
  const perf = item.perf_score != null ? item.perf_score.toFixed(0) : null

  return (
    <div className="flex items-center justify-between gap-3 border-b last:border-0 px-4 py-3">
      <div className="flex items-center gap-3 min-w-0">
        <Badge variant={isWin ? 'success' : 'destructive'} className="shrink-0">
          {isWin ? 'V' : 'D'}
        </Badge>
        <Link
          to={`/players/${playerSlug}/matches/${item.match_id}` as never}
          className="text-sm font-mono text-muted-foreground hover:text-foreground truncate"
        >
          {item.match_id.slice(0, 12)}…
        </Link>
      </div>
      <div className="flex items-center gap-4 shrink-0 text-sm">
        <span className="text-muted-foreground">
          <span className="text-info font-semibold">{item.kills}</span>K{' '}
          <span className="text-destructive">{item.deaths}</span>D
        </span>
        <span className="text-muted-foreground">K/D : <strong className="text-foreground">{kda}</strong></span>
        {perf != null && (
          <span className="text-muted-foreground hidden sm:inline">
            Perf : <strong className="text-foreground">{perf}</strong>
          </span>
        )}
      </div>
    </div>
  )
}

interface SynthesisHighlightsSectionProps {
  highlights: SynthesisHighlightsPreview
  playerSlug: string
}

export function SynthesisHighlightsSection({ highlights, playerSlug }: SynthesisHighlightsSectionProps) {
  const best = highlights.top_by_kills.length > 0
    ? highlights.top_by_kills
    : highlights.top_by_kda

  const worst = highlights.worst_by_deaths

  const hasData = best.length > 0 || worst.length > 0

  if (!hasData) {
    return (
      <Card>
        <CardHeader><CardTitle>Performances marquantes</CardTitle></CardHeader>
        <CardContent>
          <EmptyStateNotice
            title="Aucun match remarquable"
            description="Pas de données de performance pour le scope et la période sélectionnés."
          />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {best.length > 0 && (
        <Card>
          <CardHeader><CardTitle>Meilleurs matchs</CardTitle></CardHeader>
          <CardContent className="p-0">
            {best.map((item) => (
              <HighlightRow key={item.match_id} item={item} playerSlug={playerSlug} />
            ))}
          </CardContent>
        </Card>
      )}
      {worst.length > 0 && (
        <Card>
          <CardHeader><CardTitle>Matchs difficiles</CardTitle></CardHeader>
          <CardContent className="p-0">
            {worst.map((item) => (
              <HighlightRow key={item.match_id} item={item} playerSlug={playerSlug} />
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
