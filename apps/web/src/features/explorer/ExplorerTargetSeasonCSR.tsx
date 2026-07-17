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
import { composeTierLabel } from '@/lib/skillTiers'
import { unrankedBadgeURL } from '@/lib/staticAssets'
import type { CareerPlaylistCSR } from '@/lib/api/types'

interface ExplorerTargetSeasonCSRProps {
  csrs: CareerPlaylistCSR[]
  title: string
  /** Message du placeholder quand aucune donnée CSR — la carte titrée reste rendue
   *  (jamais masquée en silence : visibilité des manques de données). */
  emptyMessage: string
}

/**
 * tierLabel formate "Tier Sub" (ex: "Onyx", "Diamant III") ou "—" si non classé.
 * Compose via `composeTierLabel` (source unique, cf. skillTiers.ts) — pas de mapping
 * local dupliqué ; ici on n'ajoute que le cas non classé (tier vide → "—").
 */
function tierLabel(tier: string, subTier: number, locale: 'fr' | 'en'): string {
  const raw = tier.trim()
  if (raw === '') return '—'
  return composeTierLabel(raw, subTier, locale)
}

export function ExplorerTargetSeasonCSR({ csrs, title, emptyMessage }: ExplorerTargetSeasonCSRProps) {
  const locale = useAppShellStore((s) => s.locale)
  // Liste vide : on rend quand même la carte titrée + un message centré (même chrome
  // que ChartCard.empty) au lieu de masquer le bloc, pour repérer les manques.
  if (csrs.length === 0) {
    return (
      <div
        className="flex h-full flex-col rounded-lg border border-border bg-card"
        data-testid="explorer-target-season-csr"
      >
        <div className="flex-none border-b border-border px-4 py-2 text-sm font-medium">
          {title}
        </div>
        <div
          className="flex flex-1 items-center justify-center px-4 py-3 text-sm text-muted-foreground"
          data-testid="explorer-target-season-csr-empty"
        >
          {emptyMessage}
        </div>
      </div>
    )
  }

  return (
    <div
      className="flex h-full flex-col rounded-lg border border-border bg-card"
      data-testid="explorer-target-season-csr"
    >
      <div className="flex-none border-b border-border px-4 py-2 text-sm font-medium">
        {title}
      </div>
      <div className="flex flex-1 items-center px-4 py-3">
        <ul className="flex w-full flex-col divide-y divide-border/40">
          {csrs.map((c) => {
            // tier vide = NON CLASSÉ cette saison (état valide, pas une erreur) :
            // libellé « Non classé » + badge unranked_0.png. La liste ne contient que
            // des playlists avec données → pas de cas « irrécupérable » ici.
            const unranked = c.current.tier.trim() === ''
            const label = unranked
              ? locale === 'en'
                ? 'Unranked'
                : 'Non classé'
              : tierLabel(c.current.tier, c.current.sub_tier, locale)
            // Badge tout à DROITE (après le label). Unranked → unranked_0.png ;
            // classé → badge de rang (fallback unranked_0 si l'image manque).
            const badgeURL = unranked ? unrankedBadgeURL() : c.current.badge_image_url || unrankedBadgeURL()
            return (
              <li key={c.playlist_id} className="flex items-center gap-2 py-1.5">
                <span className="min-w-0 flex-1 truncate text-sm text-foreground">
                  {c.playlist_name || c.playlist_id}
                </span>
                <span className="flex-shrink-0 text-sm font-medium text-muted-foreground">
                  {label}
                </span>
                <img
                  src={badgeURL}
                  alt=""
                  aria-hidden="true"
                  className="h-6 w-6 flex-shrink-0 object-contain"
                  loading="lazy"
                  decoding="async"
                />
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}
