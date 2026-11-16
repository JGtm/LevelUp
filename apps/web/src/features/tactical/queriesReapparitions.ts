/**
 * queriesReapparitions — les lectures TanStack Query de la lecture « Réapparitions adverses » : la
 * répartition des réapparitions adverses pour une zone tenue, et l'image plan qui rogne les zones.
 *
 * CETTE LECTURE NE GARDE PAS LA RÉPONSE PRÉCÉDENTE (aucun `placeholderData`), comme le détail d'une cellule :
 * changer de zone tenue ou de périmètre afficherait la répartition d'une AUTRE zone sous la zone choisie.
 * La vue repart à vide et dit « Chargement » (`queriesPlaceholder.guard.test.ts` ne couvre que `queries.ts`).
 */
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { TacticalReapparitionsAdverses, TacticalReapparitionsAdversesBody } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { SANS_BARRE_META } from '@/lib/query/sansBarre'
import { useAppShellStore } from '@/stores/appShellStore'

import { hashFiltre, useTacticalMapBackgroundImage, useTacticalMapPlanImage } from './queries'

/** Ce qui détermine la lecture : le périmètre résolu (`null` = pas encore), la composition et la zone. */
export interface ParamsDesReapparitions {
  match_ids: string[] | null
  coequipiers: string[]
  /** L'indice de volume de la zone (choisie, ou de sonde) ; `null` = aucune zone lisible : rien n'est demandé. */
  volume_index: number | null
}

/** La fraîcheur d'une lecture : celle du plan (`RASTER_FRAIS_MS`), la lecture se recalcule avec le cache serveur. */
const REAPPARITIONS_FRAICHES_MS = 2 * 60 * 1000

/**
 * useReapparitionsAdverses — `POST /players/{slug}/tactical/{map}/reapparitions-adverses`.
 *
 * `match_ids` à `null` (périmètre non résolu) ou `volume_index` à `null` (aucune zone à lire) : la requête
 * N'EST PAS lancée. Une liste VIDE est une réponse légitime (le filtre ne retient rien) et part normalement.
 */
export function useReapparitionsAdverses(playerSlug: string, mapId: string, params: ParamsDesReapparitions) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  const { match_ids, coequipiers, volume_index } = params
  const corps: TacticalReapparitionsAdversesBody | null =
    volume_index === null ? null : { match_ids: match_ids ?? [], coequipiers, volume_index }
  return useQuery({
    queryKey: queryKeys.tacticalReapparitionsAdverses(playerSlug, titleSlug, mapId, hashFiltre(corps), locale),
    queryFn: () =>
      api.post<TacticalReapparitionsAdverses>(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/reapparitions-adverses`,
        corps,
      ),
    enabled: !!playerSlug && !!mapId && corps !== null && match_ids !== null,
    staleTime: REAPPARITIONS_FRAICHES_MS,
    // Un refus (zone inconnue, titre sans la capacité) est un fait, pas une panne : pas de nouvelle tentative.
    retry: false,
    meta: SANS_BARRE_META,
  })
}

/** L'image plan qui rogne les zones : son adresse, ou `null` (carte sans vue plan, ou image en échec). */
export interface PlanDuRognage {
  /** Adresse de l'image plan ; `null` = pas de rognage (les zones restent entières). */
  href: string | null
  /**
   * La décision est prise : on sait s'il y a un plan à appliquer. Tant qu'elle ne l'est pas, les zones ne
   * se peignent pas — elles se montreraient entières un instant, puis rognées.
   */
  decide: boolean
}

/**
 * usePlanDuRognage — l'image plan de la carte, demandée QUEL QUE SOIT le fond affiché (le rognage vaut
 * sur le rendu comme sur le plan). Même clé de cache que le fond plan de la page : jamais téléchargée deux
 * fois. Sans image plan au sidecar du fond, ou si elle ne se charge pas, il n'y a pas de rognage — un repli
 * nommé, pas une erreur à l'écran.
 */
export function usePlanDuRognage(playerSlug: string, mapId: string): PlanDuRognage {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const queryClient = useQueryClient()
  const { planImage, pending } = useTacticalMapPlanImage(playerSlug, mapId)
  const aUnPlan = !pending && !!planImage
  const { url, variante } = useTacticalMapBackgroundImage(playerSlug, mapId, aUnPlan, 'plan')
  const etatImage = queryClient.getQueryState(queryKeys.tacticalMapBackground(titleSlug, mapId, 'plan'))?.status
  const imageRegleee = etatImage === 'success' || etatImage === 'error'
  return {
    href: aUnPlan && variante === 'plan' ? url : null,
    decide: !pending && (!planImage || imageRegleee),
  }
}
