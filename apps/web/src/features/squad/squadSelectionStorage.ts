/**
 * squadSelectionStorage — la composition choisie sur la page Escouade, persistée par joueur en
 * localStorage (`squad-teammates-{playerSlug}`, tableau JSON de gamertags). Les pages qui
 * partagent cette sélection passent par ces deux fonctions : la clé n'existe qu'ici.
 */

/** Clé de stockage de la composition d'un joueur. */
function teammatesKey(playerSlug: string): string {
  return `squad-teammates-${playerSlug}`
}

/** Coéquipiers stockés pour ce joueur ; liste vide si rien n'est lisible. */
export function readStoredTeammates(playerSlug: string): string[] {
  try {
    const stored = localStorage.getItem(teammatesKey(playerSlug))
    return stored ? (JSON.parse(stored) as string[]) : []
  } catch {
    return []
  }
}

/** Écrit les coéquipiers de ce joueur ; un stockage indisponible n'interrompt rien. */
export function writeStoredTeammates(playerSlug: string, value: string[]): void {
  try {
    localStorage.setItem(teammatesKey(playerSlug), JSON.stringify(value))
  } catch {
    /* stockage indisponible (navigation privée, quota) : la sélection vit alors en mémoire */
  }
}
