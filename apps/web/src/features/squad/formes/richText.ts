/**
 * richText.ts — LE DÉCOUPAGE des passages en gras des textes du bloc.
 *
 * Séparé du composant qui les rend (`forms/RichText.tsx`) : un fichier qui
 * exporte un composant ET une fonction casse le rafraîchissement à chaud de
 * Vite (règle `react-refresh/only-export-components`).
 */

/** Découpe un texte sur ses paires d'astérisques. Index impair = gras. */
export function splitEmphasis(text: string): string[] {
  return text.split('**')
}
