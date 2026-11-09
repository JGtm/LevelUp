/** Libellés réduits de la barre latérale repliée : l'initiale d'un joueur, le sigle d'un jeu. */

/** Première lettre d'un gamertag, en majuscule ; `?` quand il est vide. */
export function playerInitial(gamertag: string | null | undefined): string {
  const first = Array.from((gamertag ?? '').trim())[0]
  return first ? first.toLocaleUpperCase() : '?'
}

/** Sigle d'un nom de jeu : la première lettre de ses deux premiers mots (« Halo Infinite » donne « HI », « Halo 5: Guardians » donne « H5 »). */
export function titleInitials(name: string): string {
  const letters = name
    .split(/\s+/)
    .map((word) => Array.from(word)[0] ?? '')
    .join('')
    .slice(0, 2)
  return letters ? letters.toLocaleUpperCase() : '?'
}
