/**
 * scoreTimelineKind.ts — LES TROIS LECTURES DU BLOC « Score dans le temps », côté client.
 *
 * Miroir exact des constantes Go (`games/mappings/loader_regulation.go`) : le serveur sert
 * la valeur dans `header.score_timeline_kind`, résolue depuis la DONNÉE du titre
 * (`regulation.toml [score_timeline]`, par jeton de mode cherché dans le `pair_name` BRUT).
 * Le client ne décide rien — il lit. Ces littéraux sont écrits UNE fois ici plutôt que semés
 * dans trois composants : un `'hidden'` mal orthographié dans l'un d'eux ferait réapparaître
 * un bloc sans que rien ne le signale.
 *
 * LE TROISIÈME CAS NE S'ÉCRIT PAS. Le serveur n'envoie QUE `hidden` et `events` : `curve`
 * étant le défaut d'ici, le servir explicitement serait redire le défaut sur chaque match.
 * Champ vide, titre sans table, valeur inconnue -> la COURBE, c'est-à-dire le comportement
 * d'avant le 2026-09-03.
 */

/** Le mode marque au FRAG : le bloc ne s'affiche pas (« Frags cumulés » le dit déjà). */
export const SCORE_TIMELINE_HIDDEN = 'hidden'
/** Le mode marque en 3 à 5 points : des barres verticales aux instants de marque. */
export const SCORE_TIMELINE_EVENTS = 'events'
