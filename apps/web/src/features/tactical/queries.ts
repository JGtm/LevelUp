/**
 * queries — les lectures TanStack Query de l'onglet Tactique.
 *
 * DEUX LECTURES, DEUX CLÉS, et elles ne s'invalident pas ensemble : la GRILLE dépend du
 * filtre courant (quelles cartes, combien de matchs) ; le FOND d'une carte n'en dépend pas
 * du tout — il est figé entre deux cuissons. Même partage que
 * `matchReplay` / `matchReplayBackgroundImage`.
 */
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { TacticalMapsPage } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSoloFilterStore } from '@/stores/soloFilterStore'

import { tacticalFilterQuery } from './tacticalLogic'

/**
 * useTacticalMaps — la grille des cartes jouées sous le FILTRE GLOBAL de l'omnibar.
 *
 * L'onglet n'a AUCUN filtre propre (décision de la phase 4) : il lit le même contexte que
 * les autres surfaces solo, via `useSoloFilterStore`. Un second vocabulaire de filtre
 * donnerait deux comptes de matchs différents pour la même question.
 */
export function useTacticalMaps(playerSlug: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const filterContext = useSoloFilterStore((s) => s.filterContext)
  // La chaîne de requête EST l'empreinte de cache : deux filtres différents la rendent
  // différente, et c'est précisément ce que la clé doit distinguer.
  const query = useMemo(() => tacticalFilterQuery(filterContext), [filterContext])
  return useQuery({
    queryKey: queryKeys.tacticalMaps(playerSlug, titleSlug, query),
    queryFn: () =>
      api.get<TacticalMapsPage>(
        `/players/${playerSlug}/tactical/maps${query ? `?${query}` : ''}`,
      ),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
  })
}

/**
 * useTacticalMapBackgroundUrl — l'URL affichable du fond d'une carte, ou `null`.
 *
 * L'URL D'OBJET **EST** L'ENTRÉE DE CACHE, et c'est un choix, pas une facilité. Elle est
 * créée une fois dans la requête et gardée pour la session (`staleTime` et `gcTime`
 * infinis) : une image de carte ne change qu'à une re-cuisson, et revenir sur l'onglet doit
 * réafficher la grille sans re-télécharger. La contrepartie assumée est qu'aucune URL n'est
 * révoquée avant la fermeture de l'onglet ; créer l'URL au montage de chaque vignette
 * aurait imposé un `setState` dans un effet — cascade de rendus que le lint du dépôt
 * signale, pour un gain nul à cette échelle.
 *
 * LA BORNE EST LE NOMBRE DE CARTES DU TITRE, quel que soit le nombre de joueurs consultés
 * (revue R1, W4) : la clé de cache ne porte PAS le joueur. Une image de carte est une donnée
 * de référence du titre, identique pour tout le monde ; la mettre en cache par joueur
 * retenait N fois le même contenu. Le joueur reste dans l'URL de fetch, parce que la route
 * est derrière l'ownership.
 *
 * 404 = la carte n'a pas de fond figé : cas NOMINAL (toutes les cartes n'en ont pas). Pas
 * de nouvelle tentative, rien à dire à l'utilisateur — la vignette s'affiche sans image.
 */
export function useTacticalMapBackgroundUrl(playerSlug: string, mapId: string): string | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.tacticalMapBackground(titleSlug, mapId),
    queryFn: async () => {
      const blob = await api.getBlob(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/background.png`,
      )
      // `createObjectURL` manque dans certains environnements de test : l'absence de fond
      // n'est pas une panne de grille, la vignette s'affiche sans image.
      if (typeof URL.createObjectURL !== 'function') return null
      return URL.createObjectURL(blob)
    },
    staleTime: Infinity,
    gcTime: Infinity,
    enabled: !!playerSlug && !!mapId,
    retry: false,
  })
  return data ?? null
}
