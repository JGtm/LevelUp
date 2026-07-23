/**
 * MedalsView — vue PARTAGÉE de la page Médailles. Purement présentationnelle :
 * reçoit un view-model déjà groupé/trié/localisé par le conteneur (MedalsPage).
 *
 * Rendu GÉNÉRIQUE : itère les super-sections PRÉSENTES dans le view-model, sans
 * aucun nom de section en dur (Halo Infinite → 4 super-sections riches ; Halo 5
 * & futurs titres → leur regroupement natif). Chaque catégorie : anneau de
 * maîtrise (CitationProgressRing, doré si complète) + libellé + total, puis
 * grille de MedalCard.
 */
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { MedalCard } from './MedalCard'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { MedalSummaryItem } from '@/lib/api/types'

export interface MedalCategoryView {
  key: string
  label: string
  /** Pourcentage de maîtrise (médailles distinctes obtenues / total). */
  pct: number
  /** Catégorie entièrement obtenue → anneau doré. */
  isMastered: boolean
  /** « {earned}/{total} obtenues » déjà localisé. */
  masteryLabel: string
  /** « {count} au total » (somme des compteurs) déjà localisé. */
  totalAwardedLabel: string
  items: MedalSummaryItem[]
}

export interface MedalSuperSectionView {
  key: string
  label: string
  categories: MedalCategoryView[]
}

export interface MedalsViewModel {
  superSections: MedalSuperSectionView[]
}

interface MedalsViewProps {
  vm: MedalsViewModel
  locale: ManifestLocale
  emptyTitle: string
  emptyDescription: string
}

export function MedalsView({ vm, locale, emptyTitle, emptyDescription }: MedalsViewProps) {
  if (vm.superSections.length === 0) {
    return (
      <div className="px-6 pb-6">
        <EmptyStateCard title={emptyTitle} description={emptyDescription} />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-8 px-6 pb-6">
      {vm.superSections.map((section) => (
        <section key={section.key} className="flex flex-col gap-4">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {section.label}
          </h3>
          <div className="flex flex-col gap-6">
            {section.categories.map((category) => (
              <MedalCategoryCard key={category.key} category={category} locale={locale} />
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}

function MedalCategoryCard({ category, locale }: { category: MedalCategoryView; locale: ManifestLocale }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex items-center gap-3 border-b border-border px-3 py-2">
        <CitationProgressRing pct={category.pct} isMastered={category.isMastered} size={40} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-foreground">{category.label}</div>
          <div className="text-xs text-muted-foreground">{category.masteryLabel}</div>
        </div>
        <div className="shrink-0 text-xs text-muted-foreground">{category.totalAwardedLabel}</div>
      </div>
      <div className="p-3">
        <div className="flex flex-wrap justify-center gap-x-5 gap-y-4">
          {category.items.map((item) => (
            <MedalCard key={item.medal_id} item={item} locale={locale} />
          ))}
        </div>
      </div>
    </div>
  )
}
