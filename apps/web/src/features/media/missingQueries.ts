/**
 * Queries TanStack Query — médias dont le fichier est introuvable : liste, fiches candidates,
 * fusion, oubli, confirmation d'une disparition massive, ouverture de l'explorateur.
 *
 * Les chemins échangés sont les URL servables de la liste et des candidats : le serveur les
 * reverse vers la clé de stockage. Toute écriture invalide le préfixe `mediaBase` du joueur ;
 * la confirmation d'une disparition massive attend en revanche la fin de la passe lancée côté
 * serveur, qui fait changer la version du flux (`useFeedVersion`) et rafraîchit la page.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { api, apiErrorMessage } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { getMediaModalsText } from './i18n-modals'
import type {
  MediaConfirmBulkResponse,
  MediaFilePathRequest,
  MediaForgetResponse,
  MediaMergeCandidatesResponse,
  MediaMergeRequest,
  MediaMergeResponse,
  MediaMissingResponse,
  MediaRevealResponse,
} from '@/lib/api/types'

/** Médias introuvables du joueur, avec l'état du dossier des captures. */
export function useMissingMedia(playerSlug: string, enabled = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.mediaMissing(playerSlug, titleSlug),
    queryFn: () => api.get<MediaMissingResponse>(`/players/${playerSlug}/media/missing`),
    enabled: enabled && !!playerSlug,
    staleTime: 30_000,
  })
}

/** Fiches présentes dans lesquelles fondre le média introuvable `filePath`. */
export function useMergeCandidates(playerSlug: string, filePath: string | null) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.mediaMergeCandidates(playerSlug, titleSlug, filePath),
    queryFn: () =>
      api.get<MediaMergeCandidatesResponse>(
        `/players/${playerSlug}/media/missing/candidates?file_path=${encodeURIComponent(filePath ?? '')}`,
      ),
    enabled: !!playerSlug && !!filePath,
    staleTime: 30_000,
  })
}

/** Fond une fiche introuvable dans une fiche présente (likes et rattachement repris). */
export function useMergeMissingMedia(playerSlug: string) {
  const queryClient = useQueryClient()
  const locale = useAppShellStore((s) => s.locale)
  return useMutation({
    mutationFn: (req: MediaMergeRequest) =>
      api.post<MediaMergeResponse>(`/players/${playerSlug}/media/missing/merge`, req),
    onSuccess: () => {
      toast.success(getMediaModalsText(locale).missing.mergeSuccess)
      queryClient.invalidateQueries({ queryKey: queryKeys.mediaBase(playerSlug) })
    },
    onError: (error) => {
      toast.error(getMediaModalsText(locale).missing.mergeError, {
        description: apiErrorMessage(error),
      })
    },
  })
}

/** Oublie une fiche introuvable : l'original n'est jamais touché. */
export function useForgetMissingMedia(playerSlug: string) {
  const queryClient = useQueryClient()
  const locale = useAppShellStore((s) => s.locale)
  return useMutation({
    mutationFn: (req: MediaFilePathRequest) =>
      api.post<MediaForgetResponse>(`/players/${playerSlug}/media/missing/forget`, req),
    onSuccess: () => {
      toast.success(getMediaModalsText(locale).missing.forgetSuccess)
      queryClient.invalidateQueries({ queryKey: queryKeys.mediaBase(playerSlug) })
    },
    onError: (error) => {
      toast.error(getMediaModalsText(locale).missing.forgetError, {
        description: apiErrorMessage(error),
      })
    },
  })
}

/**
 * Confirme la disparition massive en attente. Le serveur rend `confirmed: false` quand rien
 * n'attendait : la liste est alors relue tout de suite. Sinon la passe suivante applique le
 * marquage et fait changer la version du flux, qui rafraîchit la page.
 */
export function useConfirmBulkMissing(playerSlug: string) {
  const queryClient = useQueryClient()
  const locale = useAppShellStore((s) => s.locale)
  return useMutation({
    mutationFn: () =>
      api.post<MediaConfirmBulkResponse>(`/players/${playerSlug}/media/missing/confirm-bulk`),
    onSuccess: (response) => {
      if (!response.confirmed) {
        queryClient.invalidateQueries({ queryKey: queryKeys.mediaBase(playerSlug) })
      }
    },
    onError: (error) => {
      toast.error(getMediaModalsText(locale).missing.bulkConfirmError, {
        description: apiErrorMessage(error),
      })
    },
  })
}

/** Ouvre l'explorateur de fichiers sur le média ; le serveur résout le chemin depuis sa fiche. */
export function useRevealMedia(playerSlug: string) {
  const locale = useAppShellStore((s) => s.locale)
  return useMutation({
    mutationFn: (req: MediaFilePathRequest) =>
      api.post<MediaRevealResponse>(`/players/${playerSlug}/media/reveal`, req),
    onError: (error) => {
      toast.error(getMediaModalsText(locale).missing.revealError, {
        description: apiErrorMessage(error),
      })
    },
  })
}
