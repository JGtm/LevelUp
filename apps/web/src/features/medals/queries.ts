/**
 * Queries TanStack Query — page Médailles (sous-page Carrière, Lot A2).
 *
 * POST /players/{slug}/pages/medals → catalogue complet du titre + compteur
 * obtenu par joueur. Corps vide : le filtre (obtenues/non-obtenues/toutes) et le
 * tri sont appliqués côté CLIENT (données bornées ~160-215 médailles). La locale
 * scope la clé car le backend localise noms/descriptions (X-LevelUp-Locale).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MedalsPageResponse } from '@/lib/api/types'

export function useMedalsPage(playerSlug: string) {
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.medals(playerSlug, locale),
    queryFn: () => api.post<MedalsPageResponse>(`/players/${playerSlug}/pages/medals`, {}),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
