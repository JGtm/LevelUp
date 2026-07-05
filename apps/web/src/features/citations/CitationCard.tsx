/**
 * CitationCard — tuile PARTAGÉE d'une citation/commendation (Infinite ET H5). Anneau
 * de progression (CitationProgressRing) quand il y a des paliers, sinon dégradation sur
 * l'icône brute (MedalIcon) — préserve le comportement historique H5 quand
 * `commendation_definitions.tier_targets` n'est pas seedé. La maîtrise est déjà décidée
 * en amont (CitationDisplayItem.isMastered via le chokepoint citationMastery), donc
 * Infinite et H5 affichent un anneau doré identique pour une citation maîtrisée.
 */
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { MedalIcon } from '@/components/ui/MedalIcon'
import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { CitationDisplayItem } from '@/lib/citations/types'

export function CitationCard({ item, locale }: { item: CitationDisplayItem; locale: ManifestLocale }) {
  const hasTiers = item.tierCount > 0
  // Tailles par source (parité visuelle avec les pages d'origine) — classes littérales
  // pour rester détectables par le JIT Tailwind.
  const native = item.source === 'native'
  // Infinite affiche TOUJOURS l'anneau (parité historique : même à 0 % ou sans paliers).
  // Le natif dégrade sur l'icône brute quand les paliers ne sont pas seedés (tier_count=0).
  const showRing = !native || hasTiers
  const ringSize = native ? 56 : 68
  const widthClass = native ? 'w-[84px]' : 'w-[98px]'
  const tooltip = item.description ? `${item.name} : ${item.description}` : item.name
  // Pied : progression vers le prochain palier si tierée et non maîtrisée ; sinon, pour
  // la source native (vitrine de totaux à vie), on affiche le total cumulé.
  const showProgress = hasTiers && !item.isMastered

  return (
    <div title={tooltip} className={`flex cursor-default flex-col items-center gap-1 ${widthClass}`}>
      {showRing ? (
        <CitationProgressRing
          pct={item.pct}
          imageUrl={item.imageUrl}
          isMastered={item.isMastered}
          size={ringSize}
        />
      ) : item.imageUrl ? (
        <MedalIcon imageUrl={item.imageUrl} label={item.name} size={52} className="object-contain" />
      ) : (
        <div className="flex h-[52px] w-[52px] items-center justify-center rounded bg-muted px-1">
          <span className="text-3xs leading-none break-all text-center text-muted-foreground">
            {item.name}
          </span>
        </div>
      )}
      <span className="w-full truncate text-center text-[12px] leading-tight text-muted-foreground">
        {item.name}
      </span>
      {hasTiers && (
        <span className="text-[12px] font-semibold leading-none text-foreground/80">
          {item.tierIndex}/{item.tierCount}
        </span>
      )}
      {showProgress ? (
        <span className="text-3xs leading-none text-muted-foreground">
          {item.total}/{item.nextTierTarget}
        </span>
      ) : native ? (
        <span className="text-sm font-semibold leading-none text-foreground">
          {item.total.toLocaleString(intlLocale(locale))}
        </span>
      ) : null}
    </div>
  )
}
