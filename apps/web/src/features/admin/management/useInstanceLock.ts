/**
 * useInstanceLock — état et bascule du verrou « instance fermée » (ADR 0035 D5).
 *
 * Le backend accepte `PATCH /settings {instance_locked}` sous rôle admin depuis
 * que le verrou est centralisé, et `/bootstrap` sert son état — mais AUCUNE page
 * ne l'exposait : la production a dû être verrouillée à la main dans le fichier
 * de réglages. Ce hook est le seul endroit qui porte cette logique ; la section
 * ne fait que la rendre.
 *
 * L'état lu vient du store appShell, alimenté par `/bootstrap` : après un
 * changement réussi, on invalide `bootstrap` pour que l'affichage suive la
 * vérité serveur au lieu d'un optimisme local.
 */
import { useQueryClient } from '@tanstack/react-query'

import { apiErrorCode } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useUpdateSettings } from '@/features/settings/queries'
import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Code serveur quand le verrou est forcé par l'environnement
 * (LEVELUP_INSTANCE_LOCKED) : le fichier ne peut pas l'ouvrir, le PATCH est
 * refusé en 409 au lieu de réussir sur disque et de laisser l'interrupteur se
 * recocher seul (revue adversariale du 2026-09-16).
 */
export const LOCK_FORCED_CODE = 'instance_lock_forced'

export interface InstanceLock {
  /** Instance fermée ? (source : /bootstrap via le store appShell) */
  locked: boolean
  /** Envoi en cours — l'interrupteur se grise, pas de double envoi. */
  isPending: boolean
  /** Le dernier changement a échoué (403 non-admin, 422 en démo…). */
  isError: boolean
  /** Code machine du dernier échec (ex. LOCK_FORCED_CODE), null sinon. */
  errorCode: string | null
  setLocked: (next: boolean) => void
}

export function useInstanceLock(): InstanceLock {
  const locked = useAppShellStore((s) => s.instanceLocked)
  const queryClient = useQueryClient()
  const update = useUpdateSettings()

  return {
    locked,
    isPending: update.isPending,
    isError: update.isError,
    errorCode: update.isError ? (apiErrorCode(update.error) ?? null) : null,
    setLocked: (next: boolean) => {
      update.mutate(
        { instance_locked: next },
        {
          onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap })
          },
        },
      )
    },
  }
}
