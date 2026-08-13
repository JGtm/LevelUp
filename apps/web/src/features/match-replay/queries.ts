/**
 * queries.ts — lecture de l'artefact de rejeu 2D d'un match, et de son fond de carte.
 * GET /players/{slug}/matches/{matchId}/replay → ReplayDocument (404 si pas d'artefact).
 *
 * Le document est NORMALISÉ ici, à l'entrée : c'est la seule frontière entre la
 * nullabilité du transport et le rendu (cf. replayNormalize.ts). Ce qui est mis en cache
 * l'est déjà normalisé — la conversion ne se rejoue pas à chaque rendu.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { ReplayDocument, ReplayMapBackground, ReplayMapCallouts } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

import { normalizeReplayDocument } from './replayNormalize'

export function useMatchReplay(playerSlug: string, matchId: string) {
  // Le titre courant entre dans la clé comme pour la vue match : deux titres ne
  // partagent jamais un artefact, et changer de titre ne doit pas servir le cache
  // de l'autre.
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchReplay(playerSlug, titleSlug, matchId),
    queryFn: async () =>
      normalizeReplayDocument(
        await api.get<ReplayDocument>(`/players/${playerSlug}/matches/${matchId}/replay`),
      ),
    staleTime: 5 * 60_000,
    enabled: !!playerSlug && !!matchId,
    // 404 = pas d'artefact de rejeu pour ce match → état vide, ne pas réessayer.
    retry: false,
  })
}

/**
 * useReplayMapBackground charge le CALAGE du fond de carte du match.
 *
 * 404 = la carte n'a pas d'image figée (seules 21 en ont) : ce n'est pas une panne, c'est
 * le cas nominal pour les autres cartes, et le rejeu retombe sur son sol reconstruit. On ne
 * réessaie donc pas, et on ne remonte pas d'erreur à l'écran.
 */
export function useReplayMapBackground(playerSlug: string, matchId: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchReplayBackground(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<ReplayMapBackground>(
        `/players/${playerSlug}/matches/${matchId}/replay/background`,
      ),
    // La donnée est de RÉFÉRENCE (versionnée, re-cuite hors ligne) : rien ne la périme
    // pendant une session.
    staleTime: Infinity,
    enabled: !!playerSlug && !!matchId,
    retry: false,
  })
}

/**
 * useReplayMapImage charge l'IMAGE du fond et rend un `HTMLImageElement` prêt à dessiner.
 *
 * POURQUOI PAS UNE URL POSÉE DANS UN `<img>`. Le dessin se fait au canvas : il lui faut un
 * élément DÉJÀ décodé, sans quoi le premier `drawImage` ne peint rien et le fond
 * n'apparaît qu'au redimensionnement suivant. Le blob passe en plus par le client d'API,
 * qui emporte l'en-tête de titre qu'une balise `img` ne peut pas poser.
 *
 * `enabled` est piloté par l'appelant : sans calage, l'image ne sert à rien — la charger
 * serait 700 Kio pour un fond qu'on ne saurait pas placer.
 */
export function useReplayMapImage(
  playerSlug: string,
  matchId: string,
  enabled: boolean,
): HTMLImageElement | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.matchReplayBackgroundImage(playerSlug, titleSlug, matchId),
    queryFn: async () => {
      const blob = await api.getBlob(
        `/players/${playerSlug}/matches/${matchId}/replay/background.png`,
      )
      const url = URL.createObjectURL(blob)
      try {
        return await decodeImage(url)
      } finally {
        // L'URL d'objet ne sert qu'au décodage : une fois l'image chargée, les pixels
        // appartiennent à l'élément. La garder ouverte retiendrait le blob jusqu'à la
        // fermeture de l'onglet.
        URL.revokeObjectURL(url)
      }
    },
    staleTime: Infinity,
    enabled: enabled && !!playerSlug && !!matchId,
    // Absence ou échec de chargement : pas de fond, le rejeu garde son sol reconstruit.
    // Rien à dire à l'utilisateur — la carte s'affiche quand même.
    retry: false,
  })
  return data ?? null
}

/**
 * useReplayMapCallouts charge les ZONES NOMMÉES de la carte du match.
 *
 * 404 = la carte n'en a pas : cas NOMINAL des cartes Forge (leur canevas n'en porte
 * aucune — par construction, pas par lacune). Pas de retry, pas d'erreur à l'écran : le
 * calque zones ne s'affiche simplement pas.
 */
export function useReplayMapCallouts(playerSlug: string, matchId: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchReplayCallouts(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<ReplayMapCallouts>(
        `/players/${playerSlug}/matches/${matchId}/replay/callouts`,
      ),
    // Donnée de RÉFÉRENCE versionnée : rien ne la périme pendant une session.
    staleTime: Infinity,
    enabled: !!playerSlug && !!matchId,
    retry: false,
  })
}

/** decodeImage rend l'élément une fois l'image RÉELLEMENT décodée (cf. useReplayMapImage). */
function decodeImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('image de fond illisible'))
    img.src = url
  })
}
