/**
 * useKillSourceIcons — le référentiel des pictogrammes d'arme du titre, par weapon_key.
 *
 * GET /assets/{titre}/kill-sources. Une réponse par titre et par langue (le nom de l'arme est résolu
 * dans la langue de la requête : la langue est le dernier segment de la clé). C'est un référentiel
 * versionné avec l'application : il ne bouge pas pendant la session, d'où l'`staleTime` long.
 *
 * CHARGEMENT ET ERREUR NE BLOQUENT RIEN : tant que la réponse n'est pas là, ou si elle échoue, le
 * hook rend le référentiel vide — les lignes qui s'en servent s'affichent, sans pictogramme.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { KillSourceIconEntry } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { queryKeys } from '@/lib/query/keys'

import { NO_WEAPON_ICONS, weaponIconsOf, type WeaponIconsByKey } from '../model/killSourceIcons'

/** Une heure : un référentiel de titre, jamais modifié pendant la vie de la page. */
const STALE_MS = 60 * 60 * 1000

export function useKillSourceIcons(titleSlug: string, locale: Locale): WeaponIconsByKey {
  const { data } = useQuery({
    queryKey: queryKeys.killSourceIcons(titleSlug, locale),
    queryFn: () => api.get<KillSourceIconEntry[] | null>(`/assets/${titleSlug}/kill-sources`),
    enabled: !!titleSlug,
    staleTime: STALE_MS,
    // Référence stable : TanStack Query ne recalcule la table que quand la réponse change.
    select: weaponIconsOf,
  })
  return data ?? NO_WEAPON_ICONS
}
