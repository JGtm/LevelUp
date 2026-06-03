/**
 * ExplorerTargetSeasonCSR — classements CSR par playlist ranked (saison courante)
 * du joueur cible. Chrome identique aux ChartCard (cf. « Matchs par saison ») :
 * carte bordée + barre de titre + contenu. La liste est centrée verticalement
 * dans le bloc (qui s'étire à la hauteur de « Matchs par saison »).
 *
 * Tiers traduits en FR (locale FR) ; la valeur de rating brute n'est pas affichée
 * (demande user). Liste simple, sans sous-blocs bordés.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { CSR_TIER_GRID } from '@/lib/skillTiers'
import type { CareerPlaylistCSR } from '@/lib/api/types'

interface ExplorerTargetSeasonCSRProps {
  csrs: CareerPlaylistCSR[]
  title: string
}

// Traduction EN→FR des noms de tier CSR (Bronze/Silver/Gold/Platinum/Diamond/Onyx),
// dérivée de la grille canonique CSR_TIER_GRID (source unique des libellés de rang).
const CSR_TIER_FR: Record<string, string> = Object.fromEntries(
  CSR_TIER_GRID.tiers.map((t) => [t.en.toLowerCase(), t.fr]),
)

// Sous-paliers en chiffres romains (I..VI), comme partout ailleurs (cf. Go
// rankSubRoman / skillTierLabel). Index 0 = pas de sous-palier.
const SUBTIER_ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI'] as const

/** tierLabel formate "Tier Sub" (ex: "Onyx", "Diamant III") ou "—" si non classé. */
function tierLabel(tier: string, subTier: number, locale: string): string {
  const raw = tier.trim()
  if (raw === '') return '—'
  const name = locale === 'en' ? raw : CSR_TIER_FR[raw.toLowerCase()] ?? raw
  // sub_tier est 1-based (1..6) ; 0 = pas de sous-tier (Onyx, ou non classé).
  if (raw.toLowerCase() === 'onyx') return name
  return subTier >= 1 && subTier <= 6 ? `${name} ${SUBTIER_ROMAN[subTier]}` : name
}

export function ExplorerTargetSeasonCSR({ csrs, title }: ExplorerTargetSeasonCSRProps) {
  const locale = useAppShellStore((s) => s.locale)
  if (csrs.length === 0) return null

  return (
    <div
      className="flex h-full flex-col rounded-lg border border-border bg-card"
      data-testid="explorer-target-season-csr"
    >
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {title}
      </div>
      <div className="flex flex-1 items-center p-3">
        <ul className="flex w-full flex-col divide-y divide-border/40">
          {csrs.map((c) => (
            <li key={c.playlist_id} className="flex items-center gap-2 py-1.5">
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
                {tierLabel(c.current.tier, c.current.sub_tier, locale)}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
