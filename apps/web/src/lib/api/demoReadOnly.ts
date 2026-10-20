/**
 * Refus « démo en lecture seule » côté web.
 *
 * En mode démo, le serveur refuse toute écriture par `403 demo_mode_forbidden`
 * (middleware Go `DemoReadOnly`, contrat documenté dans `api/openapi.yaml`). Ce module
 * en fait UN message lisible, dans la langue de l'interface, au lieu d'un texte
 * technique anglais. `handleDemoRefusal` est appelé par le cache des mutations
 * (`app/queryClient.ts`) AVANT les `onError` propres à chaque mutation :
 *   - il remplace le message de l'erreur (les `onError` affichent alors le bon texte) ;
 *   - il affiche un seul toast (identifiant fixe : plusieurs refus rapprochés ne
 *     s'empilent pas) quand la mutation n'a pas sa propre gestion d'erreur.
 */
import { toast } from 'sonner'
import { apiErrorCode } from '@/lib/api/client'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'

/** Code d'erreur du refus démo (source Go : `middleware.DemoModeForbiddenCode`). */
const DEMO_FORBIDDEN_CODE = 'demo_mode_forbidden'

/** Identifiant du toast : un seul affiché à la fois. */
export const DEMO_TOAST_ID = 'demo-read-only'

export function isDemoRefusal(err: unknown): boolean {
  return apiErrorCode(err) === DEMO_FORBIDDEN_CODE
}

/** Message du refus dans la langue de l'interface. */
function demoRefusalMessage(locale: Locale): string {
  return formatMessage(commonManifest, 'common.shell.demo_read_only', locale)
}

/**
 * Traite un refus démo : message localisé posé sur l'erreur, toast si la mutation n'a pas
 * de gestion d'erreur propre. Rend `true` si l'erreur était un refus démo.
 */
export function handleDemoRefusal(err: unknown, hasOwnErrorHandler: boolean, locale: Locale): boolean {
  if (!isDemoRefusal(err)) return false
  const message = demoRefusalMessage(locale)
  ;(err as { message?: string }).message = message
  if (!hasOwnErrorHandler) toast.info(message, { id: DEMO_TOAST_ID })
  return true
}
