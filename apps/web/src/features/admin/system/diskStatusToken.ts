import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

/** Token d'accent du statut disque (mêmes valeurs que la fraîcheur). */
export function diskToken(status: string): SemanticToken | undefined {
  switch (status) {
    case 'ok':
      return 'success'
    case 'warn':
      return 'warning'
    case 'critical':
      return 'destructive'
    default:
      return undefined
  }
}
