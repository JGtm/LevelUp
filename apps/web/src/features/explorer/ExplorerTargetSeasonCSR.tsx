/**
 * ExplorerTargetSeasonCSR — classements CSR par playlist ranked (saison courante)
 * du joueur cible. Liste compacte : nom de playlist + tier/sub-tier + rating.
 * Données live (endpoint skill) ; n'affiche que les playlists ranked engagées
 * cette saison.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { CareerPlaylistCSR } from '@/lib/api/types'

interface ExplorerTargetSeasonCSRProps {
  csrs: CareerPlaylistCSR[]
  title: string
}

/** tierLabel formate "Tier Sub" (ex: "Onyx", "Diamond 3") ou "—" si non classé. */
function tierLabel(tier: string, subTier: number): string {
  const t = tier.trim()
  if (t === '') return '—'
  // sub_tier est 1-based (1..6) ; 0 = pas de sous-tier (Onyx, ou non classé).
  if (t.toLowerCase() === 'onyx') return t
  return subTier > 0 ? `${t} ${subTier}` : t
}

export function ExplorerTargetSeasonCSR({ csrs, title }: ExplorerTargetSeasonCSRProps) {
  if (csrs.length === 0) return null

  return (
    <Card data-testid="explorer-target-season-csr">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{title}</CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        <ul className="flex flex-col gap-1.5">
          {csrs.map((c) => (
            <li
              key={c.playlist_id}
              className="flex items-center justify-between gap-2 rounded-md border border-border bg-muted/20 px-3 py-1.5"
            >
              {c.current.badge_image_url ? (
                <img
                  src={c.current.badge_image_url}
                  alt=""
                  aria-hidden="true"
                  className="h-6 w-6 flex-shrink-0 object-contain"
                  loading="lazy"
                  decoding="async"
                />
              ) : (
                <span className="h-6 w-6 flex-shrink-0" aria-hidden="true" />
              )}
              <span className="min-w-0 flex-1 truncate text-sm text-foreground">
                {c.playlist_name || c.playlist_id}
              </span>
              <span className="flex-shrink-0 text-sm font-medium text-muted-foreground">
                {tierLabel(c.current.tier, c.current.sub_tier)}
              </span>
              {c.current.value > 0 && (
                <span className="flex-shrink-0 text-sm font-bold tabular-nums text-foreground">
                  {Math.round(c.current.value)}
                </span>
              )}
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
