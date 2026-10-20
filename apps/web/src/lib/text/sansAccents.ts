/**
 * sansAccents — le texte sans ses accents : décomposition NFD, marques diacritiques combinantes
 * (U+0300 à U+036F) retirées ; casse et blancs inchangés. LA normalisation des recherches et
 * comparaisons insensibles aux accents de l'application (« Écluse » → « Ecluse »).
 *
 * Garde-rail : `sansAccents.guard.test.ts` — `.normalize('NFD')` n'apparaît nulle part ailleurs.
 */
export function sansAccents(texte: string): string {
  return texte.normalize('NFD').replace(/[\u0300-\u036f]/g, '')
}
