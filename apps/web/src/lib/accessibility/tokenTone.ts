/**
 * tokenTone.ts — Ton dérivé d'une couleur de jeton : même teinte, clarté décalée vers le
 * premier plan du thème (plus clair sur fond sombre, plus foncé sur fond clair) et chroma
 * relevée d'autant pour que le ton reste vif au lieu de virer au gris.
 *
 * Pour quoi : graduer une échelle À L'INTÉRIEUR d'un rôle (les trois tranches de part de
 * dégâts d'une assistance, par exemple) sans emprunter une autre teinte — la famille des
 * stats de combat interdit l'emprunt (skill color-tokens) — et sans passer par l'opacité,
 * qui délave le ton sur sa piste au lieu de l'éclaircir. Beaucoup de jetons ne peuvent pas
 * être choisis plus vifs : ils sont retenus par le contraste 3:1 sur fond clair ET par la
 * séparation daltonisme de leur famille (`combatStatTokens.test.ts`) ; la variation se
 * dérive donc du jeton plutôt que de s'écrire dans la palette.
 *
 * Même doctrine que `hexComplement` : la dérivation vit ici, dans le système de jetons, et
 * les features n'écrivent aucune couleur. Elle travaille sur la valeur CSS du jeton
 * (`tokenCssVar(...)` → `var(--color-…)`), donc elle suit la palette active et le thème
 * sans recalcul : c'est le navigateur qui résout `oklch(from …)` et `light-dark(...)`.
 *
 * Contrat : `step` en unités de clarté OKLCH (0..1). 0 rend la couleur inchangée ;
 * au-delà de ~0.3 le ton décroche de la couleur source et ne se lit plus comme le même rôle.
 */

/** Décalage maximal admis : au-delà, le ton ne se lit plus comme le même rôle. */
const MAX_STEP = 0.3

export function tokenTone(color: string, step: number): string {
  if (!(step > 0)) return color
  const s = Math.min(step, MAX_STEP)
  const chroma = `calc(c * ${1 + s})`
  const darker = `oklch(from ${color} calc(l - ${s}) ${chroma} h)` // color-allow: dérivation relative d'un jeton (aucune couleur littérale) — c'est ici, dans le système de jetons, que la couleur se calcule
  const lighter = `oklch(from ${color} calc(l + ${s}) ${chroma} h)` // color-allow: idem, le pendant clair du même jeton
  return `light-dark(${darker}, ${lighter})`
}
