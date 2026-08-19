/**
 * replayFiesta.ts — LE MATCH EST-IL UNE FIESTA ? La question que le rejeu n'a pas le droit de
 * poser à son document.
 *
 * POURQUOI ELLE SE POSE. Décision produit du 2026-08-18 : hors Fiesta, les objets de puissance
 * lâchés à la mort se dessinent ; EN Fiesta, la règle R2/W4 reste entière — rien. En Fiesta
 * tout le monde ramasse et lâche des armes et de l'équipement en permanence : la carte
 * deviendrait un semis de marques qui ne dit plus rien du terrain (témoin `000d5950`, Super
 * Fiesta : 26 lâchers de puissance sur un seul match).
 *
 * POURQUOI ELLE EST DIFFICILE, ET LA MESURE EST SANS APPEL :
 *
 *  1. LE DOCUMENT DE REJEU NE PORTE AUCUN MODE. `ReplayDocument` n'a ni `mode`, ni `playlist`,
 *     ni `pair_name` ; `Coverage` non plus. Le calque ne peut donc PAS décider seul — d'où
 *     cette garde, qui vit du côté de la PAGE et lui arrive déjà tranchée.
 *  2. LA PLAYLIST NE DIT RIEN. `header.playlist_label` vaut « Quick Play » aussi bien sur le
 *     témoin Super Fiesta que sur le témoin roi de la colline (relevé en base sur les deux).
 *  3. LE LIBELLÉ DE MODE DIT PRESQUE TOUT. `header.mode_ui` est `NormalizeModeLabel(pair_name)`
 *     côté serveur : il CONSERVE l'identité de playlist « Super Fiesta » (428 matchs du corpus
 *     de 1 855) et rend « Castle Wars » (1), mais pour un `pair_name` de la forme
 *     « Fiesta:Slayer on … » il extrait le SOUS-mode (« Slayer », affiché « Assassin ») et
 *     l'indice DISPARAÎT : 3 matchs du corpus.
 *
 * D'OÙ LA RÈGLE, ET ELLE EST CONSERVATRICE PAR CONSTRUCTION : on ne dessine les lâchés QUE si
 * le match ne porte AUCUN indice de Fiesta. Un en-tête absent (réponse en vol, match view en
 * erreur) n'est PAS une preuve de non-Fiesta : il rend `unknown`, et `unknown` ne dessine rien.
 *
 * TROU MESURÉ ET ASSUMÉ : 3 matchs sur les 432 de catégorie Fiesta (0,7 %) — ceux dont le
 * `pair_name` commence par « Fiesta: » — passeront pour non-Fiesta. Le fermer demande de
 * publier la catégorie de mode (ou le `pair_name`) dans l'en-tête de la Match View, côté Go :
 * c'est un report écrit au registre, pas une correction que ce module peut faire.
 *
 * Pas de React, aucun appel réseau : une fonction pure sur ce que la réponse porte déjà.
 */
import type { MatchViewHeader } from '@/lib/api/types'

/**
 * CE QUE LA GARDE PEUT DIRE. Trois valeurs et non un booléen, parce que « je ne sais pas » et
 * « ce n'est pas une Fiesta » ne commandent pas la même chose : seule `clear` dessine.
 */
export type FiestaGuard = 'fiesta' | 'clear' | 'unknown'

/**
 * LES LIBELLÉS QUI TRAHISSENT UNE FIESTA, tels que le serveur les publie.
 *
 * Ce sont les préfixes de `pair_name` que le Go range dans la catégorie Fiesta
 * (`modePrefixToCategory`, `internal/games/halo_infinite/mode_category.go`) : `Fiesta`,
 * `Super Fiesta` et `Castle Wars`. Ils sont IDENTIQUES en français et en anglais
 * (`config/titles/halo_infinite/mappings/assets.toml`) — la comparaison ne dépend donc pas de
 * la langue de l'utilisateur, et « Fiesta » attrape « Super Fiesta » par inclusion.
 *
 * HUSKY RAID N'Y EST PAS, et c'est une décision : le Go le PROMEUT en catégorie propre, et ce
 * n'est pas un mode à équipement aléatoire — un Husky Raid se lit comme une capture de
 * drapeau, ses lâchers ont le même sens qu'ailleurs.
 */
export const FIESTA_TOKENS: readonly string[] = ['fiesta', 'castle wars']

/** Un libellé porte-t-il un indice de Fiesta ? Comparaison insensible à la casse. */
function looksFiesta(label: string | undefined): boolean {
  if (!label) return false
  const low = label.toLowerCase()
  return FIESTA_TOKENS.some((token) => low.includes(token))
}

/**
 * matchFiestaGuard — ce que l'en-tête de la Match View permet d'affirmer sur le mode.
 *
 * `undefined` (réponse pas encore là, ou match view en erreur) rend `unknown` : l'appelant ne
 * dessine alors rien. C'est le sens de la règle — l'absence d'indice n'est pas une preuve.
 *
 * Les DEUX champs sont lus. `mode_ui` est celui qui parle ; `playlist_label` ne dit rien sur le
 * corpus mesuré, mais une playlist explicitement nommée « Fiesta » sur un autre compte ou un
 * autre titre serait un indice qu'il aurait été absurde d'ignorer pour économiser une ligne.
 */
export function matchFiestaGuard(header: MatchViewHeader | undefined): FiestaGuard {
  if (!header) return 'unknown'
  return looksFiesta(header.mode_ui) || looksFiesta(header.playlist_label) ? 'fiesta' : 'clear'
}
