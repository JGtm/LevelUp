/**
 * queriesRelief — LES LECTURES DU RELIEF DU FOND d'une carte de l'onglet Tactique.
 *
 *   - `useTacticalMapOffers` : ce que le sidecar du fond annonce (une image plan, un relief) ; même requête que
 *     le calage et le plan des murs, rien de plus n'est chargé ;
 *   - `tacticalReliefSource` : où lire les deux fichiers du relief (`…/tactical/{map}/background.relief.json` et
 *     `…/background.png?variante=relief`) et sous quelles clés, pour la lecture commune des vues 2D
 *     (`useReliefView`) ;
 *   - `useTacticalBackgroundElement` : le fond de base DÉCODÉ (un élément image), seulement quand une coupe doit le
 *     moduler — le plan lit son fond par une URL d'objet, qu'un canvas ne peut pas moduler sans l'avoir décodé.
 *
 * Tout est propre à la CARTE (clés sans joueur ni langue, figé entre deux cuissons) : une carte sans relief répond
 * 404 sur le sidecar, cas nominal, jamais une erreur affichée.
 */
import { useQuery } from '@tanstack/react-query'

import { queryKeys } from '@/lib/query/keys'
import { backgroundImagePath, reliefDataPath, type MapBackgroundImageVariant } from '@/lib/replay/mapBackgroundVariant'
import type { ReliefFilesSource } from '@/lib/replay/reliefQueries'
import { useAppShellStore } from '@/stores/appShellStore'

import { useTacticalMapBackgroundSidecarQuery } from './queries'

/** Ce que le sidecar du fond d'une carte annonce. */
export interface TacticalMapOffers {
  /** Le nom de l'image plan, ou `undefined` (carte sans plan ou sans fond figé). */
  planImage: string | undefined
  /** Le sidecar porte un relief (`reliefImage` et `reliefData`). */
  reliefOffered: boolean
}

export function useTacticalMapOffers(playerSlug: string, mapId: string): TacticalMapOffers {
  const { data } = useTacticalMapBackgroundSidecarQuery(playerSlug, mapId)
  return { planImage: data?.planImage, reliefOffered: !!data?.reliefImage && !!data.reliefData }
}

/** Où lire le relief d'une carte du titre `titleSlug`, pour le joueur `playerSlug` (la route est derrière l'ownership). */
export function tacticalReliefSource(titleSlug: string, playerSlug: string, mapId: string): ReliefFilesSource {
  const imageBase = `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/background.png`
  return {
    dataKey: queryKeys.tacticalMapReliefData(titleSlug, mapId),
    dataPath: reliefDataPath(imageBase),
    imageKey: queryKeys.tacticalMapReliefImage(titleSlug, mapId),
    imagePath: backgroundImagePath(imageBase, 'relief'),
  }
}

/** decodeImage rend l'élément une fois l'image RÉELLEMENT décodée. */
function decodeImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('image de fond : échec du décodage'))
    img.src = url
  })
}

/**
 * useTacticalBackgroundElement décode le fond de base posé (`url`, l'URL d'objet de `useTacticalMapBackgroundImage`) quand
 * `enabled` ; `null` sinon ou tant qu'il se décode. Un échec laisse le fond non modulé : jamais d'erreur affichée.
 */
export function useTacticalBackgroundElement(
  mapId: string,
  variante: MapBackgroundImageVariant,
  url: string | null,
  enabled: boolean,
): HTMLImageElement | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.tacticalMapBackgroundElement(titleSlug, mapId, variante),
    queryFn: () => decodeImage(url ?? ''),
    enabled: enabled && !!url && !!mapId,
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
  })
  return data ?? null
}
