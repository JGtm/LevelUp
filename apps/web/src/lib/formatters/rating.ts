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

/**
 * formatRankDelta — formate un delta de rang signé pour l'affichage.
 *
 * CSR = entier ("+45" / "-12"), LUSR = 2 décimales ("+1.23"). Le delta LUSR est
 * désormais continu (μ remappé, cf. backend skill_v2_canonical), donc non nul à
 * presque chaque match : les 2 décimales évitent qu'un gain réel de +0.3 ne
 * s'affiche "+0". Delta nul → "±0" (CSR) / "±0.00" (LUSR), jamais "-0".
 */
export function formatRankDelta(delta: number, type: string): string {
  const isCsr = type.toLowerCase() === 'csr'
  if (delta === 0) return isCsr ? '±0' : '±0.00'
  const abs = Math.abs(delta)
  const formatted = isCsr ? String(Math.round(abs)) : abs.toFixed(2)
  return delta > 0 ? `+${formatted}` : `-${formatted}`
}
