/**
 * displayRatingLabel — normalise le libellé de méthode de note pour l'affichage.
 *
 * La famille LUSR porte la MÊME métrique côté utilisateur : 'LUSR' est le slot lu
 * par l'UI et 'LUSR_V2' est une row d'audit (Stratégie C, ADR 0024) qui ne devrait
 * jamais représenter un match dans la vue. La v2 est volontairement TRANSPARENTE pour
 * l'utilisateur → on n'expose jamais le versionnage interne : tout libellé commençant
 * par 'LUSR' s'affiche 'LUSR'. 'CSR' (classé) est conservé tel quel.
 *
 * null / undefined / chaîne vide → null (les appelants affichent leur propre fallback).
 */
export function displayRatingLabel(raw: string | null | undefined): string | null {
  if (!raw) return null
  const upper = raw.toUpperCase()
  return upper.startsWith('LUSR') ? 'LUSR' : upper
}
