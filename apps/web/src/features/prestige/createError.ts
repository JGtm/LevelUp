import type { getPrestigeText } from './i18n'

/**
 * Message localisé d'un refus de création (arc ou objectif). Le serveur ne renvoie qu'un code
 * machine et un message fixe en anglais : l'écran n'affiche jamais ce texte, un code inconnu
 * reçoit le message générique. `null` sans erreur.
 */
export function createErrorMessage(
  error: unknown,
  t: ReturnType<typeof getPrestigeText>,
): string | null {
  if (error == null) return null
  const code = typeof error === 'object' && 'code' in error ? (error as { code?: unknown }).code : undefined
  switch (code) {
    case 'cooldown_active':
      return t.cooldownErrorMessage
    case 'challenge_too_easy':
      return t.tooEasyErrorMessage
    case 'invalid_input':
      return t.invalidInputErrorMessage
    default:
      return t.createErrorGeneric
  }
}
