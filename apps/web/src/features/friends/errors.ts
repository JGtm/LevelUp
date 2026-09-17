/**
 * Messages d'erreur de la liste d'amis, FR + EN.
 *
 * L'API ne renvoie qu'un CODE machine pour ces erreurs (décision D6 du plan
 * « libellés en dur » : un texte lisible se traduit, donc il vit ici). Un code
 * inconnu retombe sur un message générique plutôt que sur une chaîne vide.
 */
import type { Locale } from '@/lib/i18n/locale'
import { apiErrorCode } from '@/lib/api/client'

type FriendsErrorKey =
  | 'friends_forbidden'
  | 'invalid_body'
  | 'invalid_friends_too_many'
  | 'invalid_friends_gamertag_too_long'
  | 'friends_load_error'
  | 'friends_save_error'
  | 'player_not_found'
  | 'player_without_xuid'
  | 'player_forbidden'
  | 'unknown'

const FRIENDS_ERROR_TEXT: Record<Locale, Record<FriendsErrorKey, string>> = {
  fr: {
    friends_forbidden: "Seul le propriétaire de ce profil (ou un administrateur) peut modifier sa liste d'amis.",
    invalid_body: 'Requête invalide.',
    invalid_friends_too_many: "Trop d'amis dans la liste (50 au maximum).",
    invalid_friends_gamertag_too_long: 'Un gamertag dépasse 50 caractères.',
    friends_load_error: "Impossible de charger la liste d'amis.",
    friends_save_error: "Impossible d'enregistrer la liste d'amis.",
    player_not_found: 'Joueur introuvable.',
    player_without_xuid: "Ce profil n'a pas de XUID : impossible de lui attacher une liste d'amis.",
    player_forbidden: "Ce profil ne t'appartient pas.",
    unknown: "Impossible de mettre à jour la liste d'amis.",
  },
  en: {
    friends_forbidden: "Only this profile's owner (or an admin) can change its friends list.",
    invalid_body: 'Invalid request.',
    invalid_friends_too_many: 'Too many friends in the list (50 maximum).',
    invalid_friends_gamertag_too_long: 'A gamertag is longer than 50 characters.',
    friends_load_error: 'Unable to load the friends list.',
    friends_save_error: 'Unable to save the friends list.',
    player_not_found: 'Player not found.',
    player_without_xuid: 'This profile has no XUID: no friends list can be attached to it.',
    player_forbidden: 'This profile is not yours.',
    unknown: 'Unable to update the friends list.',
  },
}

/** Traduit une erreur d'API de la liste d'amis. */
export function friendsErrorMessage(err: unknown, locale: string): string {
  const table = FRIENDS_ERROR_TEXT[locale === 'en' ? 'en' : 'fr']
  const code = apiErrorCode(err)
  if (code && code in table) {
    return table[code as FriendsErrorKey]
  }
  return table.unknown
}
