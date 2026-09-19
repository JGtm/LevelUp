/**
 * assistTierTone — le ton d'UNE tranche d'assistance : trois clartés OKLCH de la couleur
 * du sens (`assist-received` / `assist-given`), PAS trois opacités. L'opacité fondait les
 * tons faibles dans la piste (35 % de jaune-700 sur gris sombre = boue) et le jeton
 * lui-même ne peut pas s'éclaircir — il est retenu sombre par le contraste 3:1 sur fond
 * clair et l'écart daltonisme (combatStatTokens.test.ts : toute variante plus vive de
 * `assist-received` casse la séparation deutéranopie, mesuré le 2026-09-19).
 *
 * Le ton monte vers le premier plan du thème (`light-dark` : plus clair en sombre, plus
 * foncé en clair) avec la chroma relevée pour rester vif ; `low` = le jeton tel quel,
 * `high` = le ton le plus marqué. Jamais une autre teinte : `h` reste celui du jeton
 * (skill color-tokens, famille des stats de combat). Remplace les opacités 35 / 65 / 100 %
 * le 2026-09-19.
 */
import type { AssistTier } from './assistsI18n'

const TIER_STEP: Record<AssistTier, number> = { low: 0, mid: 0.12, high: 0.24 }

export function assistTierTone(color: string, tier: AssistTier): string {
  const step = TIER_STEP[tier]
  if (step === 0) return color
  const chroma = `calc(c * ${1 + step})`
  const darker = `oklch(from ${color} calc(l - ${step}) ${chroma} h)`
  const lighter = `oklch(from ${color} calc(l + ${step}) ${chroma} h)`
  return `light-dark(${darker}, ${lighter})`
}
