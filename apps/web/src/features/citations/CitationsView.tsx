/**
 * CitationsView — vue PARTAGÉE de la page Citations (Infinite ET commendations natives
 * H5). Purement présentationnelle : reçoit un view-model normalisé + des libellés déjà
 * traduits par le conteneur (zéro accès manifest/i18n/fetch ici). En-tête de maîtrise +
 * cartes par catégorie + grille de CitationCard. La barre de filtres est un slot
 * optionnel (Infinite uniquement).
 */
import type { ReactNode } from 'react'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { CitationCard } from './CitationCard'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { CitationsViewModel } from '@/lib/citations/types'

interface CitationsViewProps {
  vm: CitationsViewModel
  /** Locale active — pour le formatage des nombres (total à vie natif). */
  locale: ManifestLocale
  /** Barre de filtres (Infinite). Absente pour la source native. */
  filterBar?: ReactNode
  /** Titre de maîtrise déjà formaté (ex. « Citations — Maîtrise 12/121 »). */
  headerTitle: string
  /** Suffixe « complétées » par catégorie. */
  completedSuffix: string
  /**
   * Traduit une CLÉ de catégorie servie par l'API (`weapon`, `game_mode`…) en
   * libellé affichable. Fourni par le conteneur — la vue reste sans accès i18n.
   */
  categoryLabel: (key: string) => string
  emptyTitle: string
  emptyDescription: string
}

export function CitationsView({
  vm,
  locale,
  filterBar,
  headerTitle,
  completedSuffix,
  categoryLabel,
  emptyTitle,
  emptyDescription,
}: CitationsViewProps) {
  return (
    <div className="flex flex-col gap-6">
      {filterBar}
      <div className="space-y-6 px-6 pb-6">
        <h2 className="text-sm font-semibold">{headerTitle}</h2>

        {vm.categories.length === 0 ? (
          <EmptyStateCard title={emptyTitle} description={emptyDescription} />
        ) : (
          vm.categories.map((group) => (
            <div key={group.category} className="rounded-lg border border-border bg-card">
              <div className="flex items-center justify-between border-b border-border px-3 py-2 text-sm font-medium">
                <span>{categoryLabel(group.category)}</span>
                <span className="text-xs font-normal text-muted-foreground">
                  {group.completed} / {group.items.length} {completedSuffix}
                </span>
              </div>
              <div className="p-3">
                <div className="flex flex-wrap justify-center gap-x-5 gap-y-4">
                  {group.items.map((item) => (
                    <CitationCard key={item.key} item={item} locale={locale} />
                  ))}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
