/**
 * reliefQueries — LES DEUX FICHIERS DU RELIEF d'un fond de carte, chargés par le client d'API.
 *
 *   - les DONNÉES (`…/background.relief.json`, JSON en clair, 50 à 126 Ko compressés) : étages, sols du
 *     maillage, traits ; lues avec le calage, elles décident déjà de ce que les commandes proposent ;
 *   - l'IMAGE des hauteurs (`…/background.png?variante=relief`), plus lourde : demandée seulement quand
 *     une commande s'en sert (`needImage`) — fond Relief, traits du relief, coupe par étage.
 *
 * Les deux sont propres à la CARTE et figés entre deux cuissons (`staleTime` infini). Aucune langue
 * dans les clés : le relief ne porte aucun libellé. La lecture des données passe par `parseReliefData`
 * (version de schéma et forme vérifiées) : une `ReliefError` est une erreur de requête, que
 * l'appelant journalise avant de replier sur le plan. L'image reste un `Blob` ; son décodage et son
 * contrôle (`readReliefImage`) se font au moment du chargement dans le calculateur, parce que ses
 * tableaux sont CÉDÉS au worker et ne se partagent pas entre deux chargements.
 *
 * `useReliefFiles` est la lecture commune des vues 2D ; chacune fournit ses clés et ses adresses (`useMatchReliefSource`
 * pour le fond d'un match : le rejeu et « Occupation du terrain »).
 */
import { useQuery, type QueryKey } from '@tanstack/react-query'
import { useMemo } from 'react'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

import { backgroundImagePath, reliefDataPath } from './mapBackgroundVariant'
import { parseReliefData } from './relief/decodeData'
import type { ReliefData } from './relief/types'

/** Où et sous quelles clés lire le relief d'une carte. */
export interface ReliefFilesSource {
  dataKey: QueryKey
  dataPath: string
  imageKey: QueryKey
  imagePath: string
}

/** Ce que la lecture des deux fichiers a rendu à cet instant. */
export interface ReliefFiles {
  data: ReliefData | null
  blob: Blob | null
  /** Un fichier attendu est encore en vol. */
  loading: boolean
  /** La première erreur de lecture (`ReliefError`, `ApiError`…), ou `null`. */
  error: unknown
}

const NO_FILES: ReliefFiles = { data: null, blob: null, loading: false, error: null }

/**
 * useReliefFiles lit les données du relief dès que `source` est fournie, et son image quand
 * `needImage`. `source = null` : le fond n'a pas de relief, rien n'est demandé.
 */
export function useReliefFiles(source: ReliefFilesSource | null, needImage: boolean): ReliefFiles {
  const data = useQuery({
    queryKey: source?.dataKey ?? [],
    queryFn: async () => parseReliefData(await api.get<unknown>(source?.dataPath ?? '')),
    enabled: source !== null,
    staleTime: Infinity,
    // Absence, version inconnue ou forme invalide : pas de relief, le plan le remplace. Pas de nouvel essai.
    retry: false,
  })
  const image = useQuery({
    queryKey: source?.imageKey ?? [],
    queryFn: () => api.getBlob(source?.imagePath ?? ''),
    enabled: source !== null && needImage && data.isSuccess,
    staleTime: Infinity,
    retry: false,
  })
  if (source === null) return NO_FILES
  return {
    data: data.data ?? null,
    blob: image.data ?? null,
    loading: data.isPending || (needImage && data.isSuccess && image.isPending),
    error: data.error ?? image.error ?? null,
  }
}

/**
 * useMatchReliefSource : où lire le relief du fond d'un MATCH (le rejeu, « Occupation du terrain » : un relief est propre
 * à la carte du match, les clés sont celles du rejeu). `offered` est ce que le sidecar annonce (`reliefData` et
 * `reliefImage` présents) ; sans lui, ou sans joueur ni match, `null` : aucune requête.
 */
export function useMatchReliefSource(playerSlug: string, matchId: string, offered: boolean): ReliefFilesSource | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useMemo(() => {
    if (!offered || !playerSlug || !matchId) return null
    const imageBase = `/players/${playerSlug}/matches/${matchId}/replay/background.png`
    return {
      dataKey: queryKeys.matchReplayReliefData(playerSlug, titleSlug, matchId),
      dataPath: reliefDataPath(imageBase),
      imageKey: queryKeys.matchReplayReliefImage(playerSlug, titleSlug, matchId),
      imagePath: backgroundImagePath(imageBase, 'relief'),
    }
  }, [offered, playerSlug, matchId, titleSlug])
}
