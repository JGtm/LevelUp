/**
 * canvasInk.ts — les ENCRES STRUCTURELLES du canvas de rejeu.
 *
 * POURQUOI CE FICHIER EXISTE, ET POURQUOI CE N'EST PAS UNE ENTORSE À LA RÈGLE DES TOKENS.
 * Les tokens sémantiques (`--ac-*`) disent un RÔLE MÉTIER : une victoire, un tier de
 * performance, une équipe. Les traits dont ce canvas a besoin ici n'en portent aucun — l'arête
 * d'une marche et le texte d'une étiquette sont de la mise en page, exactement comme le sont
 * le fond de piste et la bordure d'un SVG, que la règle projet tolère nommément.
 *
 * Ce qu'on refuse en revanche, c'est le littéral : un `#8b97a3` en dur ne suivrait ni le thème
 * clair, ni le thème sombre, ni un changement de charte. On lit donc les variables du système
 * de design lui-même, comme le fait `resolveToken` pour les tokens.
 *
 * Toute couleur qui DIT quelque chose (une équipe, un tir, une mort) passe par un token — ce
 * fichier ne doit jamais servir à en fabriquer une.
 */
import { log } from '@/lib/accessibility/_logger'

/** Variables de mise en page du système de design employées par le canvas. */
export type InkVar = '--muted-foreground' | '--border' | '--foreground' | '--card'

/**
 * readInk lit une variable de mise en page sur :root. Rend une chaîne vide côté serveur ou si
 * la variable manque — un `fillStyle` vide laisse le contexte inchangé plutôt que de peindre
 * une couleur inventée.
 */
export function readInk(name: InkVar): string {
  if (typeof document === 'undefined') return ''
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  if (!value) {
    log.error(`replay-ink:${name}`, `Variable de mise en page "${name}" absente du thème.`)
    return ''
  }
  return value
}
