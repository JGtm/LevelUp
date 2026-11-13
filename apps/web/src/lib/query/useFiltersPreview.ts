/**
 * useFiltersPreview — POST /players/{slug}/filters/resolve sur un contexte de filtre arbitraire.
 *
 * Vit dans `lib/query/` parce que la barre de filtres du shell, celles des pages locales
 * (`features/_shared`), de l'Escouade et de la Synthèse la lisent : aucune feature ne peut la
 * porter sans que les autres en dépendent.
 */
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { FilterContextInput, FilterContextResolved } from '@/lib/api/types'

/**
 * Résout un FilterContextInput arbitraire (état pending) sans écrire dans le store.
 * Utilisé pour le feedback immédiat sur les incompatibilités de filtres dans le
 * dropdown, avant que l'utilisateur clique sur Analyser.
 *
 * `enabled` : l'Escouade ne l'active que si des filtres sont EN ATTENTE (D4.3,
 * 2026-09-23) — sinon le résolu commité dit déjà la même chose. Désactivée, la
 * requête peut encore rendre la donnée d'une clé précédente (placeholder
 * `keepPreviousData`) : l'appelant qui la désactive doit l'ignorer.
 */
export function useFiltersPreview(
  playerSlug: string,
  input: FilterContextInput,
  { enabled = true }: { enabled?: boolean } = {},
) {
  // Le titre courant scope la clé (cf. queryKeys.filtersPreview) — même motif que
  // useFiltersResolve : pas de preview périmé d'un autre titre après bascule.
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  // FNV-1a 32 bits — même algo que computeHash/computePendingHash. Crucial :
  // le hash doit refléter les diffs de cascade pour que toggler une checkbox
  // dans FiltresPill déclenche un refetch et mette à jour available_options.
  const hash = (() => {
    const s = JSON.stringify(input) ?? ''
    let h = 0x811c9dc5
    for (let i = 0; i < s.length; i++) {
      h ^= s.charCodeAt(i)
      h = Math.imul(h, 0x01000193) >>> 0
    }
    return h.toString(16).padStart(8, '0')
  })()
  return useQuery<FilterContextResolved>({
    queryKey: queryKeys.filtersPreview(playerSlug, titleSlug, hash, locale),
    queryFn: ({ signal }) =>
      api.post<FilterContextResolved>(
        `/players/${playerSlug}/filters/resolve`,
        input,
        undefined,
        { signal },
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  })
}
