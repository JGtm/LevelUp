/**
 * Queries qualité données — compteurs d'inconnus + listes détaillées.
 * Pas de polling continu : staleTime 60 s + refetch au focus + invalidation
 * après chaque action de résolution.
 */
import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  AdminDataQualityCounts,
  AdminDataQualityIssues,
  DataQualityIssueKind,
} from '@/lib/api/types'

// Locale cible des détecteurs sensibles à la traduction (untranslated_modes) :
// paramètre serveur ?locale= (défaut fr) rendu EXPLICITE côté front. On ne
// construit pas d'autre locale aujourd'hui (pas de sélecteur) → valeur constante,
// donc absente de la clé de cache (une seule valeur à l'exécution). Le libellé
// front honnête lit la locale ÉCHOTÉE par la réponse (data.locale).
export const DATA_QUALITY_LOCALE = 'fr'

export function useDataQualityCounts() {
  return useQuery({
    queryKey: queryKeys.adminDataQuality,
    queryFn: () =>
      api.get<AdminDataQualityCounts>(`/admin/monitoring/data-quality?locale=${DATA_QUALITY_LOCALE}`),
    staleTime: 60_000,
    retry: false,
  })
}

export function useDataQualityIssues(kind: DataQualityIssueKind, limit = 50, offset = 0) {
  return useQuery({
    queryKey: queryKeys.adminDataQualityIssues(kind, limit, offset),
    queryFn: () =>
      api.get<AdminDataQualityIssues>(
        `/admin/monitoring/data-quality/issues?kind=${kind}&locale=${DATA_QUALITY_LOCALE}&limit=${limit}&offset=${offset}`,
      ),
    staleTime: 60_000,
    retry: false,
    // Pagination serveur : garder la page précédente affichée le temps du fetch
    // de la suivante (pas de flash de table vide entre deux pages).
    placeholderData: keepPreviousData,
  })
}
