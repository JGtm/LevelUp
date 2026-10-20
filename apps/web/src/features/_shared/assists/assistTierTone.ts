/**
 * assistTierTone — le ton d'UNE tranche d'assistance : trois clartés de la couleur du sens
 * (`assist-received` / `assist-given`), PAS trois opacités. L'opacité fondait les tons
 * faibles dans la piste (35 % de jaune-700 sur gris sombre = boue) et le jeton lui-même ne
 * peut pas s'éclaircir — il est retenu sombre par le contraste 3:1 sur fond clair et
 * l'écart daltonisme (combatStatTokens.test.ts : toute variante plus vive de
 * `assist-received` casse la séparation deutéranopie, mesuré le 2026-09-19).
 *
 * Le calcul du ton appartient au système de jetons (`lib/accessibility/tokenTone.ts`) :
 * ici ne vit que la correspondance tranche → écart, qui est du domaine des assistances.
 * `low` = le jeton tel quel, `high` = le ton le plus marqué ; jamais une autre teinte
 * (skill color-tokens, famille des stats de combat). Remplace les opacités 35 / 65 / 100 %
 * le 2026-09-19.
 */
import { tokenTone } from '@/lib/accessibility'

import type { AssistTier } from './assistsI18n'

const TIER_STEP: Record<AssistTier, number> = { low: 0, mid: 0.12, high: 0.24 }

export function assistTierTone(color: string, tier: AssistTier): string {
  return tokenTone(color, TIER_STEP[tier])
}
